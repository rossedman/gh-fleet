// Package cmd holds gh fleet's cobra commands. Commands only parse flags
// and call into internal/app; one command per file.
package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"syscall"

	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/fang"
	"github.com/cli/go-gh/v2/pkg/term"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/rossedman/gh-fleet/internal/app"
	"github.com/rossedman/gh-fleet/internal/ghapi"
	"github.com/rossedman/gh-fleet/internal/render"
	"github.com/rossedman/gh-fleet/internal/ui"
)

// cfgFile is the --config flag.
var cfgFile string

// rootCmd is gh fleet itself.
var rootCmd = &cobra.Command{
	Use:   "fleet",
	Short: "Apply one change across many repositories and track the PRs",
	Long: `gh fleet applies one change across many repositories and tracks the resulting
pull requests to completion.

A change is described once as a campaign: which repos to target, what to run in
each, and how to open the PR. Re-running a campaign updates its PRs instead of
duplicating them.`,
	Example: `  gh fleet init pin-actions
  gh fleet targets
  gh fleet plan --repo acme/api
  gh fleet apply
  gh fleet status`,
	Annotations:   map[string]string{cobra.CommandDisplayNameAnnotation: "gh fleet"},
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		if err := initConfig(cmd); err != nil {
			return err
		}
		return checkJSONFields(cmd)
	},
}

// init registers the persistent --config flag.
func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default $XDG_CONFIG_HOME/gh-fleet/config.yaml)")
	rootCmd.SetFlagErrorFunc(flagError)
}

// jsonFieldsKey is the annotation holding a command's JSON fields.
const jsonFieldsKey = "jsonFields"

// flagError lists the JSON fields when --json is given none, as gh does.
func flagError(cmd *cobra.Command, err error) error {
	if err.Error() == "flag needs an argument: --json" {
		return fmt.Errorf("Specify one or more comma-separated fields for `--json`:\n  %s", //nolint:staticcheck // gh's wording
			strings.ReplaceAll(cmd.Annotations[jsonFieldsKey], ",", "\n  "))
	}
	return err
}

// checkJSONFields rejects --json fields the command doesn't output.
func checkJSONFields(cmd *cobra.Command) error {
	f := cmd.Flags().Lookup("json")
	if f == nil || !f.Changed {
		return nil
	}
	known := strings.Split(cmd.Annotations[jsonFieldsKey], ",")
	for _, name := range jsonFields(f.Value.String()) {
		if !slices.Contains(known, name) {
			return fmt.Errorf("Unknown JSON field: %q\nAvailable fields:\n  %s", name, strings.Join(known, "\n  ")) //nolint:staticcheck // gh's wording
		}
	}
	return nil
}

// jsonFields splits a --json value into field names.
func jsonFields(s string) []string {
	var out []string
	for _, f := range strings.Split(s, ",") {
		if f = strings.TrimSpace(f); f != "" {
			out = append(out, f)
		}
	}
	return out
}

// Root returns the root command, for documentation generation and tests.
func Root() *cobra.Command { return rootCmd }

// Execute runs gh fleet. The first Ctrl-C stops new repos from starting
// and lets in-flight ones finish; a second one exits immediately.
func Execute() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sig := make(chan os.Signal, 2)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	interrupted := make(chan struct{})
	go func() {
		<-sig
		close(interrupted)
		fmt.Fprintln(os.Stderr, "\nStopping: finishing in-flight repos and saving state. Press Ctrl-C again to quit now.")
		cancel()
		<-sig
		os.Exit(exitInterrupt)
	}()
	err := fang.Execute(ctx, rootCmd, fang.WithVersion(version), fang.WithCommit(commit), fang.WithoutManpage(),
		fang.WithErrorHandler(func(w io.Writer, _ fang.Styles, err error) { printError(w, err) }))
	stopped := false
	select {
	case <-interrupted:
		stopped = true
	default:
	}
	os.Exit(exitCode(err, stopped))
}

