package etcd

import (
	"math"

	"github.com/zeu5/dist-rl-testing/core"
	"github.com/zeu5/dist-rl-testing/policies"
	"go.etcd.io/raft/v3"
	"go.etcd.io/raft/v3/raftpb"
)

func AllInTerm(term uint64) policies.PredicateFunc {
	return func(s core.State) bool {
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
	}
}

// Wraps the predicate that works only on core.PartitionState and returns a policies.PredicateFunc
func wrapPredicate(pred func(*core.PartitionState) bool) func(core.State) bool {
	return func(s core.State) bool {
		pS, ok := s.(*core.PartitionState)
		if !ok {
			return false
		}
		return pred(pS)
	}
}

func AllInTermAtLeast(term uint64) policies.PredicateFunc {
	return wrapPredicate(func(ps *core.PartitionState) bool {
		for _, state := range ps.NodeStates {
			rs := state.(RaftNodeState).State
			if rs.Term != 0 && rs.Term < term {
				return false
			}
		}
		return true
	})
}

func AnyInTermAtLeast(term uint64) policies.PredicateFunc {
	return wrapPredicate(func(ps *core.PartitionState) bool {
		for _, state := range ps.NodeStates {
			rs := state.(RaftNodeState).State
			if rs.Term != 0 && rs.Term >= term {
				return true
			}
		}
		return false
	})
}

func AnyInTerm(term uint64) policies.PredicateFunc {
	return wrapPredicate(func(ps *core.PartitionState) bool {
		for _, state := range ps.NodeStates {
			rState := state.(RaftNodeState)
			if rState.State.Term == term {
				return true
			}
		}
		return false
	})
}

func AnyWithCommit(commit int) policies.PredicateFunc {
	return wrapPredicate(func(ps *core.PartitionState) bool {
		bootstrapedEntries := len(ps.NodeStates)
		for _, state := range ps.NodeStates {
			rState := state.(RaftNodeState)
			if int(rState.State.Commit)-bootstrapedEntries >= commit {
				return true
			}
		}
		return false
	})
}

func AllWithCommit(commit int) policies.PredicateFunc {
	return wrapPredicate(func(ps *core.PartitionState) bool {
		bootstrapedEntries := len(ps.NodeStates)
		for _, state := range ps.NodeStates {
			rState := state.(RaftNodeState)
			if int(rState.State.Commit)-bootstrapedEntries < commit {
				return false
			}
		}
		return true
	})
}

func TermDiff(diff int) policies.PredicateFunc {
	return wrapPredicate(func(ps *core.PartitionState) bool {
		maxTerm := uint64(0)
		minTerm := uint64(math.MaxUint64)

		for _, state := range ps.NodeStates {
			rState := state.(RaftNodeState).State
			if rState.Term == 0 {
				continue
			}
			if rState.Term > maxTerm {
				maxTerm = rState.Term
			}
			if rState.Term < minTerm {
				minTerm = rState.Term
			}
		}

		if maxTerm >= minTerm && maxTerm-minTerm > uint64(diff) {
			return true
		}
		return false
	})
}

// return true if at least one of the nodes is in the leader state
func LeaderElectedPredicateState() policies.PredicateFunc {
	return func(s core.State) bool {
		pS, ok := s.(*core.PartitionState)
		if !ok {
			return false
		}

		for _, state := range pS.NodeStates { // for each node state
			repState := state.(RaftNodeState)
			curState := repState.State // cast into raft.Status
			if curState.RaftState.String() == "StateLeader" {
				return true
			}
		}

		return false
	}
}

// return true if at least one of the nodes is in the leader state
func LeaderElectedPredicateStateWithTerm(term uint64) policies.PredicateFunc {
	return func(s core.State) bool {
		pS, ok := s.(*core.PartitionState)
		if !ok {
			return false
		}

		for _, state := range pS.NodeStates { // for each node state
			repState := state.(RaftNodeState)
			curState := repState.State // cast into raft.Status
			if curState.RaftState.String() == "StateLeader" && curState.Term == term {
				return true
			}
		}

		return false
	}
}

// return true if at least one of the nodes is in the leader state
func LeaderElectedPredicateNumber(elections int) policies.PredicateFunc {
	return func(s core.State) bool {
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
	}
}

// return true if there are leader election committed entries in the specified terms
func LeaderElectedPredicateNumberWithTerms(elections int, reqTerms []uint64) policies.PredicateFunc {
	return func(s core.State) bool {
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
	}
}

// return true if all the nodes are not above the specified term number
func HighestTermFornodes(term uint64) policies.PredicateFunc {
	return func(s core.State) bool {
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
	}
}

// return true if the specified node is in the leader state
func LeaderElectedPredicateSpecific(r_id uint64) policies.PredicateFunc {
	return func(s core.State) bool {
		pS, ok := s.(*core.PartitionState)
		if !ok {
			return false
		}

		state := pS.NodeStates[int(r_id)] // take the node state of the specified node_id
		repState := state.(RaftNodeState)
		curState := repState.State // cast into raft.Status
		return curState.BasicStatus.SoftState.RaftState.String() == "StateLeader"
	}
}

// return true if there is at least one entry in one of the nodes logs
func AtLeastOneLogNotEmpty() policies.PredicateFunc {
	return func(s core.State) bool {
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
	}
}

// return true if there is at least one entry in one of the nodes logs
func AtLeastOneLogNotEmptyTerm(term uint64) policies.PredicateFunc {
	return func(s core.State) bool {
		pS, ok := s.(*core.PartitionState)
		if !ok {
			return false
		}

		for _, state := range pS.NodeStates { // for each node state
			repState := state.(RaftNodeState)
			curLog := committedLog(repState.Log, repState.State)
			// filteredLog := filterEntriesNoElection(curLog)
			if len(curLog) > 0 {
				for _, ent := range curLog {
					if ent.Term == term {
						return true
					}
				}
			}
		}

		return false
	}
}

