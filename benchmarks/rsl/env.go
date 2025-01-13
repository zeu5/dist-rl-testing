package rsl

import (
	"context"
	"errors"
	"strconv"

	"github.com/zeu5/dist-rl-testing/core"
)

type MessageWrapper struct {
	m Message
}

func (m MessageWrapper) From() int {
	return int(m.m.From)
}

func (m MessageWrapper) To() int {
	return int(m.m.To)
}

func (m MessageWrapper) Hash() string {
	return m.m.Hash()
}

var _ core.Message = MessageWrapper{}

type RSLPartitionState struct {
	ReplicaStates   map[uint64]LocalState
	pendingMessages map[string]Message
	pendingRequests []Message
}

func (p *RSLPartitionState) Copy() core.PState {
	newState := &RSLPartitionState{
		ReplicaStates:   copyReplicaStates(p.ReplicaStates),
		pendingMessages: copyMessages(p.pendingMessages),
		pendingRequests: copyMessagesList(p.pendingRequests),
	}
	return newState
}

func (p *RSLPartitionState) NodeState(node int) core.NState {
	s, ok := p.ReplicaStates[uint64(node)]
	if !ok {
		return EmptyLocalState()
	}
	return s.Copy()
}

func (p *RSLPartitionState) Messages() []core.Message {
	return copyToMessageWrappers(p.pendingMessages)
}

func (p *RSLPartitionState) CanDeliverRequest() bool {
	havePrimary := false
	for _, s := range p.ReplicaStates {
		if s.State == StateStablePrimary {
			havePrimary = true
			break
		}
	}
	return havePrimary
}

func (p *RSLPartitionState) Requests() []core.Request {
	out := make([]core.Request, len(p.pendingRequests))
	for i, r := range p.pendingRequests {
		out[i] = r.Copy()
	}
	return out
}

func copyToMessageWrappers(messages map[string]Message) []core.Message {
	out := make([]core.Message, 0)
	for _, m := range messages {
		out = append(out, MessageWrapper{
			m: m.Copy(),
		})
	}
	return out
}

var _ core.PState = &RSLPartitionState{}

type RSLEnvConfig struct {
	Nodes              int
	NodeConfig         NodeConfig
	NumCommands        int
	AdditionalCommands []Command
}

// RSL partition environment
// structure that wraps the nodes to fir the RL environment
type RSLPartitionEnv struct {
	config   RSLEnvConfig
	nodes    map[uint64]*Node
	messages map[string]Message
	curState *RSLPartitionState
}

var _ core.PEnvironment = &RSLPartitionEnv{}

func NewRLSPartitionEnv(c RSLEnvConfig) *RSLPartitionEnv {
	e := &RSLPartitionEnv{
		config:   c,
		nodes:    make(map[uint64]*Node),
		messages: make(map[string]Message),
	}
	return e
}

// Reset creates new nodes and clears all messages
func (r *RSLPartitionEnv) Reset(_ *core.EpisodeContext) (core.PState, error) {
	peers := make([]uint64, r.config.Nodes)
	for i := 0; i < r.config.Nodes; i++ {
		peers[i] = uint64(i + 1)
	}

	newState := &RSLPartitionState{
		ReplicaStates:   make(map[uint64]LocalState),
		pendingMessages: copyMessages(r.messages),
		pendingRequests: make([]Message, r.config.NumCommands),
	}

	r.config.NodeConfig.Peers = peers
	for i := 0; i < r.config.Nodes; i++ {
		cfg := r.config.NodeConfig.Copy()
		cfg.ID = uint64(i + 1)
		node := NewNode(&cfg)
		r.nodes[node.ID] = node
		newState.ReplicaStates[node.ID] = node.State()

		node.Start()
	}
	for i := 0; i < r.config.NumCommands; i++ {
		cmd := Message{Type: MessageCommand, Command: Command{Data: []byte(strconv.Itoa(i + 1))}}
		newState.pendingRequests[i] = cmd
	}
	for _, c := range r.config.AdditionalCommands {
		cmd := Message{Type: MessageCommand, Command: c.Copy()}
		newState.pendingRequests = append(newState.pendingRequests, cmd)
	}
	r.curState = newState
	return newState, nil
}

func (r *RSLPartitionEnv) StartNode(nodeID int, _ *core.StepContext) (core.PState, error) {
	_, ok := r.nodes[uint64(nodeID)]
	if ok {
		// Node already started
		return nil, errors.New("node already started")
	}
	cfg := r.config.NodeConfig.Copy()
	cfg.ID = uint64(nodeID)
	node := NewNode(&cfg)
	r.nodes[uint64(nodeID)] = node

	node.Start()
	return r.curState.Copy(), nil
}

