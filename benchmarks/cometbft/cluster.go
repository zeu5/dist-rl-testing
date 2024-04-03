package cometbft

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zeu5/dist-rl-testing/core"
)

type ProposerInfo struct {
	Address []byte
	Index   int
}

func (p *ProposerInfo) Copy() *ProposerInfo {
	out := &ProposerInfo{
		Index: p.Index,
	}
	if p.Address != nil {
		out.Address = make([]byte, len(p.Address))
		copy(out.Address, p.Address)
	}
	return out
}

type CometNodeState struct {
	Height            int               `json:"-"`
	Round             int               `json:"-"`
	Step              string            `json:"-"`
	HeightRoundStep   string            `json:"height/round/step"`
	ProposalBlockHash string            `json:"proposal_block_hash"`
	LockedBlockHash   string            `json:"locked_block_hash"`
	ValidBlockHash    string            `json:"valid_block_hash"`
	Votes             []*CometNodeVotes `json:"height_vote_set"`
	Proposer          *ProposerInfo     `json:"proposer"`

	LogStdout string `json:"-"`
	LogStderr string `json:"-"`
}

type CometNodeVotes struct {
	Round              int      `json:"round"`
	Prevotes           []string `json:"prevotes"`
	PrevotesBitArray   string   `json:"prevotes_bit_array"`
	Precommits         []string `json:"precommits"`
	PrecommitsBitArray string   `json:"precommits_bit_array"`
}

func (c *CometNodeVotes) Copy() *CometNodeVotes {
	out := &CometNodeVotes{
		Round:              c.Round,
		PrevotesBitArray:   c.PrevotesBitArray,
		PrecommitsBitArray: c.PrecommitsBitArray,
	}
	if c.Precommits != nil {
		out.Precommits = make([]string, len(c.Precommits))
		copy(out.Precommits, c.Precommits)
	}
	if c.Prevotes != nil {
		out.Prevotes = make([]string, len(c.Prevotes))
		copy(out.Prevotes, c.Prevotes)
	}
	return out
}

type rawNodeState struct {
	Result *rawResultState `json:"result"`
}

type rawResultState struct {
	RoundState *CometNodeState `json:"round_state"`
}

func EmptyCometNodeState() *CometNodeState {
	return &CometNodeState{}
}

func (r *CometNodeState) Copy() *CometNodeState {
	out := &CometNodeState{
		Height:            r.Height,
		Round:             r.Round,
		Step:              r.Step,
		HeightRoundStep:   r.HeightRoundStep,
		ProposalBlockHash: r.ProposalBlockHash,
		LockedBlockHash:   r.LockedBlockHash,
		ValidBlockHash:    r.ValidBlockHash,
		LogStdout:         r.LogStdout,
		LogStderr:         r.LogStderr,
	}
	if r.Votes != nil {
		out.Votes = make([]*CometNodeVotes, len(r.Votes))
		for i, v := range r.Votes {
			out.Votes[i] = v.Copy()
		}
	}
	if r.Proposer != nil {
		out.Proposer = r.Proposer.Copy()
	}
	return out
}

type CometNodeConfig struct {
	ID         int
	RPCAddress string
	BinaryPath string
	WorkingDir string
}

type CometNode struct {
	ID      int
	process *exec.Cmd
	config  *CometNodeConfig
	ctx     context.Context
	cancel  context.CancelFunc
	active  bool

	stdout *bytes.Buffer
	stderr *bytes.Buffer
}

func NewCometNode(config *CometNodeConfig) *CometNode {
	return &CometNode{
		ID:     config.ID,
		config: config,
	}
}

func (r *CometNode) Create() {
	serverArgs := []string{
		"--home", r.config.WorkingDir,
		"--proxy_app", "persistent_kvstore",
		"istart",
	}

	ctx, cancel := context.WithCancel(context.Background())
	r.process = exec.CommandContext(ctx, r.config.BinaryPath, serverArgs...)
	r.process.Dir = r.config.WorkingDir

	r.ctx = ctx
	r.cancel = cancel
	if r.stdout == nil {
		r.stdout = new(bytes.Buffer)
	}
	if r.stderr == nil {
		r.stderr = new(bytes.Buffer)
	}
	r.process.Stdout = r.stdout
	r.process.Stderr = r.stderr
}

