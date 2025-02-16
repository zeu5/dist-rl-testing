package etcd

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"path"
	"strconv"

	"github.com/zeu5/dist-rl-testing/core"
	"github.com/zeu5/dist-rl-testing/util"
	"go.etcd.io/raft/v3"
	pb "go.etcd.io/raft/v3/raftpb"
)

// config of the raft environment
type RaftEnvironmentConfig struct {
	NumNodes      int
	ElectionTick  int
	HeartbeatTick int
	Requests      int

	TraceRecordPath string
	RecordTraces    bool
}

// Wrapper around raft nodes, storage and in transit messages to allow for implementing a partition interface
type RaftPartitionEnv struct {
	config   RaftEnvironmentConfig
	nodes    map[uint64]*raft.RawNode
	storages map[uint64]*raft.MemoryStorage
	messages map[string]pb.Message
	curState *RaftState
	rand     *rand.Rand

	curEventTrace []interface{}
}

var _ core.PEnvironment = &RaftPartitionEnv{}

// Implements core.PEnvironment interface
func NewPartitionEnvironment(config RaftEnvironmentConfig) *RaftPartitionEnv {
	return &RaftPartitionEnv{
		config:   config,
		nodes:    make(map[uint64]*raft.RawNode),
		storages: make(map[uint64]*raft.MemoryStorage),
		messages: make(map[string]pb.Message),
		rand:     rand.New(rand.NewSource(0)),

		curEventTrace: make([]interface{}, 0),
	}
}

// Constructs a new environment when required with the stored config.
// Used in parallel experiments. Implements core.PEnvironmentConstructor
type RaftPartitionEnvConstructor struct {
	config RaftEnvironmentConfig
}

var _ core.PEnvironmentConstructor = &RaftPartitionEnvConstructor{}

// Creates a new partition environment constructor
func NewPartitionEnvironmentConstructor(config RaftEnvironmentConfig) *RaftPartitionEnvConstructor {
	return &RaftPartitionEnvConstructor{
		config: config,
	}
}

// Creates a new environment
func (r *RaftPartitionEnvConstructor) NewPEnvironment(_ context.Context, _ int) core.PEnvironment {
	return NewPartitionEnvironment(r.config)
}

func (r *RaftPartitionEnv) makeNodes() {
	for i := 0; i < r.config.NumNodes; i++ {
		confChanges := make([]pb.ConfChangeV2, r.config.NumNodes)
		for i := 0; i < r.config.NumNodes; i++ {
			confChanges[i] = pb.ConfChange{NodeID: uint64(i + 1), Type: pb.ConfChangeAddNode}.AsV2()
		}
		storage := raft.NewMemoryStorage()
		nodeID := uint64(i + 1)
		r.storages[nodeID] = storage
		r.nodes[nodeID], _ = raft.NewRawNode(&raft.Config{
			ID:                        nodeID,
			ElectionTick:              r.config.ElectionTick,
			HeartbeatTick:             r.config.HeartbeatTick,
			Storage:                   storage,
			MaxSizePerMsg:             1024 * 1024,
			MaxInflightMsgs:           256,
			MaxUncommittedEntriesSize: 1 << 30,
			Logger:                    &raft.DefaultLogger{Logger: log.New(io.Discard, "", 0)},
			CheckQuorum:               true,
		})
		for _, cc := range confChanges {
			r.nodes[nodeID].ApplyConfChange(cc)
		}
	}
	initState := &RaftState{
		NodeStates:      make(map[uint64]raft.Status),
		SnapshotIndices: make(map[uint64]uint64),
		MessageMap:      copyMessageMap(r.messages),
		Logs:            make(map[uint64][]pb.Entry),
		PendingRequests: make([]pb.Message, r.config.Requests),
		ticks:           0,
	}
	for i := 0; i < r.config.Requests; i++ {
		initState.PendingRequests[i] = pb.Message{
			Type: pb.MsgProp,
			From: uint64(0),
			Entries: []pb.Entry{
				{Data: []byte(strconv.Itoa(i + 1))},
			},
		}
	}

	for id, node := range r.nodes {
		initState.NodeStates[id] = node.Status()
		initState.Logs[id] = make([]pb.Entry, 0)

	}
	r.curState = initState
}

func (p *RaftPartitionEnv) recordEventsInTransition(id int, oldS, newS raft.Status) {
	old := oldS.RaftState
	new := newS.RaftState
	oldTerm := oldS.Term
	newTerm := newS.Term
	if old != new && new == raft.StateLeader {
		p.curEventTrace = append(p.curEventTrace, map[string]interface{}{
			"name": "BecomeLeader",
			"params": map[string]interface{}{
				"node": id,
			},
		})
		p.curEventTrace = append(p.curEventTrace, map[string]interface{}{
			"name": "ClientRequest",
			"params": map[string]interface{}{
				"request": 0,
				"leader":  id,
			},
		})
	} else if (old != new && new == raft.StateCandidate) || (oldTerm < newTerm && old == new && new == raft.StateCandidate) {
		p.curEventTrace = append(p.curEventTrace, map[string]interface{}{
			"name": "Timeout",
			"params": map[string]interface{}{
				"node": id,
			},
		})
	}
}

