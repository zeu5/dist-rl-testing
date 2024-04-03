package etcd

import (
	"fmt"
	"strings"

	"github.com/zeu5/dist-rl-testing/benchmarks/common"
	"github.com/zeu5/dist-rl-testing/policies"
	"go.etcd.io/raft/v3"
)

// returns a set of hierarchies for the given name.
// If the set name is a single hierarchy then it returns that specific hierarchy
func getHierarchySet(hSet string) []common.HierarchySet {
	if !strings.Contains(strings.ToLower(hSet), "set") {
		hierarchy := GetHierarchy(hSet)
		out := make([]common.HierarchySet, 0)
		for i := len(hierarchy) - 1; i >= 0; i-- {
			out = append(out, common.HierarchySet{
				Name:       fmt.Sprintf("%s[%d]", hSet, len(hierarchy)-i),
				Predicates: hierarchy[i:],
			})
		}
		return out
	}
	var hierarchies []string
	switch hSet {
	case "set1":
		hierarchies = []string{
			"OneInTerm4", "AllInTerm5", "TermDiff2",
			"MinCommit2", "LeaderInTerm4", "OneLeaderOneCandidate",
			"AtLeastOneCommitInTerm2", "LogGap2", "LogCommitGap3",
		}
	default:
		return []common.HierarchySet{}
	}
	out := make([]common.HierarchySet, len(hierarchies))
	for i, h := range hierarchies {
		out[i] = common.HierarchySet{
			Name:       h,
			Predicates: GetHierarchy(h),
		}
	}
	return out
}

func GetHierarchy(name string) []policies.Predicate {
	switch name {
	case "OneInTerm4":
		return oneInTerm4()
	case "AllInTerm5":
		return allInTerm5()
	case "TermDiff2":
		return termDiff2()
	case "MinCommit2":
		return minCommit2()
	case "LeaderInTerm4":
		return leaderInTerm4()
	case "OneLeaderOneCandidate":
		return oneLeaderOneCandidate()
	case "AtLeastOneCommitInTerm2":
		return atleastOneCommitInTerm2()
	case "LogGap2":
		return logGap2()
	case "LogCommitGap3":
		return commitGap3()
	}
	return []policies.Predicate{}
}

func oneInTerm4() []policies.Predicate {
	return []policies.Predicate{
		{Name: "AnyInTerm2", Check: AnyInTerm(2)},
		{Name: "AnyInTerm3", Check: AnyInTerm(3)},
		{Name: "AnyInTerm4", Check: AnyInTerm(4)},
	}
}

func allInTerm5() []policies.Predicate {
	return []policies.Predicate{
		{Name: "AnyInTerm3", Check: AnyInTermAtLeast(3)},
		{Name: "AnyInTerm4", Check: AnyInTermAtLeast(5)},
		{Name: "AllInTerm5", Check: AllInTermAtLeast(5)},
	}
}

func termDiff2() []policies.Predicate {
	return []policies.Predicate{
		{Name: "TermDiff2", Check: TermDiff(2)},
	}
}

func minCommit2() []policies.Predicate {
	return []policies.Predicate{
		{Name: "MinCommit1", Check: AnyWithCommit(1)},
		{Name: "MinCommit1WithLeader", Check: AnyWithCommit(1).And(InState(raft.StateLeader))},
		{Name: "MinCommit2", Check: AnyWithCommit(2)},
	}
}

func leaderInTerm4() []policies.Predicate {
	return []policies.Predicate{
		{Name: "AnyInTerm3", Check: AnyInTerm(3)},
		{Name: "AllInTerm3", Check: AllInTerm(3)},
		{Name: "LeaderInTerm4", Check: LeaderElectedPredicateStateWithTerm(4)},
	}
}

func oneLeaderOneCandidate() []policies.Predicate {
	return []policies.Predicate{
		{Name: "Leader", Check: InState(raft.StateLeader)},
		{Name: "TermDiff1", Check: TermDiff(1)},
		{Name: "OneLeaderOneCandidate", Check: InState(raft.StateLeader).And(InState(raft.StateCandidate))},
	}
}

func atleastOneCommitInTerm2() []policies.Predicate {
	return []policies.Predicate{
		{Name: "AnyInTerm2", Check: AnyInTerm(2)},
		{Name: "LeaderInTerm2", Check: LeaderElectedPredicateStateWithTerm(2)},
		{Name: "AtLeastOneCommitInTerm2", Check: AtLeastOneLogNotEmptyTerm(2)},
	}
}

func logGap2() []policies.Predicate {
	return []policies.Predicate{
		{Name: "LogDiff1", Check: MinLogLengthDiff(1)},
		{Name: "LogDiff2", Check: MinLogLengthDiff(2)},
	}
}

func commitGap3() []policies.Predicate {
	return []policies.Predicate{
		{Name: "LogDiff1", Check: MinLogLengthDiff(1)},
		{Name: "LogDiff2", Check: MinLogLengthDiff(2)},
		{Name: "LogDiff3", Check: MinLogLengthDiff(3)},
		{Name: "LogCommitGap3", Check: MinLogCommittedLengthDiff(3)},
	}
}
