package etcd

import (
	"go.etcd.io/raft/v3"
	pb "go.etcd.io/raft/v3/raftpb"
	"go.etcd.io/raft/v3/tracker"
)

func copyMessage(m pb.Message) pb.Message {
	newMessage := pb.Message{
		Type:       m.Type,
		To:         m.To,
		From:       m.From,
		Term:       m.Term,
		LogTerm:    m.LogTerm,
		Index:      m.Index,
		Commit:     m.Commit,
		Vote:       m.Vote,
		Snapshot:   m.Snapshot,
		Reject:     m.Reject,
		RejectHint: m.RejectHint,
		Context:    m.Context,
		Responses:  m.Responses,
	}
	if len(m.Entries) != 0 {
		newMessage.Entries = make([]pb.Entry, len(m.Entries))
		for i, entry := range m.Entries {
			newMessage.Entries[i] = pb.Entry{
				Term:  entry.Term,
				Index: entry.Index,
				Type:  entry.Type,
				Data:  entry.Data,
			}
		}
	}
	return newMessage
}

func copyMessageMap(messages map[string]pb.Message) map[string]pb.Message {
	c := make(map[string]pb.Message)
	for k, m := range messages {
		c[k] = copyMessage(m)
	}
	return c
}

func copyMessagesList(messages []pb.Message) []pb.Message {
	c := make([]pb.Message, len(messages))
	for i, m := range messages {
		c[i] = copyMessage(m)
	}
	return c
}

func copyEntry(e pb.Entry) pb.Entry {
	return pb.Entry{
		Term:  e.Term,
		Index: e.Index,
		Type:  e.Type,
		Data:  e.Data,
	}
}

func copyLogList(log []pb.Entry) []pb.Entry {
	newLog := make([]pb.Entry, len(log))
	for i, entry := range log {
		newLog[i] = copyEntry(entry)
	}

	return newLog
}

func copyNodeStateMap(m map[uint64]raft.Status) map[uint64]raft.Status {
	c := make(map[uint64]raft.Status)
	for k, s := range m {
		newStatus := raft.Status{
			BasicStatus: raft.BasicStatus{
				ID: s.ID,
				HardState: pb.HardState{
					Term:   s.Term,
					Vote:   s.Vote,
					Commit: s.Commit,
				},
				SoftState: raft.SoftState{
					Lead:      s.Lead,
					RaftState: s.RaftState,
				},
				Applied:        s.Applied,
				LeadTransferee: s.LeadTransferee,
			},
			Config:   s.Config.Clone(),
			Progress: make(map[uint64]tracker.Progress),
		}
		for k, p := range s.Progress {
			newStatus.Progress[k] = tracker.Progress{
				Match:            p.Match,
				Next:             p.Next,
				State:            p.State,
				PendingSnapshot:  p.PendingSnapshot,
				RecentActive:     p.RecentActive,
				MsgAppFlowPaused: p.MsgAppFlowPaused,
				IsLearner:        p.IsLearner,
				Inflights:        p.Inflights.Clone(),
			}
		}
		c[k] = newStatus
	}
	return c
}

// copy the snapshots hashmap
func copySnapshotsMap(snapshots map[uint64]pb.SnapshotMetadata) map[uint64]pb.SnapshotMetadata {
	c := make(map[uint64]pb.SnapshotMetadata)

	for id, sn := range snapshots {
		newSnap := pb.SnapshotMetadata{
			ConfState: sn.ConfState,
			Index:     sn.Index,
			Term:      sn.Term,
		}

		c[id] = newSnap
	}

	return c
}
