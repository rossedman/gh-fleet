package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/rossedman/gh-fleet/internal/app"
	"github.com/rossedman/gh-fleet/internal/ghapi"
	"github.com/rossedman/gh-fleet/internal/ghapi/fake"
	"github.com/rossedman/gh-fleet/internal/state"
	"github.com/rossedman/gh-fleet/internal/testutil"
	"github.com/rossedman/gh-fleet/internal/ui"
	"github.com/rossedman/gh-fleet/internal/workspace"
)

// env is a fake GitHub with one repo, wired into newApp.
type env struct {
	gh   *fake.Client
	dir  string
	file string
}

// setup replaces newApp with one backed by a fake and a local bare repo,
// and writes a campaign file.
func setup(t *testing.T) *env {
	t.Helper()
	gitEnv := testutil.GitEnv(t)
	remote := testutil.NewRemote(t, gitEnv, map[string]string{"README.md": "hi\n"})
	e := &env{gh: fake.New(), dir: t.TempDir()}
	e.gh.AddRepo(ghapi.Repo{Owner: "acme", Name: "api", CanPush: true})
	e.gh.Limits = []ghapi.RateLimit{{Resource: "core", Limit: 5000, Remaining: 5000}}
	stateDir := filepath.Join(t.TempDir(), "state")
	work := t.TempDir()
	old := newApp
	t.Cleanup(func() { newApp = old })
	newApp = func(cmd *cobra.Command) (*app.App, error) {
		return &app.App{
			TTY: true,
			Out: cmd.OutOrStdout(), Err: cmd.ErrOrStderr(), In: strings.NewReader(""),
			NewClient: func(float64) (ghapi.Client, error) { return e.gh, nil },
			Store:     &state.Store{Dir: stateDir},
			Workspaces: &workspace.Manager{
				Root: work, CloneURL: func(ghapi.Repo) string { return remote.URL() },
			},
			Git: testutil.Git(gitEnv),
		}, nil
	}
	e.file = filepath.Join(e.dir, "fleet.yaml")
	return e
}

// execute runs the root command with args and returns stdout and stderr.
func execute(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	resetFlags(rootCmd)
	viper.Reset()
	cfgFile = ""
	var out, errOut bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errOut)
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	return out.String(), errOut.String(), err
}

// resetFlags restores every flag of cmd and its children to its default,
// since cobra keeps values between Execute calls.
func resetFlags(cmd *cobra.Command) {
	reset := func(f *pflag.Flag) {
		if sv, ok := f.Value.(pflag.SliceValue); ok {
			_ = sv.Replace(nil)
		} else {
			_ = f.Value.Set(f.DefValue)
		}
		f.Changed = false
	}
	cmd.Flags().VisitAll(reset)
	cmd.PersistentFlags().VisitAll(reset)
	for _, c := range cmd.Commands() {
		resetFlags(c)
	}
}

func TestInitTargetsPlanApplyStatusListClose(t *testing.T) {
	e := setup(t)
	out, initErr, err := execute(t, "init", "add-file", "--org", "acme", "--file", e.file)
	if err != nil || !strings.Contains(initErr, "Wrote") || out != "" {
		t.Fatalf("init: %q %q, %v", out, initErr, err)
	}
	b, _ := os.ReadFile(e.file)
	edited := strings.Replace(string(b), `echo "replace me" >/dev/null`, `echo added > ADDED.md`, 1)
	if err := os.WriteFile(e.file, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}

	out, errOut, err := execute(t, "targets", "-f", e.file)
	if err != nil || !strings.Contains(out, "acme/api") || !strings.Contains(errOut, "1 repo") {
		t.Fatalf("targets: %q %q, %v", out, errOut, err)
	}
	out, _, err = execute(t, "targets", "-f", e.file, "--jq", ".targets[0].reasons[0]")
	if err != nil || strings.TrimSpace(out) != "org:acme" {
		t.Fatalf("targets --jq: %q, %v", out, err)
	}

	out, errOut, err = execute(t, "plan", "-f", e.file, "--concurrency", "2")
	if err != nil || !strings.Contains(out, "+added") || !strings.Contains(errOut, "1 changed") {
		t.Fatalf("plan: %q %q, %v", out, errOut, err)
	}

	if _, _, err := execute(t, "apply", "-f", e.file); err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("apply without --yes: %v", err)
	}
	_, errOut, err = execute(t, "apply", "-f", e.file, "--yes")
	if err != nil || !strings.Contains(errOut, "1 pr open") || len(e.gh.PRs) != 1 {
		t.Fatalf("apply: %q, %v", errOut, err)
	}
	_, errOut, err = execute(t, "retry", "add-file", "--yes")
	if err != nil || !strings.Contains(errOut, "Nothing to do.") {
		t.Fatalf("retry: %q, %v", errOut, err)
	}

	out, errOut, err = execute(t, "status", "add-file")
	if err != nil || !strings.Contains(out, "pr open") || !strings.Contains(errOut, "API quota: core 5,000/5,000") {
		t.Fatalf("status: %q %q, %v", out, errOut, err)
	}
	if _, _, err = execute(t, "status", "add-file", "--exit-status"); exitCode(err, false) != exitPending {
		t.Fatalf("status --exit-status with an open PR: %v", err)
	}
	out, _, err = execute(t, "list", "--json", "name", "--jq", ".[0].name")
	if err != nil || strings.TrimSpace(out) != "add-file" {
		t.Fatalf("list: %q, %v", out, err)
	}

	_, errOut, err = execute(t, "close", "add-file", "--yes", "--comment", "done")
	if err != nil || !strings.Contains(errOut, "1 closed") {
		t.Fatalf("close: %q, %v", errOut, err)
	}
	if e.gh.Comments["acme/api#1"][0] != "done" {
		t.Fatal("close comment not posted")
	}
}

