package etcd

import (
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
		BoundaryPredicate:     TermBound(3),
	}).GetConstructor(raftEnvConstructor)

	cmp.AddAnalysis("Coverage", analysis.NewCovertAnalyzerConstructor(painter), analysis.NewCoverageComparatorConstructor(flags.SavePath))

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
