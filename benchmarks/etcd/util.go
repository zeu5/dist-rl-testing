package etcd

import (
	"bytes"

	"github.com/zeu5/dist-rl-testing/core"
	"github.com/zeu5/dist-rl-testing/util"
	"go.etcd.io/raft/v3"
	pb "go.etcd.io/raft/v3/raftpb"
)

type RaftMessageWrapper struct {
	pb.Message
}

func (r RaftMessageWrapper) From() int {
	return int(r.Message.From)
}

func (r RaftMessageWrapper) To() int {
	return int(r.Message.To)
}

func (r RaftMessageWrapper) Hash() string {
	return util.JsonHash(r.Message)
}

func copySnapshot(snap *pb.Snapshot) *pb.Snapshot {
	out := &pb.Snapshot{
		Data: bytes.Clone(snap.Data),
		Metadata: pb.SnapshotMetadata{
			ConfState: pb.ConfState{
				Voters:         make([]uint64, len(snap.Metadata.ConfState.Voters)),
				Learners:       make([]uint64, len(snap.Metadata.ConfState.Learners)),
				VotersOutgoing: make([]uint64, len(snap.Metadata.ConfState.VotersOutgoing)),
				LearnersNext:   make([]uint64, len(snap.Metadata.ConfState.LearnersNext)),
				AutoLeave:      snap.Metadata.ConfState.AutoLeave,
			},
			Index: snap.Metadata.Index,
			Term:  snap.Metadata.Term,
		},
	}
	copy(out.Metadata.ConfState.Voters, snap.Metadata.ConfState.Voters)
	copy(out.Metadata.ConfState.Learners, snap.Metadata.ConfState.Learners)
	copy(out.Metadata.ConfState.VotersOutgoing, snap.Metadata.ConfState.VotersOutgoing)
	copy(out.Metadata.ConfState.LearnersNext, snap.Metadata.ConfState.LearnersNext)
	return out
}

func (r RaftMessageWrapper) Copy() RaftMessageWrapper {
	newMessage := pb.Message{
		Type:       r.Message.Type,
		To:         r.Message.To,
		From:       r.Message.From,
		Term:       r.Message.Term,
		LogTerm:    r.Message.LogTerm,
		Index:      r.Message.Index,
		Commit:     r.Message.Commit,
		Vote:       r.Message.Vote,
		Snapshot:   copySnapshot(r.Message.Snapshot),
		Reject:     r.Message.Reject,
		RejectHint: r.Message.RejectHint,
		Context:    bytes.Clone(r.Message.Context),
		Responses:  r.Message.Responses,
	}
	if len(r.Message.Entries) != 0 {
		newMessage.Entries = make([]pb.Entry, len(r.Message.Entries))
		for i, entry := range r.Message.Entries {
			newMessage.Entries[i] = pb.Entry{
				Term:  entry.Term,
				Index: entry.Index,
				Type:  entry.Type,
				Data:  entry.Data,
			}
		}
	}
	return RaftMessageWrapper{
		Message: newMessage,
	}
}

var _ core.Message = RaftMessageWrapper{}

// State of the Raft environment
type RaftState struct {
	// States of each node (obtained from the raft implementation)
	NodeStates      map[uint64]raft.Status
	SnapshotIndices map[uint64]uint64
	// The messages in transit
	MessageMap map[string]pb.Message
	Logs       map[uint64][]pb.Entry
	// Test harness and pending requests
	PendingRequests []pb.Message

	ticks int
}

func (r *RaftState) Copy() *RaftState {
	newState := &RaftState{
		NodeStates:      copyNodeStateMap(r.NodeStates),
		SnapshotIndices: copySnapshotIndexMap(r.SnapshotIndices),
		PendingRequests: copyMessagesList(r.PendingRequests),
		Logs:            make(map[uint64][]pb.Entry),
		MessageMap:      copyMessageMap(r.MessageMap),
		ticks:           r.ticks,
	}

	for id, log := range r.Logs {
		newState.Logs[id] = copyLogList(log)
	}

	return newState
}

var _ core.PState = &RaftState{}

// State of a node in the Raft environment
type RaftNodeState struct {
	State         raft.Status
	Log           []pb.Entry
	SnapshotIndex uint64
}

// Implements the PartitionedSystemState
func (r *RaftState) NodeState(node int) core.NState {
	nID := uint64(node)
	return RaftNodeState{
		State:         r.NodeStates[nID],
		Log:           r.Logs[nID],
		SnapshotIndex: r.SnapshotIndices[nID],
	}
}

// Implements the PartitionedSystemState
func (r *RaftState) Messages() []core.Message {
	messages := make([]core.Message, len(r.MessageMap))
	i := 0
	for _, m := range r.MessageMap {
		messages[i] = RaftMessageWrapper{m}
		i++
	}
	return messages
}

func (r RaftState) Requests() []core.Request {
	requests := make([]core.Request, len(r.PendingRequests))
	for i, r := range r.PendingRequests {
		requests[i] = copyMessage(r)
	}
	return requests
}

func (r RaftState) CanDeliverRequest() bool {
	haveLeader := false
	for _, s := range r.NodeStates {
		if s.RaftState == raft.StateLeader {
			haveLeader = true
			break
		}
	}
	return haveLeader
}

func filterNormalEntries(log []pb.Entry) []pb.Entry {
	result := make([]pb.Entry, 0)
	for _, entry := range log {
		if entry.Type == pb.EntryNormal {
			result = append(result, entry)
		}
	}
	return result
}

func filterEntriesNoElection(log []pb.Entry) []pb.Entry {
	result := make([]pb.Entry, 0)
	for _, entry := range log {
		if entry.Type == 0 && len(entry.Data) > 0 {
			result = append(result, entry)
		}
	}
	return result
}

func TermBound(term int) func(core.PState) bool {
	return func(p core.PState) bool {
		state := p.(*RaftState)
		for _, ns := range state.NodeStates {
			if ns.Term > uint64(term) {
				return true
			}
		}
		return false
	}
}
