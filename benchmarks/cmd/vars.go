package cmd

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/zeu5/dist-rl-testing/benchmarks/common"
)

var (
	flags                 *common.Flags = common.DefaultFlags()
	savePath              string
	numNodes              int
	requests              int
	ticksBetweenPartition int
	staySameUpto          int
	withCrashes           bool
	maxCrashedNodes       int
	maxCrashActions       int

	numRuns                int
	episodes               int
	horizon                int
	maxConsecutiveErrors   int
	maxConsecutiveTimeouts int
	episodeTimeout         int
	parallelism            int
)

func AddFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&savePath, "save-path", flags.SavePath, "Path to save results")
	cmd.PersistentFlags().IntVar(&numNodes, "num-nodes", flags.NumNodes, "Number of nodes")
	cmd.PersistentFlags().IntVar(&requests, "requests", flags.Requests, "Number of requests")
	cmd.PersistentFlags().IntVar(&ticksBetweenPartition, "ticks-between-partition", flags.TicksBetweenPartition, "Number of ticks between partitions")
	cmd.PersistentFlags().IntVar(&staySameUpto, "stay-same-upto", flags.StaySameUpto, "Number of ticks to stay the same")
	cmd.PersistentFlags().BoolVar(&withCrashes, "with-crashes", flags.WithCrashes, "Whether to include crashes")
	cmd.PersistentFlags().IntVar(&maxCrashedNodes, "max-crashed-nodes", flags.MaxCrashedNodes, "Maximum number of crashed nodes")
	cmd.PersistentFlags().IntVar(&maxCrashActions, "max-crash-actions", flags.MaxCrashActions, "Maximum number of crash actions")

	cmd.PersistentFlags().IntVar(&numRuns, "num-runs", flags.NumRuns, "Number of runs")
	cmd.PersistentFlags().IntVar(&episodes, "episodes", flags.Episodes, "Number of episodes")
	cmd.PersistentFlags().IntVar(&horizon, "horizon", flags.Horizon, "Horizon")
	cmd.PersistentFlags().IntVar(&maxConsecutiveErrors, "max-consecutive-errors", flags.MaxConsecutiveErrors, "Maximum number of consecutive errors")
	cmd.PersistentFlags().IntVar(&maxConsecutiveTimeouts, "max-consecutive-timeouts", flags.MaxConsecutiveTimeouts, "Maximum number of consecutive timeouts")
	cmd.PersistentFlags().IntVar(&episodeTimeout, "episode-timeout", int(flags.EpisodeTimeout.Seconds()), "Episode timeout")
	cmd.PersistentFlags().IntVar(&parallelism, "parallelism", flags.Parallelism, "Number of parallel runs")
}

func UpdateFlags() {
	flags.SavePath = savePath
	flags.NumNodes = numNodes
	flags.Requests = requests
	flags.TicksBetweenPartition = ticksBetweenPartition
	flags.StaySameUpto = staySameUpto
	flags.WithCrashes = withCrashes
	flags.MaxCrashedNodes = maxCrashedNodes
	flags.MaxCrashActions = maxCrashActions

	flags.NumRuns = numRuns
	flags.Episodes = episodes
	flags.Horizon = horizon
	flags.MaxConsecutiveErrors = maxConsecutiveErrors
	flags.MaxConsecutiveTimeouts = maxConsecutiveTimeouts
	flags.EpisodeTimeout = time.Duration(episodeTimeout) * time.Second
	flags.Parallelism = parallelism
}