func (r *CometNode) Start() error {
	if r.ctx != nil || r.process != nil {
		return errors.New("comet server already started")
	}

	r.Create()
	r.active = true
	return r.process.Start()
}

func (r *CometNode) Cleanup() {
	os.RemoveAll(r.config.WorkingDir)
}

func (r *CometNode) Stop() error {
	if !r.active || r.ctx == nil || r.process == nil {
		return errors.New("comet server not started")
	}
	select {
	case <-r.ctx.Done():
		return errors.New("comet server already stopped")
	default:
	}
	r.cancel()
	err := r.process.Wait()
	r.active = false
	r.ctx = nil
	r.cancel = func() {}
	r.process = nil

	if err != nil && strings.Contains(err.Error(), "signal: killed") {
		return nil
	}

	return err
}

func (r *CometNode) Terminate() error {
	var err error = nil
	if r.active {
		err = r.Stop()
	}
	r.Cleanup()

	// stdout := strings.ToLower(r.stdout.String())
	// stderr := strings.ToLower(r.stderr.String())

	// Todo: check for bugs in the logs
	return err
}

func (r *CometNode) GetLogs() (string, string) {
	if r.stdout == nil || r.stderr == nil {
		return "", ""
	}
	return r.stdout.String(), r.stderr.String()
}

func (r *CometNode) Info() (*CometNodeState, error) {
	// Todo: figure out how to get the node state
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 5 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   5 * time.Second,
			ResponseHeaderTimeout: 5 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			DisableKeepAlives:     true,
		},
	}

	resp, err := client.Get("http://" + r.config.RPCAddress + "/consensus_state")
	if err != nil {
		return nil, fmt.Errorf("failed to read state from node: %s", err)
	}
	defer resp.Body.Close()

	stateBS, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read state from node: %s", err)
	}

	stateRaw := &rawNodeState{}
	err = json.Unmarshal(stateBS, &stateRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to read state from node: %s", err)
	}

	newState := stateRaw.Result.RoundState.Copy()
	hrs := strings.Split(newState.HeightRoundStep, "/")
	if len(hrs) == 3 {
		newState.Height, _ = strconv.Atoi(hrs[0])
		newState.Round, _ = strconv.Atoi(hrs[1])
		step, _ := strconv.Atoi(hrs[2])
		// Collapsing newheight and newround to propose. There are internal state transitions that RL has no effect over
		// Also, this reduces the nondeterminism in the initial state
		switch step {
		case 1:
			newState.Step = "RoundStepPropose" //"RoundStepNewHeight"
		case 2:
			newState.Step = "RoundStepPropose" //"RoundStepNewRound"
		case 3:
			newState.Step = "RoundStepPropose"
		case 4:
			newState.Step = "RoundStepPrevote"
		case 5:
			newState.Step = "RoundStepPrevoteWait"
		case 6:
			newState.Step = "RoundStepPrecommit"
		case 7:
			newState.Step = "RoundStepPrecommitWait"
		case 8:
			newState.Step = "RoundStepCommit"
		default:
			newState.Step = "RoundStepUnknown"
		}
	}

	newState.LogStdout = r.stdout.String()
	newState.LogStderr = r.stderr.String()

	return newState, nil
}

func (r *CometNode) IsActive() bool {
	return r.active
}

func (r *CometNode) SendRequest(req string) error {
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 5 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   5 * time.Second,
			ResponseHeaderTimeout: 5 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			DisableKeepAlives:     true,
		},
	}
	errCh := make(chan error)
	go func() {
		resp, err := client.Get("http://" + r.config.RPCAddress + "/" + req)
		if err != nil {
			errCh <- fmt.Errorf("error sending request: %s", err)
			return
		}
		defer resp.Body.Close()
		io.ReadAll(resp.Body)
		errCh <- nil
	}()
	select {
	case <-time.After(10 * time.Millisecond):
		// The reason being that a broadcast_tx_req waits for a commit before responding
		// We want to move on
		return nil
	case err := <-errCh:
		return err
	}
}

type Message struct {
	FromAlias     string           `json:"from"`
	ToAlias       string           `json:"to"`
	Data          []byte           `json:"data"`
	Type          string           `json:"type"`
	ID            string           `json:"id"`
	ParsedMessage *WrappedEnvelope `json:"-"`
	FromID        int              `json:"-"`
	ToID          int              `json:"-"`
}

