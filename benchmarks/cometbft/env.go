package cometbft

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/zeu5/dist-rl-testing/core"
	"github.com/zeu5/dist-rl-testing/util"
)

type CometEnvConstructor struct {
	cfg *CometClusterConfig
}

var _ core.PEnvironmentConstructor = &CometEnvConstructor{}

func NewCometEnvConstructor(clusterConfig *CometClusterConfig) *CometEnvConstructor {
	return &CometEnvConstructor{
		cfg: clusterConfig,
	}
}

func (c *CometEnvConstructor) NewPEnvironment(ctx context.Context, instance int) core.PEnvironment {
	cfg := c.cfg.Copy()
	cfg.BaseInterceptListenPort = cfg.BaseInterceptListenPort + 10*instance
	cfg.BaseP2PPort = cfg.BaseP2PPort + 10*instance
	cfg.BaseRPCPort = cfg.BaseRPCPort + 10*instance
	cfg.InterceptServerPort = cfg.InterceptServerPort + instance
	cfg.BaseWorkingDir = path.Join(cfg.BaseWorkingDir, strconv.Itoa(instance))

	return NewCometEnv(ctx, cfg)
}

type CometRequest struct {
	// One of "get|set"
	Type string
	// Key of the operation
	Key string
	// Value of the operation in the case of set
	Value string
}

func (c CometRequest) Copy() CometRequest {
	return CometRequest{
		Type:  c.Type,
		Key:   c.Key,
		Value: c.Value,
	}
}

type CometClusterState struct {
	NodeStates      map[int]*CometNodeState
	PendingMessages map[string]Message
	PendingRequests []CometRequest
}

func (r *CometClusterState) NodeState(id int) core.NState {
	s := r.NodeStates[id]
	return s
}

func (r *CometClusterState) Messages() []core.Message {
	out := make([]core.Message, len(r.PendingMessages))
	i := 0
	for _, m := range r.PendingMessages {
		out[i] = m.Copy()
		i++
	}
	return out
}

func (r *CometClusterState) CanDeliverRequest() bool {
	return len(r.PendingRequests) > 0
}

func (r *CometClusterState) Requests() []core.Request {
	out := make([]core.Request, len(r.PendingRequests))
	for i, r := range r.PendingRequests {
		out[i] = r.Copy()
	}
	return out
}

func (r *CometClusterState) Copy() *CometClusterState {
	out := &CometClusterState{
		NodeStates:      make(map[int]*CometNodeState),
		PendingMessages: make(map[string]Message),
		PendingRequests: make([]CometRequest, len(r.PendingRequests)),
	}
	for k, v := range r.NodeStates {
		out.NodeStates[k] = v.Copy()
	}
	for k, v := range r.PendingMessages {
		out.PendingMessages[k] = v.Copy()
	}
	for i, req := range r.PendingRequests {
		out.PendingRequests[i] = req.Copy()
	}
	return out
}

var _ core.PState = &CometClusterState{}

type CometEnv struct {
	cluster *CometCluster
	network *InterceptNetwork
	cfg     *CometClusterConfig
	ctx     context.Context

	curState *CometClusterState
}

var _ core.PEnvironment = &CometEnv{}

func (r *CometEnv) BecomeByzantine(nodeID uint64) {
	r.network.MakeNodeByzantine(nodeID)
}

func (r *CometEnv) Cleanup() {
	if r.cluster != nil {
		r.cluster.Destroy()
		r.cluster = nil
	}
}

func (r *CometEnv) Reset(_ *core.EpisodeContext) (core.PState, error) {
	if r.cluster != nil {
		if err := r.cluster.Destroy(); err != nil {
			return nil, util.NewLogError(err, r.cluster.GetLogs())
		}
	}
	r.network.Reset()
	cluster, err := NewCometCluster(r.cfg)
	if err != nil {
		return nil, err
	}
	r.cluster = cluster
	if err := r.cluster.Start(); err != nil {
		r.cluster.Destroy()
		return nil, util.NewLogError(err, r.cluster.GetLogs())
	}

	r.network.WaitForNodes(r.cfg.NumNodes)

	newState := &CometClusterState{
		NodeStates:      r.cluster.GetNodeStates(),
		PendingMessages: r.network.GetAllMessages(),
		PendingRequests: make([]CometRequest, r.cfg.NumRequests),
	}
	for i := 0; i < r.cfg.NumRequests; i++ {
		if rand.Intn(2) == 0 {
			newState.PendingRequests[i] = CometRequest{
				Type: "get",
				Key:  "k",
			}
		} else {
			newState.PendingRequests[i] = CometRequest{
				Type:  "set",
				Key:   "k",
				Value: strconv.Itoa(i + 1),
			}
		}
	}
	r.curState = newState
	return newState, nil
}

