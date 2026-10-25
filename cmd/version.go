package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Build information, set with -ldflags at release time.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// versionCmd prints build information.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version, commit and build date",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "gh-fleet %s (commit %s, built %s)\n", version, commit, date)
		return err
	},
}

// init registers the version command and its flags.
func init() {
	rootCmd.AddCommand(versionCmd)
}