func (r *RSLPartitionEnv) StopNode(node int, _ *core.StepContext) (core.PState, error) {
	delete(r.nodes, uint64(node))
	return r.curState.Copy(), nil
}

func (r *RSLPartitionEnv) ReceiveRequest(req core.Request, _ *core.StepContext) (core.PState, error) {
	newState := &RSLPartitionState{
		ReplicaStates:   copyReplicaStates(r.curState.ReplicaStates),
		pendingMessages: copyMessages(r.messages),
		pendingRequests: make([]Message, 0),
	}
	remainingRequests := r.curState.pendingRequests
	cmd := req.(Message)
	for _, n := range r.nodes {
		if n.IsPrimary() {
			n.Propose(cmd.Command)
			remainingRequests = r.curState.pendingRequests[1:]
			break
		}
	}
	for _, re := range remainingRequests {
		newState.pendingRequests = append(newState.pendingRequests, re.Copy())
	}
	r.curState = newState
	return newState, nil
}

// Moves clock of each process by one
// Checks for messages after clock has advanced
func (r *RSLPartitionEnv) Tick(_ *core.StepContext) (core.PState, error) {
	newState := &RSLPartitionState{
		ReplicaStates:   make(map[uint64]LocalState),
		pendingMessages: make(map[string]Message),
	}
	for _, node := range r.nodes {
		node.Tick()
		if node.HasReady() {
			rd := node.Ready()
			for _, m := range rd.Messages {
				r.messages[m.Hash()] = m
			}
		}
		newState.ReplicaStates[node.ID] = node.State()
	}
	newState.pendingMessages = copyMessages(r.messages)
	newState.pendingRequests = copyMessagesList(r.curState.pendingRequests)
	r.curState = newState
	return newState, nil
}

func (l *RSLPartitionEnv) DeliverMessages(messages []core.Message, _ *core.StepContext) (core.PState, error) {
	var s core.PState = nil
	for _, m := range messages {
		s = l.deliverMessage(m)
	}
	return s, nil
}

// Deliver the message
func (r *RSLPartitionEnv) deliverMessage(m core.Message) core.PState {
	// Node that m is of type MessageWrapper
	message := m.(MessageWrapper).m

	if message.Type == MessageCommand {
		for _, n := range r.nodes {
			if n.IsPrimary() {
				n.Propose(message.Command)
				delete(r.messages, m.Hash())
			}
		}
	} else {
		node, ok := r.nodes[message.To]
		if ok {
			node.Step(message)
			delete(r.messages, message.Hash())
		}
	}

	newState := &RSLPartitionState{
		ReplicaStates:   make(map[uint64]LocalState),
		pendingMessages: make(map[string]Message),
	}
	for id, node := range r.nodes {
		newState.ReplicaStates[id] = node.State()
		if node.HasReady() {
			rd := node.Ready()
			for _, m := range rd.Messages {
				r.messages[m.Hash()] = m
			}
		}
	}
	newState.pendingMessages = copyMessages(r.messages)
	newState.pendingRequests = copyMessagesList(r.curState.pendingRequests)
	r.curState = newState

	return newState
}

func (l *RSLPartitionEnv) DropMessages(messages []core.Message, _ *core.StepContext) (core.PState, error) {
	var s core.PState
	for _, m := range messages {
		s = l.dropMessage(m)
	}
	return s, nil
}

// Remove the message from the pool
func (r *RSLPartitionEnv) dropMessage(m core.Message) core.PState {
	// Node that m is of type MessageWrapper
	message := m.(MessageWrapper).m
	delete(r.messages, message.Hash())
	newState := &RSLPartitionState{
		ReplicaStates:   copyReplicaStates(r.curState.ReplicaStates),
		pendingMessages: copyMessages(r.messages),
		pendingRequests: copyMessagesList(r.curState.pendingRequests),
	}
	r.curState = newState
	return newState
}

type RSLPartitionEnvConstructor struct {
	config RSLEnvConfig
}

var _ core.PEnvironmentConstructor = &RSLPartitionEnvConstructor{}

func NewRSLPartitionEnvConstructor(c RSLEnvConfig) *RSLPartitionEnvConstructor {
	return &RSLPartitionEnvConstructor{
		config: c,
	}
}

func (r *RSLPartitionEnvConstructor) NewPEnvironment(_ context.Context, _ int) core.PEnvironment {
	return NewRLSPartitionEnv(r.config)
}
