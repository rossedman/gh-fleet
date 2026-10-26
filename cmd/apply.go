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
		o := app.ApplyOptions{Common: c, Reopen: viper.GetBool("reopen"), CloseNoop: viper.GetBool("close-noop")}
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
	rootCmd.AddCommand(applyCmd)
}
