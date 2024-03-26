package etcd

import (
	"github.com/zeu5/dist-rl-testing/core"
	"github.com/zeu5/dist-rl-testing/policies"
	"go.etcd.io/raft/v3"
	"go.etcd.io/raft/v3/raftpb"
)

func AllInTerm(term uint64) policies.Predicate {
	return policies.Predicate{
		Name: "AllInTerm",
		Check: func(s core.State) bool {
			pS, ok := s.(*core.PartitionState)
			if !ok {
				return false
			}

			for _, state := range pS.NodeStates { // for each node state
				repState := state.(RaftNodeState)
				curState := repState.State // cast into raft.Status
				if curState.Term == term {
					return true
				}
			}

			return false
		},
	}
}

// return true if at least one of the nodes is in the leader state
func LeaderElectedPredicateState() policies.Predicate {
	return policies.Predicate{
		Name: "LeaderElectedPredicateState",
		Check: func(s core.State) bool {
			pS, ok := s.(*core.PartitionState)
			if !ok {
				return false
			}

			for _, state := range pS.NodeStates { // for each node state
				repState := state.(RaftNodeState)
				curState := repState.State // cast into raft.Status
				if curState.BasicStatus.SoftState.RaftState.String() == "StateLeader" {
					return true
				}
			}

			return false
		},
	}
}

// return true if at least one of the nodes is in the leader state
func LeaderElectedPredicateStateWithTerm(term uint64) policies.Predicate {
	return policies.Predicate{
		Name: "LeaderElectedPredicateStateWithTerm",
		Check: func(s core.State) bool {
			pS, ok := s.(*core.PartitionState)
			if !ok {
				return false
			}

			for _, state := range pS.NodeStates { // for each node state
				repState := state.(RaftNodeState)
				curState := repState.State // cast into raft.Status
				if curState.BasicStatus.SoftState.RaftState.String() == "StateLeader" && curState.Term == term {
					return true
				}
			}

			return false
		},
	}
}

// return true if at least one of the nodes is in the leader state
func LeaderElectedPredicateNumber(elections int) policies.Predicate {
	return policies.Predicate{
		Name: "LeaderElectedPredicateNumber",
		Check: func(s core.State) bool {
			pS, ok := s.(*core.PartitionState)
			if !ok {
				return false
			}

			for _, state := range pS.NodeStates { // for each node state
				repState := state.(RaftNodeState)
				curLog := committedLog(repState.Log, repState.State)
				terms := make(map[uint64]bool, 0)          // set of unique terms
				filteredLog := filterNormalEntries(curLog) // remove dummy entries
				for _, ent := range filteredLog {          // for each entry
					if len(ent.Data) == 0 {
						terms[ent.Term] = true // add its term to the set
					}
				}
				if len(terms) == elections { // check number of unique terms
					return true
				}
			}

			return false
		},
	}
}

// return true if there are leader election committed entries in the specified terms
func LeaderElectedPredicateNumberWithTerms(elections int, reqTerms []uint64) policies.Predicate {
	return policies.Predicate{
		Name: "LeaderElectedPredicateNumberWithTerms",
		Check: func(s core.State) bool {
			pS, ok := s.(*core.PartitionState)
			if !ok {
				return false
			}

			for _, state := range pS.NodeStates { // for each node state
				repState := state.(RaftNodeState)
				curLog := committedLog(repState.Log, repState.State)
				terms := make(map[uint64]bool, 0)          // set of unique terms
				filteredLog := filterNormalEntries(curLog) // remove dummy entries
				termIndex := 0
				for _, ent := range filteredLog { // for each entry
					if len(ent.Data) == 0 { // it's a leader election
						if ent.Term == reqTerms[termIndex] { // it happened in the specified term
							terms[ent.Term] = true           // add its term to the set
							if termIndex < len(reqTerms)-1 { // check to not get out of array
								termIndex++ // move to next target term
							}
						}
					}
				}
				if len(terms) >= elections { // check number of unique terms
					return true
				}
			}

			return false
		},
	}
}

// return true if all the nodes are not above the specified term number
func HighestTermFornodes(term uint64) policies.Predicate {
	return policies.Predicate{
		Name: "HighestTermFornodes",
		Check: func(s core.State) bool {
			pS, ok := s.(*core.PartitionState)
			if !ok {
				return false
			}

			for _, state := range pS.NodeStates { // for each node state
				repState := state.(RaftNodeState)
				curState := repState.State // cast into raft.Status
				if curState.Term > term {
					return false
				}
			}
			return true
		},
	}
}

// return true if the specified node is in the leader state
func LeaderElectedPredicateSpecific(r_id uint64) policies.Predicate {
	return policies.Predicate{
		Name: "LeaderElectedPredicateSpecific",
		Check: func(s core.State) bool {
			pS, ok := s.(*core.PartitionState)
			if !ok {
				return false
			}

			state := pS.NodeStates[int(r_id)] // take the node state of the specified node_id
			repState := state.(RaftNodeState)
			curState := repState.State // cast into raft.Status
			return curState.BasicStatus.SoftState.RaftState.String() == "StateLeader"
		},
	}
}

