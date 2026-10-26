package cmd

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/rossedman/gh-fleet/internal/app"
)

// execCmd runs a read-only command across targets.
var execCmd = &cobra.Command{
	Use:   "exec -- <command> [args...]",
	Short: "Run a read-only command across targets and aggregate the output",
	Long: `Clone every target, run a command in it and tally the output lines across
repos. Nothing is pushed and no PRs are opened.

Targets come from ad-hoc flags (--org, --topic, --language, --file, --search,
--repo, --repo-file) when any is given, otherwise from the campaign (--campaign, default
./fleet.yaml). Repos you can read but not push to are included. The command
runs directly, without a shell; wrap it in sh -c for pipes.`,
	Example: `  gh fleet exec --org acme --file Dockerfile -- grep -h '^FROM' Dockerfile
  gh fleet exec --org acme --language Go -- sh -c "grep '^go ' go.mod"
  gh fleet exec --raw -- git log -1 --format=%cs`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := common(cmd, nil)
		if err != nil {
			return err
		}
		c.File = "" // --file is a path filter here, not the campaign file
		o := app.ExecOptions{
			Common:    c,
			Campaign:  viper.GetString("campaign"),
			Argv:      args,
			Orgs:      viper.GetStringSlice("org"),
			Topics:    viper.GetStringSlice("topic"),
			Languages: viper.GetStringSlice("language"),
			Files:     viper.GetStringSlice("file"),
			Search:    viper.GetString("search"),
			Raw:       viper.GetBool("raw"),
			Timeout:   viper.GetDuration("timeout"),
		}
		return run(cmd, func(a *app.App) error { return a.Exec(cmd.Context(), o) })
	},
}

// init registers the exec command and its flags.
func init() {
	execCmd.Flags().String("campaign", "", "take targets from this campaign file or name (default ./fleet.yaml)")
	execCmd.Flags().StringSlice("repo", nil, "target these repos (owner/name, repeatable); with a campaign, limit to them")
	addRepoFileFlags(execCmd, "target the repos listed in this file, one per line (- for stdin, repeatable); with a campaign, limit to them")
	execCmd.Flags().Int("concurrency", 0, "repos processed in parallel (default 8)")
	addRunFlags(execCmd)
	addOutputFlags(execCmd)
	execCmd.Flags().StringSlice("org", nil, "target this org's repos (repeatable)")
	execCmd.Flags().StringSlice("topic", nil, "only repos with any of these topics")
	execCmd.Flags().StringSlice("language", nil, "only repos with any of these primary languages")
	execCmd.Flags().StringSlice("file", nil, "only repos where any of these paths exist")
	execCmd.Flags().String("search", "", "only repos with a hit for this code search query")
	execCmd.Flags().Bool("raw", false, "print each repo's output instead of a tally")
	execCmd.Flags().Duration("timeout", 5*time.Minute, "time limit per repo")
	rootCmd.AddCommand(execCmd)
}
