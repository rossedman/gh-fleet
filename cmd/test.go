package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/rossedman/gh-fleet/internal/app"
)

// testCmd runs a campaign's change against local test cases.
var testCmd = &cobra.Command{
	Use:   "test [campaign]",
	Short: "Run the change against local test cases, without GitHub",
	Long: `Run the change against the test cases in tests/ next to the campaign file and
check the result. Nothing is cloned and nothing touches GitHub.

Each case is a directory:

  tests/<case>/before/     the repo's files before the change
  tests/<case>/after/      the files the change must produce (omit when the
                           change should leave the repo alone)
  tests/<case>/case.yaml   optional: repo, defaultBranch, language, topics,
                           properties, vars and env to run as, and wantVars,
                           wantTitle or wantError to check

Every step runs, shell steps included, in a git repository holding before/.
The case passes when the files match after/ exactly and a second run changes
nothing. --update writes after/ from what the change produced.

--dir runs the change on a copy of a local directory instead, such as a clone
you already have, and prints the diff, variables and PR title.`,
	Example: `  gh fleet test
  gh fleet test --run expired -v
  gh fleet test --update
  gh fleet test --dir ../billing-api --repo acme/billing-api`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		o := app.TestOptions{
			Common:  common(args),
			Run:     viper.GetString("run"),
			Update:  viper.GetBool("update"),
			Dir:     viper.GetString("dir"),
			Verbose: viper.GetBool("verbose"),
		}
		return run(cmd, func(a *app.App) error { return a.Test(cmd.Context(), o) })
	},
}

// init registers the test command and its flags.
func init() {
	testCmd.Flags().StringP("file", "f", "", "campaign file (default ./fleet.yaml)")
	testCmd.Flags().String("run", "", "only run cases whose name matches this regular expression")
	testCmd.Flags().Bool("update", false, "rewrite each case's after/ from the change's result")
	testCmd.Flags().String("dir", "", "run the change on a copy of this directory and print the diff")
	testCmd.Flags().StringSlice("repo", nil, "with --dir, the owner/name the change runs as (default acme/<dir name>)")
	testCmd.Flags().BoolP("verbose", "v", false, "print each case's step output and PR title")
	addOutputFlags(testCmd)
	rootCmd.AddCommand(testCmd)
}