// return true if there is at least one entry in one of the nodes logs
func AtLeastOneLogNotEmpty() policies.Predicate {
	return policies.Predicate{
		Name: "AtLeastOneLogNotEmpty",
		Check: func(s core.State) bool {
			pS, ok := s.(*core.PartitionState)
			if !ok {
				return false
			}

			for _, state := range pS.NodeStates { // for each node state
				repState := state.(RaftNodeState)
				curLog := committedLog(repState.Log, repState.State)
				if len(filterEntriesNoElection(curLog)) > 0 {
					return true
				}
			}

			return false
		},
	}
}

// return true if there is at least one entry in one of the nodes logs
func AtLeastOneLogNotEmptyTerm(term uint64) policies.Predicate {
	return policies.Predicate{
		Name: "AtLeastOneLogNotEmptyTerm",
		Check: func(s core.State) bool {
			pS, ok := s.(*core.PartitionState)
			if !ok {
				return false
			}

			for _, state := range pS.NodeStates { // for each node state
				repState := state.(RaftNodeState)
				curLog := committedLog(repState.Log, repState.State)
				filteredLog := filterEntriesNoElection(curLog)
				if len(filteredLog) > 0 {
					for _, ent := range filteredLog {
						if ent.Term == term {
							return true
						}
					}
				}
			}

			return false
		},
	}
}

// return true if there is at least one empty node log
func AtLeastOneLogEmpty() policies.Predicate {
	return policies.Predicate{
		Name: "AtLeastOneLogEmpty",
		Check: func(s core.State) bool {
			pS, ok := s.(*core.PartitionState)
			if !ok {
				return false
			}

			for _, state := range pS.NodeStates { // for each node state
				repState := state.(RaftNodeState)
				curLog := committedLog(repState.Log, repState.State)
				if len(filterEntriesNoElection(curLog)) == 0 {
					return true
				}
			}

			return false
		},
	}
}

// return true if there is at least one log with a single entry
func AtLeastOneLogOneEntry() policies.Predicate {
	return policies.Predicate{
		Name: "AtLeastOneLogOneEntry",
		Check: func(s core.State) bool {
			pS, ok := s.(*core.PartitionState)
			if !ok {
				return false
			}

			for _, state := range pS.NodeStates { // for each node state
				repState := state.(RaftNodeState)
				curLog := committedLog(repState.Log, repState.State)
				if len(filterEntriesNoElection(curLog)) == 1 {
					return true
				}
			}

			return false
		},
	}
}

// return true if there is at least one log with a single entry and a higher-term leader election entry
func AtLeastOneLogOneEntryPlusSubsequentLeaderElection() policies.Predicate {
	return policies.Predicate{
		Name: "AtLeastOneLogOneEntryPlusSubsequentLeaderElection",
		Check: func(s core.State) bool {
			pS, ok := s.(*core.PartitionState)
			if !ok {
				return false
			}

			for _, state := range pS.NodeStates { // for each node state
				repState := state.(RaftNodeState)
				curLog := committedLog(repState.Log, repState.State)
				if len(filterEntriesNoElection(curLog)) == 1 {
					// take term of the committed entry
					committedEntryTerm := filterEntriesNoElection((curLog))[0].Term

					unfLog := filterNormalEntries(curLog)
					for _, ent := range unfLog { // for each entry, included leader elections
						if ent.Term > committedEntryTerm {
							return true
						}
					}
				}
			}

			return false
		},
	}
}

// return true if there is a log with at least one entry and a higher-term leader election entry
func AtLeastOneEntryANDSubsequentLeaderElection() policies.Predicate {
	return policies.Predicate{
		Name: "AtLeastOneEntryANDSubsequentLeaderElection",
		Check: func(s core.State) bool {
			pS, ok := s.(*core.PartitionState)
			if !ok {
				return false
			}

			for _, state := range pS.NodeStates { // for each node state
				repState := state.(RaftNodeState)
				curLog := committedLog(repState.Log, repState.State)
				if len(filterEntriesNoElection(curLog)) >= 1 {
					// take term of the committed entry
					committedEntryTerm := filterEntriesNoElection((curLog))[0].Term

					unfLog := filterNormalEntries(curLog)
					for _, ent := range unfLog { // for each entry, included leader elections
						if ent.Term > committedEntryTerm {
							return true
						}
					}
				}
			}

			return false
		},
	}
}

