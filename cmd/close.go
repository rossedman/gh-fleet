package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/rossedman/gh-fleet/internal/app"
)

// closeCmd closes every open campaign PR.
var closeCmd = &cobra.Command{
	Use:   "close [campaign]",
	Short: "Close all open campaign PRs and delete their branches",
	Long: `Close every open PR the campaign opened and delete its branch. Merged and
already closed PRs are left alone. Asks for confirmation unless --yes is
passed.`,
	Example: `  gh fleet close
  gh fleet close --repo acme/api --comment "Superseded by #42" --yes`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		o := app.CloseOptions{Common: common(args), Comment: viper.GetString("comment"), KeepBranches: viper.GetBool("keep-branches")}
		return run(cmd, func(a *app.App) error { return a.Close(cmd.Context(), o) })
	},
}

// init registers the close command and its flags.
func init() {
	addCampaignFlags(closeCmd)
	addWriteFlags(closeCmd)
	addOutputFlags(closeCmd)
	closeCmd.Flags().String("comment", "", "comment to post on each PR before closing it")
	closeCmd.Flags().Bool("keep-branches", false, "close PRs but keep their branches")
	rootCmd.AddCommand(closeCmd)
}
