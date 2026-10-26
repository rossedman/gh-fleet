package cmd

import (
	"github.com/spf13/cobra"

	"github.com/rossedman/gh-fleet/internal/app"
	"github.com/rossedman/gh-fleet/internal/target"
)

// targetsCmd lists the repos a campaign will touch.
var targetsCmd = &cobra.Command{
	Use:   "targets [campaign]",
	Short: "List the repos a campaign will touch and why each matched",
	Long: `Resolve the campaign's targets and list every repo it will touch, with the
filters it matched. Repos without push access are listed as skipped.

Org listings are cached for 10 minutes.`,
	Example: `  gh fleet targets
  gh fleet targets --json --jq '.targets[].repo | .owner + "/" + .name'`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(a *app.App) error { return a.Targets(cmd.Context(), common(args)) })
	},
}

// init registers the targets command and its flags.
func init() {
	addCampaignFlags(targetsCmd)
	addOutputFlags(targetsCmd, target.Result{})
	rootCmd.AddCommand(targetsCmd)
}