// return true if there is at least one log with a single entry and a node with a term higher than the entry
func AtLeastOneLogOneEntryPlusnodeInHigherTerm() policies.Predicate {
	return policies.Predicate{
		Name: "AtLeastOneLogOneEntryPlusnodeInHigherTerm",
		Check: func(s core.State) bool {
			pS, ok := s.(*core.PartitionState)
			if !ok {
				return false
			}

			for _, state := range pS.NodeStates { // for each node state
				repState := state.(RaftNodeState)
				curLog := committedLog(repState.Log, repState.State)
				if len(filterEntriesNoElection(curLog)) == 1 {
					// take term of the committed entry
					committedEntryTerm := filterEntriesNoElection((curLog))[0].Term

					for _, otherState := range pS.NodeStates { // check all nodes
						otherRepState := otherState.(RaftNodeState)

						if otherRepState.State.Term > committedEntryTerm { // compare node term with entry term
							return true
						}
					}
				}
			}

			return false
		},
	}
}

// return true if the specified node's log is empty
func EmptyLogSpecific(r_id uint64) policies.Predicate {
	return policies.Predicate{
		Name: "EmptyLogSpecific",
		Check: func(s core.State) bool {
			pS, ok := s.(*core.PartitionState)
			if !ok {
				return false
			}

			state := pS.NodeStates[int(r_id)] // take the node state of the specified node_id
			repState := state.(RaftNodeState)
			curLog := committedLog(repState.Log, repState.State)
			return len(filterNormalEntries(curLog)) == 0
		},
	}
}

// return true if there is at least one node log with the specified number of entries (no more, dummy entries are ignored)
func ExactEntriesInLog(num int) policies.Predicate {
	return policies.Predicate{
		Name: "ExactEntriesInLog",
		Check: func(s core.State) bool {
			pS, ok := s.(*core.PartitionState)
			if !ok {
				return false
			}

			for _, state := range pS.NodeStates { // for each node state
				repState := state.(RaftNodeState)
				curLog := committedLog(repState.Log, repState.State)
				if len(filterEntriesNoElection(curLog)) == num {
					return true
				}
			}

			return false
		},
	}
}

// return true if the specified node log has the specified number of entries (no more, dummy entries are ignored)
func ExactEntriesInLogSpecific(r_id uint64, num int) policies.Predicate {
	return policies.Predicate{
		Name: "ExactEntriesInLogSpecific",
		Check: func(s core.State) bool {
			pS, ok := s.(*core.PartitionState)
			if !ok {
				return false
			}

			state := pS.NodeStates[int(r_id)] // take the node state of the specified node_id
			repState := state.(RaftNodeState)
			curLog := committedLog(repState.Log, repState.State)
			return len(filterEntriesNoElection(curLog)) == num
		},
	}
}

// return true if there is at least one node log with entries committed in, at least, the specified number of different terms (dummy entries are ignored)
func EntriesInDifferentTermsInLog(uniqueTerms int) policies.Predicate {
	return policies.Predicate{
		Name: "EntriesInDifferentTermsInLog",
		Check: func(s core.State) bool {
			pS, ok := s.(*core.PartitionState)
			if !ok {
				return false
			}

			for _, state := range pS.NodeStates { // for each node state
				repState := state.(RaftNodeState)
				curLog := committedLog(repState.Log, repState.State)
				terms := make(map[uint64]bool, 0)              // set of unique terms
				filteredLog := filterEntriesNoElection(curLog) // remove dummy entries
				for _, ent := range filteredLog {              // for each entry
					terms[ent.Term] = true // add its term to the set
				}
				if len(terms) >= uniqueTerms { // check number of unique terms
					return true
				}
			}

			return false
		},
	}
}

// return true if there are at least the specified number of requests in the stack
func StackSizeLowerBound(value int) policies.Predicate {
	return policies.Predicate{
		Name: "StackSizeLowerBound",
		Check: func(s core.State) bool {
			pS, ok := s.(*core.PartitionState)
			if !ok {
				return false
			}

			return len(pS.Requests) >= value
		},
	}
}

func InState(state raft.StateType) policies.Predicate {
	return policies.Predicate{
		Name: "InState",
		Check: func(s core.State) bool {
			pS, ok := s.(*core.PartitionState)
			if !ok {
				return false
			}
			for _, rs := range pS.NodeStates {
				repState := rs.(RaftNodeState)
				if repState.State.RaftState == state {
					return true
				}
			}
			return false
		},
	}
}

func InStateWithCommittedEntries(state raft.StateType, num int) policies.Predicate {
	return policies.Predicate{
		Name: "InStateWithCommittedEntries",
		Check: func(s core.State) bool {
			pS, ok := s.(*core.PartitionState)
			if !ok {
				return false
			}
			for _, rs := range pS.NodeStates {
				repState := rs.(RaftNodeState)
				curLog := committedLog(repState.Log, repState.State)
				filteredLog := filterEntriesNoElection(curLog)
				if repState.State.RaftState == state && len(filteredLog) == num {
					return true
				}
			}
			return false
		},
	}
}

// extract the committed log for a node
func committedLog(log []raftpb.Entry, status raft.Status) []raftpb.Entry {
	committed := int(status.Commit)
	var result []raftpb.Entry

	if committed < len(log) {
		result = copyLogList(log[0:committed])
	} else {
		result = copyLogList(log)
	}

	return result
}
