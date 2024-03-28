package etcd

import (
	"github.com/zeu5/dist-rl-testing/policies"
	"go.etcd.io/raft/v3"
)

func GetHierarchy(name string) []policies.Predicate {
	switch name {
	case "OneInTerm3":
		return oneInTerm3()
	case "AllInTerm2":
		return allInTerm2()
	case "TermDiff2":
		return termDiff2()
	case "MinCommit2":
		return minCommit2()
	case "LeaderInTerm4":
		return leaderInTerm4()
	case "OneLeaderOneCandidate":
		return oneLeaderOneCandidate()
	}
	return []policies.Predicate{}
}

func oneInTerm3() []policies.Predicate {
	return []policies.Predicate{
		// {Name: "AnyInTerm2", Check: AllInTerm(2)},
		// {Name: "AllInTerm2", Check: AllInTerm(2)},
		{Name: "AnyInTerm3", Check: AnyInTerm(3)},
	}
}

func allInTerm2() []policies.Predicate {
	return []policies.Predicate{
		{Name: "AllInTerm2", Check: AllInTerm(2)},
	}
}

func termDiff2() []policies.Predicate {
	return []policies.Predicate{
		{Name: "TermDiff2", Check: TermDiff(2)},
	}
}

func minCommit2() []policies.Predicate {
	return []policies.Predicate{
		{Name: "MinCommit2", Check: AnyWithCommit(2)},
	}
}

func leaderInTerm4() []policies.Predicate {
	return []policies.Predicate{
		// {Name: "AllInTerm3", Check: AllInTerm(3)},
		{Name: "LeaderInTerm4", Check: LeaderElectedPredicateStateWithTerm(4)},
	}
}

func oneLeaderOneCandidate() []policies.Predicate {
	return []policies.Predicate{
		{Name: "OneLeaderOneCandidate", Check: InState(raft.StateLeader).And(InState(raft.StateCandidate))},
	}
}
