package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/rossedman/gh-fleet/internal/app"
	"github.com/rossedman/gh-fleet/internal/state"
)

// applyCmd runs the change and opens or updates PRs.
var applyCmd = &cobra.Command{
	Use:   "apply [campaign]",
	Short: "Run the change, push branches, and open or update PRs",
	Long: `Run the change in every target, push the campaign branch and open a PR, or
update the existing one. Running apply again never duplicates PRs: it updates
the ones that changed.

A merged PR's repo is skipped. A PR closed without merging is left alone
unless --reopen is passed. When the change is already on the base branch and
a PR is still open, --close-noop closes it.

A repo that the last plan or apply (within a day) found nothing to change in
is skipped without cloning, as long as it has no campaign PR, its base branch
hasn't moved and the campaign's files are the same. --recheck runs every repo,
for a change that depends on something outside the repo, such as the latest
version of a tool.

apply asks for confirmation before pushing anything unless --yes is passed.`,
	Example: `  gh fleet apply
  gh fleet apply --repo acme/api --yes
  gh fleet apply --close-noop`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := common(cmd, args)
		if err != nil {
			return err
		}
		o := app.ApplyOptions{Common: c, Reopen: viper.GetBool("reopen"), CloseNoop: viper.GetBool("close-noop"), Recheck: viper.GetBool("recheck")}
		return run(cmd, func(a *app.App) error { return a.Apply(cmd.Context(), o) })
	},
}

// init registers the apply command and its flags.
func init() {
	addCampaignFlags(applyCmd)
	addRunFlags(applyCmd)
	addWriteFlags(applyCmd)
	addOutputFlags(applyCmd, []state.RepoState{})
	applyCmd.Flags().Bool("reopen", false, "reopen campaign PRs that were closed without merging")
	applyCmd.Flags().Bool("close-noop", false, "close open PRs whose change is already on the base branch")
	applyCmd.Flags().Bool("recheck", false, "run the change even in repos the last run found nothing to change in")
	rootCmd.AddCommand(applyCmd)
}
