package etcd

import (
	"github.com/zeu5/dist-rl-testing/core"
	pb "go.etcd.io/raft/v3/raftpb"
)

func wrapColor(f func(RaftNodeState) (string, interface{})) core.KVPainter {
	return func(s core.NState) (string, interface{}) {
		ns := s.(RaftNodeState)
		return f(ns)
	}

}

func ColorState() core.KVPainter {
	return wrapColor(func(s RaftNodeState) (string, interface{}) {
		return "state", s.State.RaftState.String()
	})
}

// return the current term of the node, should be just a number
func ColorTerm() core.KVPainter {
	return wrapColor(func(s RaftNodeState) (string, interface{}) {
		return "term", s.State.Term
	})
}

// return the current term of the node, provides a bound that is the maximum considered term
func ColorBoundedTerm(bound int) core.KVPainter {
	return wrapColor(func(s RaftNodeState) (string, interface{}) {
		term := s.State.Term
		if term > uint64(bound) {
			term = uint64(bound)
		}
		return "boundedTerm", term
	})
}

func ColorCommit() core.KVPainter {
	return wrapColor(func(s RaftNodeState) (string, interface{}) {
		return "commit", s.State.Commit
	})
}

func ColorApplied() core.KVPainter {
	return wrapColor(func(s RaftNodeState) (string, interface{}) {
		return "applied", s.State.Applied
	})
}

func ColorIndex() core.KVPainter {
	return wrapColor(func(s RaftNodeState) (string, interface{}) {
		if len(s.Log) == 0 {
			return "index", 0
		}
		return "index", s.Log[len(s.Log)-1].Index
	})
}

func ColorVote() core.KVPainter {
	return wrapColor(func(s RaftNodeState) (string, interface{}) {
		return "vote", s.State.Vote
	})
}

func ColorLeader() core.KVPainter {
	return wrapColor(func(s RaftNodeState) (string, interface{}) {
		return "leader", s.State.Lead
	})
}

// return the snapshot index of a node
func ColorSnapshotIndex() core.KVPainter {
	return wrapColor(func(s RaftNodeState) (string, interface{}) {
		return "snapshotIndex", s.SnapshotIndex
	})
}

// return the snapshot index of a node
func ColorNodeID() core.KVPainter {
	return wrapColor(func(s RaftNodeState) (string, interface{}) {
		return "nodeID", s.State.ID
	})
}

// return the log length of a node
func ColorLogLength() core.KVPainter {
	return wrapColor(func(s RaftNodeState) (string, interface{}) {
		return "logLength", len(s.Log)
	})
}

// return the log length of a node
func ColorLog() core.KVPainter {
	return wrapColor(func(s RaftNodeState) (string, interface{}) {
		result := make([]LogEntryAbs, 0)
		curLog := filterNormalEntries(s.Log)
		for _, ent := range curLog {
			e := LogEntryAbs{
				Term:     ent.Term,
				EntType:  ent.Type,
				dataSize: len(ent.Data),
			}
			result = append(result, e)
		}
		return "log", result
	})
}

func ColorBoundedLog(termBound int) core.KVPainter {
	return wrapColor(func(s RaftNodeState) (string, interface{}) {
		result := make([]LogEntryAbs, 0)
		curLog := filterNormalEntries(s.Log)
		for _, ent := range curLog {
			e := LogEntryAbs{
				Term:     ent.Term,
				EntType:  ent.Type,
				dataSize: len(ent.Data),
			}
			if int(e.Term) > termBound {
				e.Term = uint64(termBound) + 1
			}
			result = append(result, e)
		}
		return "boundedLog", result
	})
}

type LogEntryAbs struct {
	Term     uint64
	EntType  pb.EntryType
	dataSize int
}