func (p *RaftPartitionEnv) recordSendAndReceiveEvents(message pb.Message, send bool) {
	name := "DeliverMessage"
	if send {
		name = "SendMessage"
	}
	p.curEventTrace = append(p.curEventTrace, map[string]interface{}{
		"name": name,
		"params": map[string]interface{}{
			"type":     message.Type.String(),
			"term":     message.Term,
			"from":     message.From,
			"to":       message.To,
			"log_term": message.LogTerm,
			"entries":  message.Entries,
			"index":    message.Index,
			"commit":   message.Commit,
			"vote":     message.Vote,
			"reject":   message.Reject,
		},
	})
}

// deliver the specified message in the system and returns the subsequent state, no tick pass?
func (p *RaftPartitionEnv) deliverMessage(m core.Message) core.PState {
	rm := m.(RaftMessageWrapper)
	node, exists := p.nodes[rm.Message.To]
	msgK := util.JsonHash(rm.Message)
	if exists {
		node.Step(rm.Message)
	}
	delete(p.messages, msgK)
	if p.config.RecordTraces {
		p.recordSendAndReceiveEvents(rm.Message, false)
	}

	newState := p.curState.Copy()
	for id, node := range p.nodes {
		if node.HasReady() {
			ready := node.Ready()
			if !raft.IsEmptySnap(ready.Snapshot) {
				p.storages[id].ApplySnapshot(ready.Snapshot)
				// snap, err := p.storages[id].Snapshot()
			}
			if len(ready.Entries) > 0 {
				p.storages[id].Append(ready.Entries)
			}
			if p.config.RecordTraces && len(ready.CommittedEntries) > 0 {
				p.curEventTrace = append(p.curEventTrace, map[string]interface{}{
					"name": "AdvanceCommitIndex",
					"params": map[string]interface{}{
						"i": int(id),
					},
				})
			}
			for _, message := range ready.Messages {
				msgK := util.JsonHash(message)
				p.messages[msgK] = message
				if p.config.RecordTraces {
					p.recordSendAndReceiveEvents(message, true)
				}
			}
			node.Advance(ready)
		}
		// add status
		status := node.Status()
		newState.NodeStates[id] = status
		if p.config.RecordTraces {
			p.recordEventsInTransition(int(id), p.curState.NodeStates[id], status)
		}

		// add log
		newState.Logs[id] = make([]pb.Entry, 0)
		storage := p.storages[id]
		lastIndex, _ := storage.LastIndex()
		ents, err := storage.Entries(1, lastIndex+1, 1024*1024) // hardcoded value from link_env.go
		if err == nil {
			newState.Logs[id] = copyLogList(ents)
		} else {
			panic("error in reading entries in the log")
		}

		// add snapshot index
		snap, err := storage.Snapshot()
		if err == nil {
			newState.SnapshotIndices[id] = snap.Metadata.Index
		}
	}
	newState.MessageMap = copyMessageMap(p.messages)
	p.curState = newState
	return newState
}

// drops the specified message in the system, no tick pass
func (p *RaftPartitionEnv) dropMessage(m core.Message) core.PState {
	delete(p.messages, m.Hash())
	newState := p.curState.Copy()
	delete(newState.MessageMap, m.Hash())
	p.curState = newState
	return newState
}

func (r *RaftPartitionEnv) Reset(eCtx *core.EpisodeContext) (core.PState, error) {
	r.messages = make(map[string]pb.Message)
	if r.config.RecordTraces && len(r.curEventTrace) > 0 {

		traceRecordPath := path.Join(r.config.TraceRecordPath, strconv.Itoa(eCtx.Run), eCtx.Experiment)
		util.EnsureDir(traceRecordPath)
		util.SaveJson(
			path.Join(
				traceRecordPath,
				fmt.Sprintf("trace_%d.json", eCtx.Episode),
			),
			r.curEventTrace,
		)
	}
	r.curEventTrace = make([]interface{}, 0)
	r.makeNodes()
	return r.curState, nil
}

