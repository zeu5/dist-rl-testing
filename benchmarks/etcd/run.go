package etcd

import (
	"errors"
	"os"
	"path"

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
		ColorBoundedTerm(6),
		ColorBoundedLog(6),
	}

	traceRecordPath := path.Join(flags.SavePath, "event-traces")
	if flags.RecordEventTraces {
		os.MkdirAll(traceRecordPath, 0755)
	}

	raftEnvConstructor := NewPartitionEnvironmentConstructor(RaftEnvironmentConfig{
		NumNodes:        flags.NumNodes,
		ElectionTick:    16,
		HeartbeatTick:   4,
		Requests:        flags.Requests,
		TraceRecordPath: traceRecordPath,
		RecordTraces:    flags.RecordEventTraces,
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

	if flags.Debug {
		cmp.AddAnalysis("Debug", analysis.NewPrintDebugAnalyzerConstructor(flags.SavePath, flags.Episodes-10), analysis.NewNoOpComparatorConstructor())
	}
	cmp.AddAnalysis("Errors", analysis.NewErrorAnalyzerConstructor(flags.SavePath), analysis.NewNoOpComparatorConstructor())
	cmp.AddAnalysis("Coverage", analysis.NewColorAnalyzerConstructor(painter), analysis.NewColorComparatorConstructor(flags.SavePath))

	cmp.AddExperiment(&core.ParallelExperiment{
		Name:        "Random",
		Environment: partitionEnvConstructor,
		Policy:      &policies.RandomPolicyConstructor{},
	})
	cmp.AddExperiment(&core.ParallelExperiment{
		Name:        "BonusMax",
		Environment: partitionEnvConstructor,
		Policy:      policies.NewBonusPolicyGreedyRewardConstructor(0.2, 0.95, 0.05),
	})
	cmp.AddExperiment(&core.ParallelExperiment{
		Name:        "NegRLVisits",
		Environment: partitionEnvConstructor,
		Policy:      policies.NewSoftMaxNegFreqPolicyConstructor(0.3, 0.7, 1),
	})
	cmp.AddExperiment(&core.ParallelExperiment{
		Name:        "UCBZero",
		Environment: partitionEnvConstructor,
		Policy: policies.NewUCBZeroPolicyConstructor(policies.UCBZeroParams{
			StateSize:   10000,
			ActionsSize: 100,
			Horizon:     flags.Horizon,
			Episodes:    flags.Episodes,
			Epsilon:     0.05,
			Constant:    0.5,
		}),
	})
	return cmp
}

// Creates a comparison with the predicate hierarchy policy for the set of specified hierarchies
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
		ColorBoundedTerm(6),
		ColorBoundedLog(6),
	}

	raftEnvConstructor := NewPartitionEnvironmentConstructor(RaftEnvironmentConfig{
		NumNodes:        flags.NumNodes,
		ElectionTick:    9,
		HeartbeatTick:   3,
		Requests:        flags.Requests,
		RecordTraces:    flags.RecordEventTraces,
		TraceRecordPath: path.Join(flags.SavePath, "event-traces"),
	})
	painter := core.NewComposedPainter(colors...).Painter()
	partitionEnvConstructor := (&core.PEnvironmentConfig{
		TicksBetweenPartition: flags.TicksBetweenPartition,
		Painter:               painter,
		NumNodes:              flags.NumNodes,
		MaxMessagesPerTick:    100,
		StaySameUpto:          flags.StaySameUpto,
		WithCrashes:           flags.WithCrashes,
		MaxCrashedNodes:       flags.MaxCrashedNodes,
		MaxCrashActions:       flags.MaxCrashActions,
		BoundaryPredicate:     TermBound(9),
	}).GetConstructor(raftEnvConstructor)

	if flags.Debug {
		cmp.AddAnalysis("Debug", analysis.NewPrintDebugAnalyzerConstructor(flags.SavePath, flags.Episodes-10), analysis.NewNoOpComparatorConstructor())
	}
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
				0.2, 0.95, 0.05,
				h.Predicates...,
			),
		})
	}
	cmp.AddAnalysis("Errors", analysis.NewErrorAnalyzerConstructor(flags.SavePath), analysis.NewNoOpComparatorConstructor())
	cmp.AddExperiment(&core.ParallelExperiment{
		Name:        "Random",
		Environment: partitionEnvConstructor,
		Policy:      &policies.RandomPolicyConstructor{},
	})
	cmp.AddExperiment(&core.ParallelExperiment{
		Name:        "BonusMax",
		Environment: partitionEnvConstructor,
		Policy:      policies.NewBonusPolicyGreedyRewardConstructor(0.2, 0.95, 0.05),
	})
	cmp.AddExperiment(&core.ParallelExperiment{
		Name:        "NegRLVisits",
		Environment: partitionEnvConstructor,
		Policy:      policies.NewSoftMaxNegFreqPolicyConstructor(0.3, 0.7, 1),
	})
	return cmp, nil
}
