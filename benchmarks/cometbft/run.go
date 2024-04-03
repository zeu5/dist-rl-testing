package cometbft

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/zeu5/dist-rl-testing/analysis"
	"github.com/zeu5/dist-rl-testing/benchmarks/common"
	"github.com/zeu5/dist-rl-testing/core"
	"github.com/zeu5/dist-rl-testing/policies"
)

func PreparePureCoverageComparison(flags *common.Flags) (*core.ParallelComparison, error) {
	cmp := core.NewParallelComparison()

	colors := []core.KVPainter{
		ColorHeight(),
		ColorStep(),
		ColorProposal(),
		ColorCurRoundVotes(),
		ColorRound(),
	}

	exePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("error fetching cur directory: %s", err)
	}
	cometBinaryPath := path.Join(filepath.Dir(exePath), "benchmarks", "cometbft", "cometbft")
	workDir := path.Join(flags.SavePath, "work")

	os.MkdirAll(workDir, 0755)

	raftEnvConstructor := NewCometEnvConstructor(&CometClusterConfig{
		NumNodes:                flags.NumNodes,
		NumRequests:             flags.Requests,
		CometBinaryPath:         cometBinaryPath,
		BaseInterceptListenPort: 2023,
		BaseRPCPort:             26856,
		BaseP2PPort:             26656,
		InterceptServerPort:     7074,
		BaseWorkingDir:          workDir,
		CreateEmptyBlocks:       true,
		TickDuration:            50 * time.Millisecond,
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
		BoundaryPredicate:     HeightBound(3),
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
	return cmp, nil
}
