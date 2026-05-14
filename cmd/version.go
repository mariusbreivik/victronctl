package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(
				cmd.OutOrStdout(),
				"victronctl %s\ncommit: %s\nbuilt: %s\ngo: %s\nos/arch: %s/%s\n",
				version,
				commit,
				date,
				runtime.Version(),
				runtime.GOOS,
				runtime.GOARCH,
			)
		},
	}
}

func init() {
	rootCmd.Version = version
	rootCmd.AddCommand(newVersionCommand())
}
