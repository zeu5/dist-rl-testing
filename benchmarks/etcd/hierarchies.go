package etcd

import "github.com/zeu5/dist-rl-testing/policies"

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
	}
	return []policies.Predicate{}
}

func oneInTerm3() []policies.Predicate {
	return []policies.Predicate{
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
