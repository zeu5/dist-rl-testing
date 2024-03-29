package etcd

import (
	"fmt"
	"strings"

	"github.com/zeu5/dist-rl-testing/policies"
	"go.etcd.io/raft/v3"
)

type hierarchySet struct {
	Name       string
	Predicates []policies.Predicate
}

// returns a set of hierarchies for the given name.
// If the set name is a single hierarchy then it returns that specific hierarchy
func getHierarchySet(hSet string) []hierarchySet {
	if !strings.Contains(strings.ToLower(hSet), "set") {
		hierarchy := GetHierarchy(hSet)
		out := make([]hierarchySet, 0)
		for i := len(hierarchy) - 1; i >= 0; i-- {
			out = append(out, hierarchySet{
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
			"OneInTerm4", "AllInTerm3", "TermDiff2",
			"MinCommit2", "LeaderInTerm4", "OneLeaderOneCandidate",
		}
	default:
		return []hierarchySet{}
	}
	out := make([]hierarchySet, len(hierarchies))
	for i, h := range hierarchies {
		out[i] = hierarchySet{
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
	case "AllInTerm3":
		return allInTerm3()
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

func oneInTerm4() []policies.Predicate {
	return []policies.Predicate{
		// {Name: "AnyInTerm2", Check: AllInTerm(2)},
		// {Name: "AllInTerm2", Check: AllInTerm(2)},
		{Name: "AnyInTerm4", Check: AnyInTerm(4)},
	}
}

func allInTerm3() []policies.Predicate {
	return []policies.Predicate{
		{Name: "AllInTerm3", Check: AllInTerm(3)},
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
		{Name: "AllInTerm3", Check: AllInTerm(3)},
		{Name: "LeaderInTerm4", Check: LeaderElectedPredicateStateWithTerm(4)},
	}
}

func oneLeaderOneCandidate() []policies.Predicate {
	return []policies.Predicate{
		{Name: "OneLeaderOneCandidate", Check: InState(raft.StateLeader).And(InState(raft.StateCandidate))},
	}
}