// initConfig loads the config file and environment into viper and binds
// the running command's flags, so precedence is flag > env > file.
func initConfig(cmd *cobra.Command) error {
	v := viper.GetViper()
	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		dir := os.Getenv("XDG_CONFIG_HOME")
		if dir == "" {
			if home, err := os.UserHomeDir(); err == nil {
				dir = filepath.Join(home, ".config")
			}
		}
		v.AddConfigPath(filepath.Join(dir, "gh-fleet"))
		if home, err := os.UserHomeDir(); err == nil {
			v.AddConfigPath(home)
		}
		v.AddConfigPath(".")
		v.SetConfigName("config")
		v.SetConfigType("yaml")
	}
	v.SetEnvPrefix("GH_FLEET")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()
	if err := v.ReadInConfig(); err != nil {
		var nf viper.ConfigFileNotFoundError
		if !errors.As(err, &nf) && (cfgFile != "" || !errors.Is(err, os.ErrNotExist)) {
			return fmt.Errorf("read config: %w", err)
		}
	}
	var bindErr error
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if f.Name == "help" || f.Name == "config" {
			return
		}
		if err := v.BindPFlag(f.Name, f); err != nil && bindErr == nil {
			bindErr = err
		}
	})
	return bindErr
}

// newApp builds the App for a command; tests replace it.
var newApp = func(cmd *cobra.Command) (*app.App, error) {
	t := term.FromEnv()
	out := cmd.OutOrStdout()
	errOut := cmd.ErrOrStderr()
	color := ui.ColorEnabled(t.IsTerminalOutput())
	if color {
		out = colorprofile.NewWriter(out, os.Environ())
		errOut = colorprofile.NewWriter(errOut, os.Environ())
	}
	interactive := t.IsTerminalOutput() && isTerminal(os.Stdin) && os.Getenv("GH_PROMPT_DISABLED") == ""
	errTTY := isTerminal(os.Stderr) || os.Getenv("GH_FORCE_TTY") != ""
	return app.Default(out, errOut, cmd.InOrStdin(), t.IsTerminalOutput(), errTTY, interactive, color)
}

// isTerminal reports whether f is a character device.
func isTerminal(f *os.File) bool {
	st, err := f.Stat()
	return err == nil && st.Mode()&os.ModeCharDevice != 0
}

// addCampaignFlags adds the flags every campaign command takes.
func addCampaignFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("file", "f", "", "campaign file (default ./fleet.yaml)")
	cmd.Flags().StringSlice("repo", nil, "limit to these repos (owner/name, repeatable)")
	addRepoFileFlags(cmd, "limit to the repos listed in this file, one per line (- for stdin, repeatable)")
	cmd.Flags().Int("concurrency", 0, "repos processed in parallel (default limits.concurrency, else 8)")
	cmd.Flags().Float64("write-rate", 1, "GitHub API writes per second")
}

// addRepoFileFlags adds --repo-file, with the given help, and the
// exclusion flags.
func addRepoFileFlags(cmd *cobra.Command, repoFileHelp string) {
	cmd.Flags().StringSlice("repo-file", nil, repoFileHelp)
	cmd.Flags().StringSlice("exclude", nil, "leave out these repos (owner/name, repeatable)")
	cmd.Flags().StringSlice("exclude-file", nil, "leave out the repos listed in this file, one per line (- for stdin, repeatable)")
}

// addOutputFlags adds --json, --jq and --template. samples are the values
// the command prints as JSON, which give the fields --json accepts.
func addOutputFlags(cmd *cobra.Command, samples ...any) {
	cmd.Flags().String("json", "", "output JSON with the specified `fields` (comma-separated; --json alone lists them)")
	cmd.Flags().StringP("jq", "q", "", "filter JSON output using a jq `expression`")
	cmd.Flags().StringP("template", "t", "", "format JSON output using a Go template; see gh help formatting")
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations[jsonFieldsKey] = strings.Join(render.FieldsOf(samples...), ",")
}