func (p *RaftPartitionEnv) Tick(epCtx *core.StepContext) (core.PState, error) {
	for _, node := range p.nodes {
		node.Tick()
	}
	newState := p.curState.Copy()
	for id, node := range p.nodes {
		if node.HasReady() {
			ready := node.Ready()
			if !raft.IsEmptySnap(ready.Snapshot) {
				p.storages[id].ApplySnapshot(ready.Snapshot)
			}
			if len(ready.Entries) > 0 {
				p.storages[id].Append(ready.Entries)
			}
			if p.config.RecordTraces && len(ready.CommittedEntries) > 0 {
				p.curEventTrace = append(p.curEventTrace, map[string]interface{}{
					"name": "AdvanceCommitIndex",
					"params": map[string]interface{}{
						"i": int(id),
					},
				})
			}
			for _, message := range ready.Messages {
				msgK := util.JsonHash(message)
				p.messages[msgK] = message
				if p.config.RecordTraces {
					p.recordSendAndReceiveEvents(message, true)
				}
			}
			node.Advance(ready)
		}

		// add status
		status := node.Status()
		newState.NodeStates[id] = status
		if p.config.RecordTraces {
			p.recordEventsInTransition(int(id), p.curState.NodeStates[id], status)
		}

		// add log
		newState.Logs[id] = make([]pb.Entry, 0)

		storage := p.storages[id]
		lastIndex, _ := storage.LastIndex()
		ents, err := storage.Entries(1, lastIndex+1, 1024*1024) // hardcoded value from link_env.go
		if err == nil {
			// TODO: copy logs instead of assigning directly
			newState.Logs[id] = copyLogList(ents)
		} else {
			panic("error in reading entries in the log")
		}

		// add snapshot index
		snap, err := storage.Snapshot()
		if err == nil {
			newState.SnapshotIndices[id] = snap.Metadata.Index
		}
	}
	newState.MessageMap = copyMessageMap(p.messages)
	newState.PendingRequests = copyMessagesList(p.curState.PendingRequests)
	p.curState = newState
	return newState, nil
}

func (l *RaftPartitionEnv) DeliverMessages(messages []core.Message, epCtx *core.StepContext) (core.PState, error) {
	var s core.PState = nil
	for _, m := range messages {
		s = l.deliverMessage(m)
	}
	return s, nil
}

func (l *RaftPartitionEnv) DropMessages(messages []core.Message, epCtx *core.StepContext) (core.PState, error) {
	var s core.PState
	for _, m := range messages {
		s = l.dropMessage(m)
	}
	return s, nil
}

func (p *RaftPartitionEnv) ReceiveRequest(req core.Request, epCtx *core.StepContext) (core.PState, error) {
	newState := p.curState.Copy()
	newState.PendingRequests = make([]pb.Message, 0)

	haveLeader := false
	leader := uint64(0)
	for id, node := range p.nodes {
		if node.Status().RaftState == raft.StateLeader {
			haveLeader = true
			leader = id
			break
		}
	}
	remainingRequests := p.curState.PendingRequests
	if haveLeader {
		message := req.(pb.Message)
		message.To = leader
		p.nodes[leader].Step((message))
		remainingRequests = p.curState.PendingRequests[1:]
		if p.config.RecordTraces {
			request, _ := strconv.Atoi(string(message.Entries[0].Data))
			p.curEventTrace = append(p.curEventTrace, map[string]interface{}{
				"name": "ClientRequest",
				"params": map[string]interface{}{
					"request": request,
					"leader":  leader,
				},
			})
		}
	}
	for _, r := range remainingRequests {
		newState.PendingRequests = append(newState.PendingRequests, copyMessage(r))
	}
	p.curState = newState
	return newState, nil
}

func (r *RaftPartitionEnv) StopNode(nodeID int, epCtx *core.StepContext) (core.PState, error) {
	if r.config.RecordTraces {
		r.curEventTrace = append(r.curEventTrace, map[string]interface{}{
			"name": "Remove",
			"params": map[string]interface{}{
				"i": nodeID,
			},
		})
	}
	delete(r.nodes, uint64(nodeID))
	return r.curState, nil
}

func (r *RaftPartitionEnv) StartNode(nodeID int, epCtx *core.StepContext) (core.PState, error) {
	node := uint64(nodeID)
	_, exists := r.nodes[node]
	if exists {
		return nil, fmt.Errorf("node %d already started", node)
	}
	if r.config.RecordTraces {
		r.curEventTrace = append(r.curEventTrace, map[string]interface{}{
			"name": "Add",
			"params": map[string]interface{}{
				"i": nodeID,
			},
		})
	}
	r.nodes[node], _ = raft.NewRawNode(&raft.Config{
		ID:                        node,
		ElectionTick:              r.config.ElectionTick,
		HeartbeatTick:             r.config.HeartbeatTick,
		Storage:                   r.storages[node],
		MaxSizePerMsg:             1024 * 1024,
		MaxInflightMsgs:           256,
		MaxUncommittedEntriesSize: 1 << 30,
		Logger:                    &raft.DefaultLogger{Logger: log.New(io.Discard, "", 0)},
	})
	peers := make([]raft.Peer, r.config.NumNodes)
	for i := 0; i < r.config.NumNodes; i++ {
		peers[i] = raft.Peer{ID: uint64(i + 1)}
	}
	r.nodes[node].Bootstrap(peers)
	return r.curState, nil
}
