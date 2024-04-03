package cmd

import "github.com/spf13/cobra"

func RootCommand() *cobra.Command {
	cmd := &cobra.Command{
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			UpdateFlags()
			flags.Record()
		},
	}
	AddFlags(cmd)

	cmd.AddCommand(
		EtcdCommand(),
		RSLCommand(),
		CometBFTCommand(),
	)

	return cmd
}
