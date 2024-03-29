package common

import "github.com/zeu5/dist-rl-testing/policies"

type HierarchySet struct {
	Name       string
	Predicates []policies.Predicate
}
