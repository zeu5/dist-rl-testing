package analysis

import "github.com/zeu5/dist-rl-testing/core"

type NoOpComparator struct {
}

var _ core.Comparator = &NoOpComparator{}

func NewNoOpComparator() *NoOpComparator {
	return &NoOpComparator{}
}

func (n *NoOpComparator) Compare(_ []string, _ []core.DataSet) {
}

type NoOpComparatorConstructor struct {
}

var _ core.ComparatorConstructor = &NoOpComparatorConstructor{}

func NewNoOpComparatorConstructor() *NoOpComparatorConstructor {
	return &NoOpComparatorConstructor{}
}

func (n *NoOpComparatorConstructor) NewComparator(_ int) core.Comparator {
	return NewNoOpComparator()
}
