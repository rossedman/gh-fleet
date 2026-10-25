// Package cmd holds gh fleet's cobra commands. Commands only parse flags
// and call into internal/app; one command per file.
package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/fang"
	"github.com/cli/go-gh/v2/pkg/term"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/rossedman/gh-fleet/internal/app"
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
	Annotations:       map[string]string{cobra.CommandDisplayNameAnnotation: "gh fleet"},
	SilenceUsage:      true,
	SilenceErrors:     true,
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error { return initConfig(cmd) },
}

// init registers the persistent --config flag.
func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default $XDG_CONFIG_HOME/gh-fleet/config.yaml)")
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
	go func() {
		<-sig
		fmt.Fprintln(os.Stderr, "\nStopping: finishing in-flight repos and saving state. Press Ctrl-C again to quit now.")
		cancel()
		<-sig
		os.Exit(130)
	}()
	if err := fang.Execute(ctx, rootCmd, fang.WithVersion(version), fang.WithCommit(commit), fang.WithoutManpage()); err != nil {
		os.Exit(1)
	}
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
	interactive := t.IsTerminalOutput() && isTerminal(os.Stdin)
	return app.Default(out, errOut, cmd.InOrStdin(), interactive, color)
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
	cmd.Flags().Int("concurrency", 0, "repos processed in parallel (default limits.concurrency, else 8)")
	cmd.Flags().Float64("write-rate", 1, "GitHub API writes per second")
}

// addOutputFlags adds --json and --jq.
func addOutputFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("json", false, "print JSON instead of a table")
	cmd.Flags().String("jq", "", "filter JSON output with a jq expression")
}

// addRunFlags adds the flags of commands that clone and run the change.
func addRunFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("keep-workdirs", false, "leave clones on disk for debugging")
}

// addWriteFlags adds the flags of commands that write to GitHub.
func addWriteFlags(cmd *cobra.Command) {
	cmd.Flags().BoolP("yes", "y", false, "skip the confirmation prompt")
}

// common reads the shared flags through viper.
func common(args []string) app.Common {
	c := app.Common{
		File:         viper.GetString("file"),
		Repos:        viper.GetStringSlice("repo"),
		Concurrency:  viper.GetInt("concurrency"),
		WriteRate:    viper.GetFloat64("write-rate"),
		Yes:          viper.GetBool("yes"),
		KeepWorkdirs: viper.GetBool("keep-workdirs"),
		JSON:         viper.GetBool("json"),
		JQ:           viper.GetString("jq"),
	}
	if len(args) > 0 {
		c.Campaign = args[0]
	}
	return c
}

// run builds the App and calls f with it.
func run(cmd *cobra.Command, f func(a *app.App) error) error {
	a, err := newApp(cmd)
	if err != nil {
		return err
	}
	return f(a)
}
