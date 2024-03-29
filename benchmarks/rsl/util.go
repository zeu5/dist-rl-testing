package rsl

import "github.com/zeu5/dist-rl-testing/core"

func IsProposalUptoDate(decree int, ballot Ballot, compare Proposal) bool {
	return (decree == compare.Decree && ballot.Num > compare.Ballot.Num) || (decree == compare.Decree+1 && ballot.Num == compare.Ballot.Num)
}

func copyMessages(messages map[string]Message) map[string]Message {
	out := make(map[string]Message)
	for k, m := range messages {
		out[k] = m.Copy()
	}
	return out
}

func copyReplicaStates(states map[uint64]LocalState) map[uint64]LocalState {
	out := make(map[uint64]LocalState)
	for id, s := range states {
		out[id] = s.Copy()
	}
	return out
}

func copyMessagesList(messages []Message) []Message {
	out := make([]Message, len(messages))
	for i, m := range messages {
		out[i] = m.Copy()
	}
	return out
}

func BallotBound(bound int) func(core.PState) bool {
	return func(s core.PState) bool {
		ps, ok := s.(*RSLPartitionState)
		if !ok {
			return false
		}
		for _, rs := range ps.ReplicaStates {
			if rs.MaxAcceptedProposal.Ballot.Num > bound {
				return true
			}
		}
		return false
	}
}
