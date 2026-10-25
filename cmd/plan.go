package cmd

import (
	"github.com/spf13/cobra"

	"github.com/rossedman/gh-fleet/internal/app"
)

// planCmd is a dry run of apply.
var planCmd = &cobra.Command{
	Use:   "plan [campaign]",
	Short: "Run the change in fresh clones and show per-repo diffs, without pushing",
	Long: `Clone every target, run the change script and print the unified diff each repo
would get. Nothing is pushed and nothing is written to GitHub. Repos where the
script changes nothing are reported as noop.

The script sees FLEET_DRY_RUN=true.`,
	Example: `  gh fleet plan --repo acme/api
  gh fleet plan --json --jq '[.[] | select(.status == "changed") | .repo]'`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(a *app.App) error { return a.Plan(cmd.Context(), common(args)) })
	},
}

// init registers the plan command and its flags.
func init() {
	addCampaignFlags(planCmd)
	addRunFlags(planCmd)
	addOutputFlags(planCmd)
	rootCmd.AddCommand(planCmd)
}
