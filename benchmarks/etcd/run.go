package etcd

import (
	"errors"
	"strings"

	"github.com/zeu5/dist-rl-testing/analysis"
	"github.com/zeu5/dist-rl-testing/benchmarks/common"
	"github.com/zeu5/dist-rl-testing/core"
	"github.com/zeu5/dist-rl-testing/policies"
)

func PreparePureCoverageComparison(flags *common.Flags) *core.ParallelComparison {
	cmp := core.NewParallelComparison()

	colors := []core.KVPainter{
		ColorState(),
		ColorCommit(),
		ColorLeader(),
		ColorVote(),
		ColorIndex(),
		ColorBoundedTerm(3),
		ColorBoundedLog(3),
	}

	raftEnvConstructor := NewPartitionEnvironmentConstructor(RaftEnvironmentConfig{
		NumNodes:          flags.NumNodes,
		ElectionTick:      10,
		HeartbeatTick:     2,
		Requests:          flags.Requests,
		SnapshotFrequency: 0,
	})
	painter := core.NewComposedPainter(colors...).Painter()
	partitionEnvConstructor := (&core.PEnvironmentConfig{
		TicksBetweenPartition: flags.TicksBetweenPartition,
		Painter:               painter,
		NumNodes:              flags.NumNodes,
		MaxMessagesPerTick:    100,
		StaySameUpto:          flags.StaySameUpto,
		WithCrashes:           flags.WithCrashes,
		MaxCrashedNodes:       flags.MaxCrashActions,
		BoundaryPredicate:     TermBound(9),
	}).GetConstructor(raftEnvConstructor)

	cmp.AddAnalysis("Coverage", analysis.NewColorAnalyzerConstructor(painter), analysis.NewColorComparatorConstructor(flags.SavePath))

	cmp.AddExperiment(&core.ParallelExperiment{
		Name:        "Random",
		Environment: partitionEnvConstructor,
		Policy:      &policies.RandomPolicyConstructor{},
	})
	cmp.AddExperiment(&core.ParallelExperiment{
		Name:        "BonusMax",
		Environment: partitionEnvConstructor,
		Policy:      policies.NewBonusPolicyGreedyRewardConstructor(0.1, 0.99, 0.05),
	})
	cmp.AddExperiment(&core.ParallelExperiment{
		Name:        "NegRLVisits",
		Environment: partitionEnvConstructor,
		Policy:      policies.NewSoftMaxNegFreqPolicyConstructor(0.3, 0.7, 1),
	})
	cmp.AddExperiment(&core.ParallelExperiment{
		Name:        "NegRL",
		Environment: partitionEnvConstructor,
		Policy:      policies.NewSoftMaxNegPolicyConstructor(0.1, 0.99, 0),
	})
	return cmp
}

type hierarchySet struct {
	Name       string
	Predicates []policies.Predicate
}

func getHierarchySet(hSet string) []hierarchySet {
	if !strings.Contains(strings.ToLower(hSet), "set") {
		return []hierarchySet{
			{Name: hSet, Predicates: GetHierarchy(hSet)},
		}
	}
	switch hSet {
	case "set1":
		return []hierarchySet{
			{Name: "OneInTerm3", Predicates: GetHierarchy("OneInTerm3")},
			{Name: "AllInTerm2", Predicates: GetHierarchy("AllInTerm2")},
			{Name: "TermDiff2", Predicates: GetHierarchy("TermDiff2")},
			{Name: "MinCommit2", Predicates: GetHierarchy("MinCommit2")},
		}
	default:
		return []hierarchySet{}
	}
}

func PrepareHierarchyComparison(flags *common.Flags, hSet string) (*core.ParallelComparison, error) {

	hierarchies := getHierarchySet(hSet)

	if len(hierarchies) == 0 {
		return nil, errors.New("no hierarcies that match the criterion")
	}

	cmp := core.NewParallelComparison()

	colors := []core.KVPainter{
		ColorState(),
		ColorCommit(),
		ColorLeader(),
		ColorVote(),
		ColorIndex(),
		ColorBoundedTerm(3),
		ColorBoundedLog(3),
	}

	raftEnvConstructor := NewPartitionEnvironmentConstructor(RaftEnvironmentConfig{
		NumNodes:          flags.NumNodes,
		ElectionTick:      10,
		HeartbeatTick:     2,
		Requests:          flags.Requests,
		SnapshotFrequency: 0,
	})
	painter := core.NewComposedPainter(colors...).Painter()
	partitionEnvConstructor := (&core.PEnvironmentConfig{
		TicksBetweenPartition: flags.TicksBetweenPartition,
		Painter:               painter,
		NumNodes:              flags.NumNodes,
		MaxMessagesPerTick:    100,
		StaySameUpto:          flags.StaySameUpto,
		WithCrashes:           flags.WithCrashes,
		MaxCrashedNodes:       flags.MaxCrashActions,
		BoundaryPredicate:     TermBound(9),
	}).GetConstructor(raftEnvConstructor)

	for _, h := range hierarchies {
		cmp.AddAnalysis(
			"HierarchyCoverage_"+h.Name,
			analysis.NewPredicateAnalyzerConstructor(painter, h.Predicates),
			analysis.NewPredicateComparatorConstructor(h.Name, flags.SavePath),
		)
		cmp.AddExperiment(&core.ParallelExperiment{
			Name:        "PredHRL_" + h.Name,
			Environment: partitionEnvConstructor,
			Policy: policies.NewHierarchyPolicyConstructor(
				0.1, 0.99, 0, true,
				h.Predicates...,
			),
		})
	}

	cmp.AddExperiment(&core.ParallelExperiment{
		Name:        "Random",
		Environment: partitionEnvConstructor,
		Policy:      &policies.RandomPolicyConstructor{},
	})
	cmp.AddExperiment(&core.ParallelExperiment{
		Name:        "BonusMax",
		Environment: partitionEnvConstructor,
		Policy:      policies.NewBonusPolicyGreedyRewardConstructor(0.1, 0.99, 0.05),
	})
	cmp.AddExperiment(&core.ParallelExperiment{
		Name:        "NegRLVisits",
		Environment: partitionEnvConstructor,
		Policy:      policies.NewSoftMaxNegFreqPolicyConstructor(0.3, 0.7, 1),
	})
	return cmp, nil
}
