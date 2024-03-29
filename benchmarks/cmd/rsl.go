package cmd

import (
	"context"
	"os"
	"os/signal"

	"github.com/spf13/cobra"
	"github.com/zeu5/dist-rl-testing/benchmarks/rsl"
	"github.com/zeu5/dist-rl-testing/core"
)

func RSLCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rsl",
		Short: "Run rsl benchmarks",
	}

	cmd.AddCommand(
		rslPureCovCommand(),
		rslHierarchyCommand(),
	)

	return cmd
}

func rslPureCovCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cov",
		Short: "Run rsl benchmarks for purecov",
		Run: func(cmd *cobra.Command, args []string) {
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

			cmp := rsl.PreparePureCoverageComparison(flags)
			cmp.Run(ctx, flags.NumRuns, &core.RunConfig{
				Episodes:                     flags.Episodes,
				Horizon:                      flags.Horizon,
				ThresholdConsecutiveErrors:   flags.MaxConsecutiveErrors,
				ThresholdConsecutiveTimeouts: flags.MaxConsecutiveTimeouts,
				EpisodeTimeout:               flags.EpisodeTimeout,
			}, flags.Parallelism)
			close(doneCh)
		},
	}

	return cmd
}

func rslHierarchyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hierarchy",
		Args:  cobra.ExactArgs(1),
		Short: "Run rsl benchmarks for hierarchy coverage",
		RunE: func(cmd *cobra.Command, args []string) error {
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

			cmp, err := rsl.PrepareHierarchyComparison(flags, args[0])
			if err != nil {
				return err
			}
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
