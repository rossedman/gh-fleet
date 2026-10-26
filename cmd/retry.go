package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/rossedman/gh-fleet/internal/app"
)

// retryCmd re-runs failed repos.
var retryCmd = &cobra.Command{
	Use:   "retry [campaign]",
	Short: "Run apply again for repos that failed or have no PR",
	Long: `Run apply again for the repos whose last run failed, never ran, or changed
but were not pushed (for example because limits.maxOpenPRs was reached).
Repos with a PR, merged or closed PRs and no-op repos are left alone.`,
	Example: `  gh fleet retry
  gh fleet retry --yes`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := common(cmd, args)
		if err != nil {
			return err
		}
		o := app.ApplyOptions{Common: c, Reopen: viper.GetBool("reopen"), CloseNoop: viper.GetBool("close-noop")}
		return run(cmd, func(a *app.App) error { return a.Retry(cmd.Context(), o) })
	},
}

// init registers the retry command and its flags.
func init() {
	addCampaignFlags(retryCmd)
	addRunFlags(retryCmd)
	addWriteFlags(retryCmd)
	addOutputFlags(retryCmd)
	retryCmd.Flags().Bool("reopen", false, "reopen campaign PRs that were closed without merging")
	retryCmd.Flags().Bool("close-noop", false, "close open PRs whose change is already on the base branch")
	rootCmd.AddCommand(retryCmd)
}
