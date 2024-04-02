package rsl

import (
	"github.com/zeu5/dist-rl-testing/core"
	"github.com/zeu5/dist-rl-testing/policies"
)

func wrapPredicate(f func(*core.PartitionState) bool) policies.PredicateFunc {
	return func(s core.State) bool {
		ps, ok := s.(*core.PartitionState)
		if !ok {
			return false
		}
		return f(ps)
	}
}

func Decided() policies.PredicateFunc {
	return wrapPredicate(func(ps *core.PartitionState) bool {
		for _, rs := range ps.NodeStates {
			decided := rs.(LocalState).Decided
			if decided > 0 {
				return true
			}
		}
		return false
	})
}

func InState(state RSLState) policies.PredicateFunc {
	return wrapPredicate(func(ps *core.PartitionState) bool {
		for _, rs := range ps.NodeStates {
			if rs.(LocalState).State == state {
				return true
			}
		}
		return false
	})
}

func NodePrimary(node int) policies.PredicateFunc {
	return wrapPredicate(func(ps *core.PartitionState) bool {
		for n, rs := range ps.NodeStates {
			if rs.(LocalState).State == StateStablePrimary && n == node {
				return true
			}
		}
		return false
	})
}

func NodePrimaryInBallot(node int, ballot int) policies.PredicateFunc {
	return wrapPredicate(func(ps *core.PartitionState) bool {
		for n, rs := range ps.NodeStates {
			ls := rs.(LocalState)
			if ls.State == StateStablePrimary && n == node && ls.MaxAcceptedProposal.Ballot.Num == ballot {
				return true
			}
		}
		return false
	})
}

func InStateAndBallot(state RSLState, ballot int) policies.PredicateFunc {
	return wrapPredicate(func(ps *core.PartitionState) bool {
		for _, rs := range ps.NodeStates {
			ls := rs.(LocalState)
			if ls.State == state && ls.MaxAcceptedProposal.Ballot.Num == ballot {
				return true
			}
		}
		return false
	})
}

func OutSyncBallotBy(diff int) policies.PredicateFunc {
	return wrapPredicate(func(ps *core.PartitionState) bool {
		ballotsMap := make(map[int]bool)
		for _, rs := range ps.NodeStates {
			ls := rs.(LocalState)
			ballotsMap[ls.MaxAcceptedProposal.Ballot.Num] = true
		}
		ballots := make([]int, 0)
		for b := range ballotsMap {
			ballots = append(ballots, b)
		}

		maxBallot := ballots[0]
		minBallot := ballots[0]
		for _, b := range ballots {
			if b > maxBallot {
				maxBallot = b
			}
			if b < minBallot {
				minBallot = b
			}
		}
		return maxBallot-minBallot >= diff
	})
}

func AllInSync() policies.PredicateFunc {
	return wrapPredicate(func(ps *core.PartitionState) bool {
		ballotsMap := make(map[int]bool)
		for _, rs := range ps.NodeStates {
			ls := rs.(LocalState)
			ballotsMap[ls.MaxAcceptedProposal.Ballot.Num] = true
		}
		if len(ballotsMap) != 1 {
			return false
		}
		for b := range ballotsMap {
			return b != 0
		}
		return false
	})
}

func AllAtMostBallot(maxBallot int) policies.PredicateFunc {
	return wrapPredicate(func(ps *core.PartitionState) bool {
		ballotsMap := make(map[int]bool)
		for _, rs := range ps.NodeStates {
			ls := rs.(LocalState)
			ballotsMap[ls.MaxAcceptedProposal.Ballot.Num] = true
		}
		for b := range ballotsMap {
			if b > maxBallot {
				return false
			}
		}
		return true
	})
}

func AllAtLeastBallot(minBallot int) policies.PredicateFunc {
	return wrapPredicate(func(ps *core.PartitionState) bool {
		ballotsMap := make(map[int]bool)
		for _, rs := range ps.NodeStates {
			ls := rs.(LocalState)
			ballotsMap[ls.MaxAcceptedProposal.Ballot.Num] = true
		}
		for b := range ballotsMap {
			if b < minBallot {
				return false
			}
		}
		return true
	})
}

func NodeDecided(node int) policies.PredicateFunc {
	return wrapPredicate(func(ps *core.PartitionState) bool {
		for n, rs := range ps.NodeStates {
			decided := rs.(LocalState).Decided
			if n == node && decided > 0 {
				return true
			}
		}
		return false
	})
}

func AnyDecided(d int) policies.PredicateFunc {
	return wrapPredicate(func(ps *core.PartitionState) bool {
		for _, rs := range ps.NodeStates {
			decided := rs.(LocalState).Decided
			if decided == d {
				return true
			}
		}
		return false
	})
}

func AtMostDecided(d int) policies.PredicateFunc {
	return wrapPredicate(func(ps *core.PartitionState) bool {
		for _, rs := range ps.NodeStates {
			decided := rs.(LocalState).Decided
			if decided > d {
				return false
			}
		}
		return true
	})
}

func AtLeastDecided(d int) policies.PredicateFunc {
	return wrapPredicate(func(ps *core.PartitionState) bool {
		for _, rs := range ps.NodeStates {
			decided := rs.(LocalState).Decided
			if decided < d {
				return false
			}
		}
		return true
	})
}

func NodeNumDecided(node, d int) policies.PredicateFunc {
	return wrapPredicate(func(ps *core.PartitionState) bool {
		for n, rs := range ps.NodeStates {
			decided := rs.(LocalState).Decided
			if decided == d && node == n {
				return true
			}
		}
		return false
	})
}

func AnyInBallot(ballot int) policies.PredicateFunc {
	return wrapPredicate(func(ps *core.PartitionState) bool {
		for _, rs := range ps.NodeStates {
			if rs.(LocalState).MaxAcceptedProposal.Ballot.Num == ballot {
				return true
			}
		}
		return false
	})
}

func AllInBallot(ballot int) policies.PredicateFunc {
	return wrapPredicate(func(ps *core.PartitionState) bool {
		for _, rs := range ps.NodeStates {
			if rs.(LocalState).MaxAcceptedProposal.Ballot.Num < ballot {
				return false
			}
		}
		return true
	})
}

func NodeInBallot(node, ballot int) policies.PredicateFunc {
	return wrapPredicate(func(ps *core.PartitionState) bool {
		for n, rs := range ps.NodeStates {
			if rs.(LocalState).MaxAcceptedProposal.Ballot.Num == ballot && n == node {
				return true
			}
		}
		return false
	})
}

func InPreparedBallot(ballot int) policies.PredicateFunc {
	return wrapPredicate(func(ps *core.PartitionState) bool {
		for _, rs := range ps.NodeStates {
			if rs.(LocalState).MaxPreparedBallot.Num == ballot {
				return true
			}
		}
		return false
	})
}

func NodeInPreparedBallot(node, ballot int) policies.PredicateFunc {
	return wrapPredicate(func(ps *core.PartitionState) bool {
		for n, rs := range ps.NodeStates {
			if rs.(LocalState).MaxPreparedBallot.Num == ballot && n == node {
				return true
			}
		}
		return false
	})
}
