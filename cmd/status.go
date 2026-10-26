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

--format markdown prints a progress report to paste into an issue, a PR or
chat: what needs attention, open PRs and what each waits on, and merged
PRs. --format html prints the same report as a self-contained page.`,
	Example: `  gh fleet status
  gh fleet status pin-actions --json --jq '.counts'
  gh fleet status --format markdown | gh issue comment 42 --body-file -
  gh fleet status --format html > report.html`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		o := app.StatusOptions{Common: common(args), Format: viper.GetString("format")}
		return run(cmd, func(a *app.App) error { return a.Status(cmd.Context(), o) })
	},
}

// init registers the status command and its flags.
func init() {
	addCampaignFlags(statusCmd)
	addOutputFlags(statusCmd)
	statusCmd.Flags().String("format", "table", "table, or a report to share: markdown or html")
	rootCmd.AddCommand(statusCmd)
}