type WrappedEnvelope struct {
	ChanID byte   `json:"chid"`
	Msg    []byte `json:"msg"`
}

func (m Message) To() int {
	return m.ToID
}

func (m Message) From() int {
	return m.FromID
}

func (m Message) Copy() Message {
	n := Message{
		FromAlias: m.FromAlias,
		ToAlias:   m.ToAlias,
		Data:      m.Data,
		Type:      m.Type,
		ID:        m.ID,
		FromID:    m.FromID,
		ToID:      m.ToID,
	}
	if m.ParsedMessage != nil {
		n.ParsedMessage = &WrappedEnvelope{
			ChanID: m.ParsedMessage.ChanID,
			Msg:    make([]byte, len(m.ParsedMessage.Msg)),
		}
		copy(n.ParsedMessage.Msg, m.ParsedMessage.Msg)
	}
	return n
}

func (m Message) Hash() string {
	return m.ID
}

var _ core.Message = Message{}

type nodeKeys struct {
	Public  []byte
	Private []byte
}
type nodeInfo struct {
	ID    int
	Alias string
	Addr  string
	Keys  *nodeKeys
}

type InterceptNetwork struct {
	Port   int
	ctx    context.Context
	server *http.Server

	lock     *sync.Mutex
	nodes    map[int]*nodeInfo
	aliasMap map[string]int
	// Make this bag of messages
	messages map[string]Message
}

func NewInterceptNetwork(ctx context.Context, port int) *InterceptNetwork {

	f := &InterceptNetwork{
		Port:     port,
		ctx:      ctx,
		lock:     new(sync.Mutex),
		nodes:    make(map[int]*nodeInfo),
		aliasMap: make(map[string]int),
		messages: make(map[string]Message),
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.POST("/replica", f.handleReplica)
	r.POST("/event", dummyHandler)
	r.POST("/message", f.handleMessage)
	f.server = &http.Server{
		Addr:    fmt.Sprintf("localhost:%d", port),
		Handler: r,
	}

	return f
}

func dummyHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func (n *InterceptNetwork) MakeNodeByzantine(nodeID uint64) {
	n.lock.Lock()
	nI, ok := n.nodes[int(nodeID)]
	nodeAddr := nI.Addr
	if !ok {
		n.lock.Unlock()
		return
	}
	n.lock.Unlock()

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 5 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   5 * time.Second,
			ResponseHeaderTimeout: 5 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			DisableKeepAlives:     true,
		},
	}
	bs := []byte("ok")
	resp, err := client.Post("http://"+nodeAddr+"/byzantine", "application/text", bytes.NewBuffer(bs))
	if err == nil {
		io.ReadAll(resp.Body)
		resp.Body.Close()
	}
}

func (n *InterceptNetwork) GetAllMessages() map[string]Message {
	out := make(map[string]Message)
	n.lock.Lock()
	defer n.lock.Unlock()
	for k, m := range n.messages {
		out[k] = m.Copy()
	}
	return out
}

func (n *InterceptNetwork) MessageCount() int {
	n.lock.Lock()
	defer n.lock.Unlock()

	// count := len(n.messages)
	return len(n.messages)
}

func (n *InterceptNetwork) handleMessage(c *gin.Context) {
	m := Message{}
	if err := c.ShouldBindJSON(&m); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to unmarshal request"})
		return
	}

	we := &WrappedEnvelope{}
	if err := json.Unmarshal(m.Data, we); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to unmarshal request"})
		return
	}
	m.ParsedMessage = we

	n.lock.Lock()
	m.FromID = n.aliasMap[m.FromAlias]
	m.ToID = n.aliasMap[m.ToAlias]
	n.messages[m.ID] = m
	n.lock.Unlock()

	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func (n *InterceptNetwork) handleReplica(c *gin.Context) {
	nodeI := &nodeInfo{}
	if err := c.ShouldBindJSON(&nodeI); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to unmarshal request"})
		return
	}

	n.lock.Lock()
	n.nodes[nodeI.ID] = nodeI
	n.aliasMap[nodeI.Alias] = nodeI.ID
	n.lock.Unlock()

	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func (n *InterceptNetwork) Start() {
	go func() {
		n.server.ListenAndServe()
	}()

	go func() {
		<-n.ctx.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		n.server.Shutdown(ctx)
	}()
}

