package cmd

import (
	"github.com/spf13/cobra"

	"github.com/rossedman/gh-fleet/internal/app"
	"github.com/rossedman/gh-fleet/internal/state"
)

// listCmd lists campaigns in local state.
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List campaigns in local state",
	Long: `List every campaign with local state under $XDG_STATE_HOME/gh-fleet, with its
last update, per-status repo counts and campaign file. A listed name can be
passed to any command in place of --file.`,
	Example: `  gh fleet list
  gh fleet list --json name --jq '.[].name'`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return run(cmd, func(a *app.App) error { return a.List(common(nil)) })
	},
}

// init registers the list command and its flags.
func init() {
	addOutputFlags(listCmd, []state.Summary{})
	rootCmd.AddCommand(listCmd)
}
