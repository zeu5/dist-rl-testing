package rsl

import (
	"fmt"
	"strings"

	"github.com/zeu5/dist-rl-testing/benchmarks/common"
	"github.com/zeu5/dist-rl-testing/policies"
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
			"AnyBallot3", "AnyDecided3", "AllBallot3",
			"EntryBallot2", "PrimaryInBallot2",
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
	case "AnyBallot3":
		return anyBallot3()
	case "AnyDecided3":
		return anyDecided3()
	case "AllBallot3":
		return allBallot3()
	case "EntryBallot2":
		return entryBallot2()
	case "PrimaryInBallot2":
		return primaryInBallot2()
	}
	return []policies.Predicate{}
}

func anyBallot3() []policies.Predicate {
	return []policies.Predicate{
		{Name: "AnyBallot3", Check: AnyInBallot(3)},
	}
}

func allBallot3() []policies.Predicate {
	return []policies.Predicate{
		{Name: "AllBallot3", Check: AllAtLeastBallot(3)},
	}
}

func anyDecided3() []policies.Predicate {
	return []policies.Predicate{
		{Name: "AnyDecided3", Check: AnyDecided(3)},
	}
}

func entryBallot2() []policies.Predicate {
	return []policies.Predicate{
		{Name: "PrimaryInBallot2", Check: InStateAndBallot(StateStablePrimary, 2)},
		{Name: "EntryBallot2", Check: EntryInBallot(2)},
	}
}

func primaryInBallot2() []policies.Predicate {
	return []policies.Predicate{
		{Name: "AnyInBallot2", Check: AnyInBallot(2)},
		{Name: "PrimaryInBallot2", Check: InStateAndBallot(StateStablePrimary, 2)},
	}
}
