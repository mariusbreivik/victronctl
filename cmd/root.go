package cmd

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:           "victronctl",
	Short:         "CLI for Victron Energy VRM workflows",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() error {
	return rootCmd.Execute()
}
