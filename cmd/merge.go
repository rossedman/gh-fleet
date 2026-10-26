package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/rossedman/gh-fleet/internal/app"
)

// mergeCmd merges the campaign PRs that are ready.
var mergeCmd = &cobra.Command{
	Use:   "merge [campaign]",
	Short: "Merge campaign PRs that are approved and passing, and say what the rest wait on",
	Long: `Merge every open campaign PR that is ready: GitHub reports it mergeable, its
checks pass and it is approved. The others are listed with what they wait on:
a review, running or failing checks, a conflict, or a base that moved ahead.

By default a PR needs an approving review and passing checks even where the
repo doesn't require them; --allow-unreviewed and --allow-no-checks relax
that. Merging uses the head commit gh fleet checked, so a PR that gets a new
push in the meantime is left for the next run.

--auto turns on auto-merge for PRs that aren't ready yet, so GitHub merges
each one when its requirements pass. --update-branch merges the base into
PRs that are behind it. --max rolls a campaign out in batches. Branches of
merged PRs are deleted unless --keep-branches is passed.

--dry-run shows what would happen. Asks for confirmation unless --yes is
passed.`,
	Example: `  gh fleet merge --dry-run
  gh fleet merge --max 10 --yes
  gh fleet merge --auto --update-branch --method squash`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := common(cmd, args)
		if err != nil {
			return err
		}
		o := app.MergeOptions{
			Common:          c,
			Method:          viper.GetString("method"),
			Auto:            viper.GetBool("auto"),
			UpdateBranch:    viper.GetBool("update-branch"),
			Max:             viper.GetInt("max"),
			AllowUnreviewed: viper.GetBool("allow-unreviewed"),
			AllowNoChecks:   viper.GetBool("allow-no-checks"),
			KeepBranches:    viper.GetBool("keep-branches"),
			DryRun:          viper.GetBool("dry-run"),
		}
		return run(cmd, func(a *app.App) error { return a.Merge(cmd.Context(), o) })
	},
}

// init registers the merge command and its flags.
func init() {
	addCampaignFlags(mergeCmd)
	addWriteFlags(mergeCmd)
	addOutputFlags(mergeCmd)
	mergeCmd.Flags().String("method", "squash", "merge method: merge, squash or rebase")
	mergeCmd.Flags().Bool("auto", false, "turn on auto-merge for PRs that aren't ready yet")
	mergeCmd.Flags().Bool("update-branch", false, "merge the base into PRs that are behind it")
	mergeCmd.Flags().Int("max", 0, "merge or auto-merge at most this many PRs (0: no limit)")
	mergeCmd.Flags().Bool("allow-unreviewed", false, "merge PRs in repos that don't require a review")
	mergeCmd.Flags().Bool("allow-no-checks", false, "merge PRs whose head has no checks")
	mergeCmd.Flags().Bool("keep-branches", false, "keep the branches of merged PRs")
	mergeCmd.Flags().Bool("dry-run", false, "show what would happen without changing anything")
	rootCmd.AddCommand(mergeCmd)
}