func TestExecAndVersion(t *testing.T) {
	setup(t)
	out, _, err := execute(t, "exec", "--repo", "acme/api", "--", "cat", "README.md")
	if err != nil || !strings.Contains(out, "hi") {
		t.Fatalf("exec: %q, %v", out, err)
	}
	out, _, err = execute(t, "version")
	if err != nil || !strings.HasPrefix(out, "gh-fleet dev") {
		t.Fatalf("version: %q, %v", out, err)
	}
	if Root() != rootCmd {
		t.Fatal("Root")
	}
}

func TestConfigPrecedence(t *testing.T) {
	setup(t)
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfg, []byte("concurrency: 3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := execute(t, "list", "--config", cfg); err != nil {
		t.Fatal(err)
	}
	if viper.GetInt("concurrency") != 3 {
		t.Fatalf("config file not read: %d", viper.GetInt("concurrency"))
	}
	t.Setenv("GH_FLEET_CONCURRENCY", "5")
	if _, _, err := execute(t, "list", "--config", cfg); err != nil {
		t.Fatal(err)
	}
	if viper.GetInt("concurrency") != 5 {
		t.Fatalf("env does not override file: %d", viper.GetInt("concurrency"))
	}
	if _, _, err := execute(t, "plan", "--config", cfg, "--concurrency", "7", "-f", "/nonexistent.yaml"); err == nil {
		t.Fatal("expected missing campaign error")
	}
	if viper.GetInt("concurrency") != 7 {
		t.Fatalf("flag does not override env: %d", viper.GetInt("concurrency"))
	}
	if _, _, err := execute(t, "list", "--config", filepath.Join(dir, "missing.yaml")); err == nil {
		t.Fatal("expected error for a missing --config file")
	}
	bad := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(bad, []byte("a: [\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := execute(t, "list", "--config", bad); err == nil {
		t.Fatal("expected error for a bad config file")
	}
	t.Setenv("XDG_CONFIG_HOME", dir)
	if _, _, err := execute(t, "list"); err != nil {
		t.Fatalf("XDG config: %v", err)
	}
}

func TestNewAppDefault(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	a, err := newApp(rootCmd)
	if err != nil || a.Store == nil || a.Interactive {
		t.Fatalf("app = %+v, %v", a, err)
	}
	t.Setenv("CLICOLOR_FORCE", "1")
	if a, err := newApp(rootCmd); err != nil || !a.Color {
		t.Fatalf("forced color: %v", err)
	}
	f, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	_ = isTerminal(f)
	if isTerminal(os.NewFile(^uintptr(0), "bad")) {
		t.Fatal("invalid file reported as terminal")
	}
}

func TestEveryCommandDocumented(t *testing.T) {
	for _, c := range rootCmd.Commands() {
		if c.Name() == "help" || c.Name() == "completion" {
			continue
		}
		if c.Short == "" || c.Example == "" && c.Name() != "version" {
			t.Errorf("%s needs Short and Example", c.Name())
		}
	}
}

func TestJSONFieldsAndTemplate(t *testing.T) {
	setup(t)
	_, _, err := execute(t, "lib", "--json")
	if err == nil || !strings.HasPrefix(err.Error(), "Specify one or more comma-separated fields for `--json`:\n  ") || !strings.Contains(err.Error(), "\n  module\n") {
		t.Fatalf("bare --json: %v", err)
	}
	if _, _, err := execute(t, "lib", "--json", "module,nope"); err == nil || !strings.Contains(err.Error(), `Unknown JSON field: "nope"`) {
		t.Fatalf("unknown field: %v", err)
	}
	out, _, err := execute(t, "lib", "--json", "module", "--jq", ".[0] | keys | join(\",\")")
	if err != nil || strings.TrimSpace(out) != "module" {
		t.Fatalf("--json module: %q, %v", out, err)
	}
	out, _, err = execute(t, "lib", "--json", "module,summary", "-t", `{{range .}}{{.module}}{{"\n"}}{{end}}`)
	if err != nil || !strings.HasPrefix(out, "android.star\nansible.star\n") {
		t.Fatalf("--template: %q, %v", out, err)
	}
}

func TestExitCodesAndErrors(t *testing.T) {
	for _, tt := range []struct {
		err  error
		stop bool
		want int
	}{
		{nil, false, exitOK},
		{errors.New("boom"), false, exitError},
		{fmt.Errorf("apply: %w", ui.ErrNotConfirmed), false, exitCancel},
		{ghapi.ErrAuthRequired, false, exitAuth},
		{app.ErrPending, false, exitPending},
		{nil, true, exitInterrupt},
		{&app.FailedError{N: 2}, false, exitError},
	} {
		if got := exitCode(tt.err, tt.stop); got != tt.want {
			t.Errorf("exitCode(%v, %v) = %d, want %d", tt.err, tt.stop, got, tt.want)
		}
	}
	var b bytes.Buffer
	printError(&b, errors.New("no fleet.yaml in the current directory"))
	if b.String() != "no fleet.yaml in the current directory\n" {
		t.Errorf("plain error = %q", b.String())
	}
	b.Reset()
	printError(&b, errors.New("unknown flag: --bogus"))
	if !strings.Contains(b.String(), "unknown flag: --bogus\n\nRun gh fleet <command> --help for usage.") {
		t.Errorf("usage error = %q", b.String())
	}
	b.Reset()
	printError(&b, app.ErrPending)
	if b.Len() != 0 {
		t.Errorf("pending printed %q", b.String())
	}
	setup(t)
	if _, _, err := execute(t, "list", "--json", "nope"); err == nil {
		t.Error("unknown field on a campaign command accepted")
	}
}

func TestRepoFiles(t *testing.T) {
	e := setup(t)
	if _, _, err := execute(t, "init", "add-file", "--org", "acme", "--file", e.file); err != nil {
		t.Fatal(err)
	}
	list := filepath.Join(e.dir, "repos.txt")
	empty := filepath.Join(e.dir, "empty.txt")
	_ = os.WriteFile(list, []byte("# targets\nhttps://github.com/acme/api\n"), 0o600)
	_ = os.WriteFile(empty, []byte("# none yet\n"), 0o600)
	count := "--jq=.targets | length"
	for _, tt := range []struct {
		args []string
		want string
	}{
		{[]string{"--repo-file", list}, "1"},
		{[]string{"--repo-file", list, "--exclude", "acme/api"}, "0"},
		{[]string{"--exclude-file", list}, "0"},
		{[]string{"--exclude-file", empty}, "1"},
	} {
		out, _, err := execute(t, append([]string{"targets", "-f", e.file, count}, tt.args...)...)
		if err != nil || strings.TrimSpace(out) != tt.want {
			t.Errorf("%v: %q, %v", tt.args, out, err)
		}
	}

	rootCmd.SetIn(strings.NewReader("acme/api\n"))
	t.Cleanup(func() { rootCmd.SetIn(nil) })
	out, _, err := execute(t, "targets", "-f", e.file, count, "--exclude-file", "-")
	if err != nil || strings.TrimSpace(out) != "0" {
		t.Errorf("stdin: %q, %v", out, err)
	}

	for _, tt := range []struct {
		args []string
		want string
	}{
		{[]string{"--repo-file", empty}, "--repo-file lists no repos"},
		{[]string{"--repo-file", filepath.Join(e.dir, "missing")}, "missing"},
		{[]string{"--exclude-file", filepath.Join(e.dir, "missing")}, "missing"},
		{[]string{"--repo-file", "-", "--exclude-file", "-"}, "only one of --repo-file and --exclude-file can read stdin"},
		{[]string{"--repo-file", e.file}, "is not owner/name"},
	} {
		if _, _, err := execute(t, append([]string{"plan", "-f", e.file}, tt.args...)...); err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%v: err = %v, want %q", tt.args, err, tt.want)
		}
	}
}
