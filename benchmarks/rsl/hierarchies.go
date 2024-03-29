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
		hierarchies = []string{}
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
	return []policies.Predicate{}
}