func (n *InterceptNetwork) Reset() {
	n.lock.Lock()
	defer n.lock.Unlock()

	n.messages = make(map[string]Message)
	n.nodes = make(map[int]*nodeInfo)
	n.aliasMap = make(map[string]int)
}

func (n *InterceptNetwork) WaitForNodes(numNodes int) bool {
	timeout := time.After(2 * time.Second)
	numConnectedNodes := 0
	for numConnectedNodes != numNodes {
		select {
		case <-n.ctx.Done():
			return false
		case <-timeout:
			return false
		case <-time.After(1 * time.Millisecond):
		}
		n.lock.Lock()
		numConnectedNodes = len(n.nodes)
		n.lock.Unlock()
	}
	return true
}

func (n *InterceptNetwork) SendMessage(ctx context.Context, id string) error {
	n.lock.Lock()
	m, ok := n.messages[id]
	nodeAddr := ""
	if ok {
		nodeAddr = n.nodes[m.ToID].Addr
	}
	n.lock.Unlock()

	if !ok {
		return fmt.Errorf("no such message with id: %s", id)
	}

	bs, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("error marshalling message: %s", err)
	}
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 5 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   5 * time.Second,
			ResponseHeaderTimeout: 5 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			DisableKeepAlives:     true,
		},
	}
	req, err := http.NewRequest("POST", "http://"+nodeAddr+"/message", bytes.NewBuffer(bs))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(ctx)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	io.ReadAll(resp.Body)
	resp.Body.Close()

	n.lock.Lock()
	delete(n.messages, id)
	n.lock.Unlock()
	return nil
}

func (n *InterceptNetwork) DeleteMessage(id string) {
	n.lock.Lock()
	defer n.lock.Unlock()

	delete(n.messages, id)
}

type CometClusterConfig struct {
	NumNodes                int
	CometBinaryPath         string
	BaseInterceptListenPort int
	BaseRPCPort             int
	BaseP2PPort             int
	InterceptServerPort     int
	BaseWorkingDir          string
	NumRequests             int
	CreateEmptyBlocks       bool

	TimeoutPropose   int
	TimeoutPrevote   int
	TimeoutPrecommit int
	TimeoutCommit    int

	TickDuration time.Duration
}

func (c *CometClusterConfig) Copy() *CometClusterConfig {
	return &CometClusterConfig{
		NumNodes:                c.NumNodes,
		CometBinaryPath:         c.CometBinaryPath,
		BaseInterceptListenPort: c.BaseInterceptListenPort,
		BaseRPCPort:             c.BaseRPCPort,
		InterceptServerPort:     c.InterceptServerPort,
		BaseWorkingDir:          c.BaseWorkingDir,
		NumRequests:             c.NumRequests,
		CreateEmptyBlocks:       c.CreateEmptyBlocks,
		TimeoutPropose:          c.TimeoutPropose,
		TimeoutPrevote:          c.TimeoutPrevote,
		TimeoutPrecommit:        c.TimeoutPrecommit,
		TimeoutCommit:           c.TimeoutCommit,
		TickDuration:            c.TickDuration,
	}
}

func (c *CometClusterConfig) SetDefaults() {
	if c.TimeoutPropose == 0 {
		c.TimeoutPropose = 500
	}
	if c.TimeoutPrevote == 0 {
		c.TimeoutPrevote = 20
	}
	if c.TimeoutPrecommit == 0 {
		c.TimeoutPrecommit = 20
	}
	if c.TimeoutCommit == 0 {
		c.TimeoutCommit = 20
	}
	if c.TickDuration == 0 {
		c.TickDuration = 30 * time.Millisecond
	}
}

func (c *CometClusterConfig) GetNodeConfig(id int) *CometNodeConfig {
	return &CometNodeConfig{
		ID:         id,
		WorkingDir: path.Join(c.BaseWorkingDir, fmt.Sprintf("node%d", id)),
		RPCAddress: fmt.Sprintf("localhost:%d", c.BaseRPCPort+(id-1)),
		BinaryPath: c.CometBinaryPath,
	}
}

type CometCluster struct {
	Nodes  map[int]*CometNode
	config *CometClusterConfig
}

