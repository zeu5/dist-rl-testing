package cmd

import (
	"context"
	"os"
	"os/signal"

	"github.com/spf13/cobra"
	"github.com/zeu5/dist-rl-testing/benchmarks/cometbft"
	"github.com/zeu5/dist-rl-testing/core"
)

func CometBFTCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cometbft",
		Short: "Run cometbft benchmarks",
	}

	cmd.AddCommand(
		cometbftPureCovCommand(),
	)

	return cmd
}

func cometbftPureCovCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cov",
		Short: "Run cometbft benchmarks for purecov",
		RunE: func(cmd *cobra.Command, args []string) error {

			cmp, err := cometbft.PreparePureCoverageComparison(flags)
			if err != nil {
				return err
			}

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt) // channel for interrupts from os

			doneCh := make(chan struct{}) // channel for done signal from application

			ctx, cancel := context.WithCancel(context.Background())
			go func() { // start a go-routine
				select { // can wait on multiple channels
				case <-sigCh:
				case <-doneCh:
				}
				cancel()
			}()
			cmp.Run(ctx, flags.NumRuns, &core.RunConfig{
				Episodes:                     flags.Episodes,
				Horizon:                      flags.Horizon,
				ThresholdConsecutiveErrors:   flags.MaxConsecutiveErrors,
				ThresholdConsecutiveTimeouts: flags.MaxConsecutiveTimeouts,
				EpisodeTimeout:               flags.EpisodeTimeout,
			}, flags.Parallelism)
			close(doneCh)
			return nil
		},
	}

	return cmd
}
