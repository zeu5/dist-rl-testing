package cmd

import (
	"context"
	"os"
	"os/signal"

	"github.com/spf13/cobra"
	"github.com/zeu5/dist-rl-testing/benchmarks/etcd"
	"github.com/zeu5/dist-rl-testing/core"
)

func EtcdCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "etcd",
		Short: "Run etcd benchmarks",
	}

	cmd.AddCommand(
		etcdPureCovCommand(),
		etcdHierarchyCommand(),
	)

	return cmd
}

func etcdPureCovCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cov",
		Short: "Run etcd benchmarks for purecov",
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

			cmp := etcd.PreparePureCoverageComparison(flags)
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

func etcdHierarchyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hierarchy",
		Args:  cobra.ExactArgs(1),
		Short: "Run etcd benchmarks for purecov",
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

			cmp, err := etcd.PrepareHierarchyComparison(flags, args[1])
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