func NewCometCluster(config *CometClusterConfig) (*CometCluster, error) {
	config.SetDefaults()
	c := &CometCluster{
		config: config,
		Nodes:  make(map[int]*CometNode),
	}
	// Make nodes
	for i := 1; i <= c.config.NumNodes; i++ {
		nConfig := config.GetNodeConfig(i)
		c.Nodes[i] = NewCometNode(nConfig)
	}

	if _, err := os.Stat(config.BaseWorkingDir); err != nil {
		os.MkdirAll(config.BaseWorkingDir, 0750)
	}

	testnetArgs := []string{
		"itestnet",
		"--o", c.config.BaseWorkingDir,
		"--v", strconv.Itoa(c.config.NumNodes),
		"--timeout-propose", strconv.Itoa(c.config.TimeoutPropose),
		"--timeout-prevote", strconv.Itoa(c.config.TimeoutPrevote),
		"--timeout-precommit", strconv.Itoa(c.config.TimeoutPrecommit),
		"--timeout-commit", strconv.Itoa(c.config.TimeoutCommit),
		"--p2p-port", strconv.Itoa(c.config.BaseP2PPort),
		"--rpc-port", strconv.Itoa(c.config.BaseRPCPort),
		"--intercept-port", strconv.Itoa(c.config.BaseInterceptListenPort),
		"--intercept-server-addr", fmt.Sprintf("localhost:%d", c.config.InterceptServerPort),
		"--debug",
	}
	if config.CreateEmptyBlocks {
		testnetArgs = append(testnetArgs, "--create-empty-blocks")
	}

	cmd := exec.Command(config.CometBinaryPath, testnetArgs...)
	stdout := new(bytes.Buffer)
	cmd.Stdout = stdout
	stderr := new(bytes.Buffer)
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("error creating config: %s\n\nstdout: \n%s\n\n stderr: %s", err, stdout.String(), stderr.String())
	}

	return c, nil
}

func (c *CometCluster) GetNodeStates() map[int]*CometNodeState {
	out := make(map[int]*CometNodeState)
	outCh := make(chan struct {
		State *CometNodeState
		Node  int
	}, c.config.NumNodes)

	wg := new(sync.WaitGroup)
	for id, node := range c.Nodes {
		wg.Add(1)
		go func(id int, node *CometNode, wg *sync.WaitGroup, sCh chan struct {
			State *CometNodeState
			Node  int
		}) {
			state, err := node.Info()
			if err != nil {
				state = EmptyCometNodeState()
				state.LogStdout = node.stdout.String()
				state.LogStderr = node.stderr.String()
			}
			sCh <- struct {
				State *CometNodeState
				Node  int
			}{
				State: state,
				Node:  id,
			}

			wg.Done()
		}(id, node, wg, outCh)
	}

	wg.Wait()
	close(outCh)

	for s := range outCh {
		out[s.Node] = s.State.Copy()
	}

	return out
}

func (c *CometCluster) Start() error {
	for i := 1; i <= c.config.NumNodes; i++ {
		node := c.Nodes[i]
		if err := node.Start(); err != nil {
			return fmt.Errorf("error starting node %d: %s", i, err)
		}
	}
	return nil
}

func (c *CometCluster) Destroy() error {
	var err error = nil
	for _, node := range c.Nodes {
		e := node.Terminate()
		if e != nil {
			err = e
		}
	}
	return err
}

func (c *CometCluster) GetNode(id int) (*CometNode, bool) {
	node, ok := c.Nodes[id]
	return node, ok
}

func (c *CometCluster) GetLogs() string {
	logLines := []string{}
	for nodeID, node := range c.Nodes {
		logLines = append(logLines, fmt.Sprintf("logs for node: %d\n", nodeID))
		stdout, stderr := node.GetLogs()
		logLines = append(logLines, "----- Stdout -----", stdout, "----- Stderr -----", stderr, "\n\n")
	}
	return strings.Join(logLines, "\n")
}

func (c *CometCluster) Execute(req string) error {
	var activeNode *CometNode = nil
	for i := 1; i <= c.config.NumNodes; i++ {
		node, ok := c.Nodes[i]
		if ok && node.IsActive() {
			activeNode = node
			break
		}
	}
	if activeNode == nil {
		return errors.New("no active node")
	}

	return activeNode.SendRequest(req)
}
