package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/rossedman/gh-fleet/internal/app"
)

// statusCmd reports campaign progress.
var statusCmd = &cobra.Command{
	Use:   "status [campaign]",
	Short: "Show every target's PR, state, checks, reviews and last error",
	Long: `Refresh every target's PR from GitHub, in one batched GraphQL query per 50
repos, and print a table with the PR link, state, check rollup, review decision
and the last error recorded locally.

GitHub is the source of truth: PRs are found by branch and by the hidden
campaign marker, so status is correct even after the local state file is
deleted. The remaining API quota is shown at the end.

--exit-status sets the exit code from the result, for scripts and CI: 1 if
a repo failed, 8 while any still needs work, and 0 once every repo is
merged, closed, a no-op or skipped.

--format markdown prints a progress report to paste into an issue, a PR or
chat: what needs attention, open PRs and what each waits on, and merged
PRs. --format html prints the same report as a self-contained page.`,
	Example: `  gh fleet status
  gh fleet status pin-actions --json counts --jq '.counts'
  gh fleet status --exit-status && echo "every PR is merged"
  gh fleet status --format markdown | gh issue comment 42 --body-file -
  gh fleet status --format html > report.html`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := common(cmd, args)
		if err != nil {
			return err
		}
		o := app.StatusOptions{Common: c, Format: viper.GetString("format"), ExitStatus: viper.GetBool("exit-status")}
		return run(cmd, func(a *app.App) error { return a.Status(cmd.Context(), o) })
	},
}

// init registers the status command and its flags.
func init() {
	addCampaignFlags(statusCmd)
	addOutputFlags(statusCmd, app.StatusReport{})
	statusCmd.Flags().String("format", "table", "table, or a report to share: markdown or html")
	statusCmd.Flags().Bool("exit-status", false, "exit 1 if a repo failed, or 8 while repos still need work")
	rootCmd.AddCommand(statusCmd)
}
