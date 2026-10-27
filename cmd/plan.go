package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/rossedman/gh-fleet/internal/app"
	"github.com/rossedman/gh-fleet/internal/state"
)

// planCmd is a dry run of apply.
var planCmd = &cobra.Command{
	Use:   "plan [campaign]",
	Short: "Run the change in fresh clones and show per-repo diffs, without pushing",
	Long: `Clone every target, run the change script and print the unified diff each repo
would get. Nothing is pushed and nothing is written to GitHub. Repos where the
script changes nothing are reported as noop.

Diffs are printed at the end, each distinct diff once with the repos that
get it, so a change made the same way in 300 repos is read once. Blob hashes
and hunk line numbers don't count as differences. --ungrouped prints each
repo's diff as it finishes instead.

With limits.waves, plan takes one wave at a time, like apply: the wave apply
would roll out now. --wave plans another wave, and --all-waves plans every
target. --repo plans the repos you name, whatever their wave.

The script sees FLEET_DRY_RUN=true.`,
	Example: `  gh fleet plan --repo acme/api
  gh fleet plan --wave 3
  gh fleet plan --json repo,status --jq '[.[] | select(.status == "changed") | .repo]'`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := common(cmd, args)
		if err != nil {
			return err
		}
		o := app.PlanOptions{Common: c, Ungrouped: viper.GetBool("ungrouped"), Wave: viper.GetInt("wave"), AllWaves: viper.GetBool("all-waves")}
		return run(cmd, func(a *app.App) error { return a.Plan(cmd.Context(), o) })
	},
}

// init registers the plan command and its flags.
func init() {
	addCampaignFlags(planCmd)
	addRunFlags(planCmd)
	addOutputFlags(planCmd, []state.RepoState{})
	planCmd.Flags().Bool("ungrouped", false, "print each repo's diff as it finishes instead of grouping identical diffs")
	planCmd.Flags().Int("wave", 0, "plan this `wave` of limits.waves (default the one apply would roll out now)")
	planCmd.Flags().Bool("all-waves", false, "plan every target, whatever its wave")
	rootCmd.AddCommand(planCmd)
}