// return true if there is at least one empty node log
func AtLeastOneLogEmpty() policies.PredicateFunc {
	return func(s core.State) bool {
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
	}
}

// return true if there is at least one log with a single entry
func AtLeastOneLogOneEntry() policies.PredicateFunc {
	return func(s core.State) bool {
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
	}
}

// return true if there is at least one log with a single entry and a higher-term leader election entry
func AtLeastOneLogOneEntryPlusSubsequentLeaderElection() policies.PredicateFunc {
	return func(s core.State) bool {
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
	}
}

// return true if there is a log with at least one entry and a higher-term leader election entry
func AtLeastOneEntryANDSubsequentLeaderElection() policies.PredicateFunc {
	return func(s core.State) bool {
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
	}
}

// return true if there is at least one log with a single entry and a node with a term higher than the entry
func AtLeastOneLogOneEntryPlusnodeInHigherTerm() policies.PredicateFunc {
	return func(s core.State) bool {
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
	}
}

// return true if the specified node's log is empty
func EmptyLogSpecific(r_id uint64) policies.PredicateFunc {
	return func(s core.State) bool {
		pS, ok := s.(*core.PartitionState)
		if !ok {
			return false
		}

		state := pS.NodeStates[int(r_id)] // take the node state of the specified node_id
		repState := state.(RaftNodeState)
		curLog := committedLog(repState.Log, repState.State)
		return len(filterNormalEntries(curLog)) == 0
	}
}

// return true if there is at least one node log with the specified number of entries (no more, dummy entries are ignored)
func ExactEntriesInLog(num int) policies.PredicateFunc {
	return func(s core.State) bool {
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
	}
}

// return true if the specified node log has the specified number of entries (no more, dummy entries are ignored)
func ExactEntriesInLogSpecific(r_id uint64, num int) policies.PredicateFunc {
	return func(s core.State) bool {
		pS, ok := s.(*core.PartitionState)
		if !ok {
			return false
		}

		state := pS.NodeStates[int(r_id)] // take the node state of the specified node_id
		repState := state.(RaftNodeState)
		curLog := committedLog(repState.Log, repState.State)
		return len(filterEntriesNoElection(curLog)) == num
	}
}

// return true if there is at least one node log with entries committed in, at least, the specified number of different terms (dummy entries are ignored)
func EntriesInDifferentTermsInLog(uniqueTerms int) policies.PredicateFunc {
	return func(s core.State) bool {
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
	}
}

// return true if there are at least the specified number of requests in the stack
func StackSizeLowerBound(value int) policies.PredicateFunc {
	return func(s core.State) bool {
		pS, ok := s.(*core.PartitionState)
		if !ok {
			return false
		}

		return len(pS.Requests) >= value
	}
}

func InState(state raft.StateType) policies.PredicateFunc {
	return func(s core.State) bool {
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
	}
}

func InStateWithCommittedEntries(state raft.StateType, num int) policies.PredicateFunc {
	return func(s core.State) bool {
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
	}
}

func MinLogLengthDiff(diff int) policies.PredicateFunc {
	return wrapPredicate(func(ps *core.PartitionState) bool {
		minLogLength := math.MaxInt
		maxLogLength := math.MinInt

		for _, rs := range ps.NodeStates {
			l := len(rs.(RaftNodeState).Log)
			if l > maxLogLength {
				maxLogLength = l
			}
			if l < minLogLength {
				minLogLength = l
			}
		}
		return maxLogLength-minLogLength >= diff && minLogLength > 0 && maxLogLength > 0
	})
}

func MinLogCommittedLengthDiff(diff int) policies.PredicateFunc {
	return wrapPredicate(func(ps *core.PartitionState) bool {
		minLogLength := math.MaxInt
		maxLogLength := math.MinInt

		for _, rs := range ps.NodeStates {
			nrs := rs.(RaftNodeState)
			l := len(committedLog(nrs.Log, nrs.State))
			if l > maxLogLength {
				maxLogLength = l
			}
			if l < minLogLength {
				minLogLength = l
			}
		}
		return maxLogLength-minLogLength >= diff && minLogLength > 0 && maxLogLength > 0
	})
}

func MinCommitGap(diff int) policies.PredicateFunc {
	return wrapPredicate(func(ps *core.PartitionState) bool {
		minCommit := math.MaxInt
		maxCommit := math.MinInt

		for _, rs := range ps.NodeStates {
			l := int(rs.(RaftNodeState).State.Commit)
			if l > maxCommit {
				maxCommit = l
			}
			if l < minCommit {
				minCommit = l
			}
		}
		return maxCommit-minCommit >= diff
	})
}

func TermDiffWithLeader(diff int) policies.PredicateFunc {
	return wrapPredicate(func(ps *core.PartitionState) bool {
		maxTerm := uint64(0)
		minTerm := uint64(math.MaxUint64)
		minTermNode := 0

		for node, state := range ps.NodeStates {
			rState := state.(RaftNodeState).State
			if rState.Term == 0 {
				continue
			}
			if rState.Term > maxTerm {
				maxTerm = rState.Term
			}
			if rState.Term < minTerm {
				minTerm = rState.Term
				minTermNode = node
			}
		}

		if maxTerm >= minTerm && minTerm > 0 && maxTerm-minTerm > uint64(diff) {
			state := ps.NodeStates[minTermNode]
			return state.(RaftNodeState).State.RaftState == raft.StateLeader
		}
		return false
	})
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