func (r *CometEnv) Tick(ctx *core.StepContext) (core.PState, error) {
	select {
	case <-time.After(r.cfg.TickDuration):
	case <-ctx.Context.Done():
	}
	newState := &CometClusterState{
		NodeStates:      r.cluster.GetNodeStates(),
		PendingMessages: r.network.GetAllMessages(),
		PendingRequests: copyRequests(r.curState.PendingRequests),
	}
	r.curState = newState
	return newState, nil
}

func (r *CometEnv) DeliverMessages(messages []core.Message, epCtx *core.StepContext) (core.PState, error) {
	newState := r.curState.Copy()

	errs := []string{}
	for _, m := range messages {
		rm, ok := m.(Message)
		if !ok {
			continue
		}
		if err := r.network.SendMessage(epCtx.Context, rm.ID); err != nil {
			errs = append(errs, err.Error())
		}
	}
	// too many errors delivering messages
	if len(errs) > 0 && len(errs) >= len(messages)/10 {
		epCtx.AdditionalInfo["logs"] = r.cluster.GetLogs()
		return nil, util.NewLogError(fmt.Errorf("too many errors sending messages: %s", strings.Join(errs, "\n")), r.cluster.GetLogs())
	}
	newState.PendingMessages = r.network.GetAllMessages()
	r.curState = newState
	return newState, nil
}

func (r *CometEnv) DropMessages(messages []core.Message, epCtx *core.StepContext) (core.PState, error) {
	newState := &CometClusterState{
		NodeStates: make(map[int]*CometNodeState),
	}
	for id, s := range r.curState.NodeStates {
		newState.NodeStates[id] = s.Copy()
	}
	for _, m := range messages {
		rm, ok := m.(Message)
		if !ok {
			continue
		}
		r.network.DeleteMessage(rm.ID)
	}
	newState.PendingMessages = r.network.GetAllMessages()
	newState.PendingRequests = copyRequests(r.curState.PendingRequests)
	r.curState = newState

	return newState, nil
}

func (r *CometEnv) ReceiveRequest(req core.Request, epCtx *core.StepContext) (core.PState, error) {
	newState := r.curState.Copy()

	cReq := req.(CometRequest)
	queryString := ""
	if cReq.Type == "get" {
		queryString = fmt.Sprintf(`abci_query?data="%s"`, cReq.Key)
	} else if cReq.Type == "set" {
		queryString = fmt.Sprintf(`broadcast_tx_commit?tx="%s=%s"`, cReq.Key, cReq.Value)
	}
	if queryString == "" {
		r.curState = newState
		return newState, nil
	}

	r.cluster.Execute(queryString)
	newState.PendingRequests = copyRequests(newState.PendingRequests[1:])

	r.curState = newState
	return newState, nil
}

func (r *CometEnv) StopNode(nodeID int, epCtx *core.StepContext) (core.PState, error) {
	if r.cluster == nil {
		return nil, errors.New("cluster not initialized")
	}
	node, ok := r.cluster.GetNode(nodeID)
	if !ok {
		return nil, fmt.Errorf("node %d does not exist", nodeID)
	}
	if !node.IsActive() {
		return nil, fmt.Errorf("node %d is already inactive", nodeID)
	}
	return r.curState.Copy(), node.Stop()
}

func (r *CometEnv) StartNode(nodeID int, epCtx *core.StepContext) (core.PState, error) {
	if r.cluster == nil {
		return nil, errors.New("cluster not initialized")
	}
	node, ok := r.cluster.GetNode(nodeID)
	if !ok {
		return nil, fmt.Errorf("node %d does not exist", nodeID)
	}
	if node.IsActive() {
		return nil, fmt.Errorf("node %d is already active", nodeID)
	}
	return r.curState.Copy(), node.Start()
}

func NewCometEnv(ctx context.Context, cfg *CometClusterConfig) *CometEnv {
	network := NewInterceptNetwork(ctx, cfg.InterceptServerPort)
	network.Start()
	env := &CometEnv{
		ctx:     ctx,
		cfg:     cfg,
		cluster: nil,
		network: network,

		curState: nil,
	}

	go func(ctx context.Context, env *CometEnv) {
		<-ctx.Done()
		env.Cleanup()
	}(ctx, env)

	return env
}