// addRunFlags adds the flags of commands that clone and run the change.
func addRunFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("keep-workdirs", false, "leave clones on disk for debugging")
}

// addWriteFlags adds the flags of commands that write to GitHub.
func addWriteFlags(cmd *cobra.Command) {
	cmd.Flags().BoolP("yes", "y", false, "skip the confirmation prompt")
}

// common reads the shared flags through viper, and the repo lists named
// by --repo-file and --exclude-file.
func common(cmd *cobra.Command, args []string) (app.Common, error) {
	c := app.Common{
		File:         viper.GetString("file"),
		Repos:        viper.GetStringSlice("repo"),
		Concurrency:  viper.GetInt("concurrency"),
		WriteRate:    viper.GetFloat64("write-rate"),
		Yes:          viper.GetBool("yes"),
		KeepWorkdirs: viper.GetBool("keep-workdirs"),
		JSON:         viper.GetString("json") != "",
		JSONFields:   jsonFields(viper.GetString("json")),
		JQ:           viper.GetString("jq"),
		Template:     viper.GetString("template"),
	}
	if len(args) > 0 {
		c.Campaign = args[0]
	}
	files, excludeFiles := viper.GetStringSlice("repo-file"), viper.GetStringSlice("exclude-file")
	if countStdin(files)+countStdin(excludeFiles) > 1 {
		return c, errors.New("only one of --repo-file and --exclude-file can read stdin")
	}
	listed, err := app.ReadRepoFiles(files, cmd.InOrStdin())
	if err != nil {
		return c, err
	}
	if len(files) > 0 && len(listed) == 0 {
		return c, errors.New("--repo-file lists no repos")
	}
	c.Repos = append(c.Repos, listed...)
	excluded, err := app.ReadRepoFiles(excludeFiles, cmd.InOrStdin())
	if err != nil {
		return c, err
	}
	c.Exclude = append(viper.GetStringSlice("exclude"), excluded...)
	return c, nil
}

// countStdin counts the - entries in paths.
func countStdin(paths []string) int {
	n := 0
	for _, p := range paths {
		if p == "-" {
			n++
		}
	}
	return n
}

// run builds the App and calls f with it.
func run(cmd *cobra.Command, f func(a *app.App) error) error {
	a, err := newApp(cmd)
	if err != nil {
		return err
	}
	a.Version = version
	return f(a)
}

// Exit codes, the same as gh's.
const (
	exitOK      = 0
	exitError   = 1
	exitCancel  = 2
	exitAuth    = 4
	exitPending = 8
	// exitInterrupt is the shell's code for a run stopped by Ctrl-C.
	exitInterrupt = 130
)

// exitCode maps the error a command returned to the process exit code.
// interrupted is true when Ctrl-C stopped the run.
func exitCode(err error, interrupted bool) int {
	switch {
	case interrupted:
		return exitInterrupt
	case err == nil:
		return exitOK
	case errors.Is(err, ui.ErrNotConfirmed):
		return exitCancel
	case errors.Is(err, ghapi.ErrAuthRequired):
		return exitAuth
	case errors.Is(err, app.ErrPending):
		return exitPending
	}
	return exitError
}

// usagePrefixes start the errors cobra and pflag return for a bad command
// line.
var usagePrefixes = []string{
	"unknown command", "unknown flag", "unknown shorthand flag", "flag needs an argument",
	"invalid argument", "accepts ", "requires at least", "bad flag syntax",
}

// printError writes err to w as plain text, the way gh does, so it reads
// the same in a terminal, a log or a pipe. A command line mistake gets a
// pointer to --help; a pending status, which only sets the exit code,
// prints nothing.
func printError(w io.Writer, err error) {
	if errors.Is(err, app.ErrPending) {
		return
	}
	fmt.Fprintln(w, err)
	for _, p := range usagePrefixes {
		if strings.HasPrefix(err.Error(), p) {
			fmt.Fprintln(w, "\nRun gh fleet <command> --help for usage.")
			return
		}
	}
}
