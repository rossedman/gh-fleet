package cmd

import (
	"bytes"
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
	out, _, err := execute(t, "init", "add-file", "--org", "acme", "--file", e.file)
	if err != nil || !strings.Contains(out, "Wrote") {
		t.Fatalf("init: %q, %v", out, err)
	}
	b, _ := os.ReadFile(e.file)
	edited := strings.Replace(string(b), `echo "replace me" >/dev/null`, `echo added > ADDED.md`, 1)
	if err := os.WriteFile(e.file, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}

	out, _, err = execute(t, "targets", "-f", e.file)
	if err != nil || !strings.Contains(out, "acme/api") || !strings.Contains(out, "1 repo") {
		t.Fatalf("targets: %q, %v", out, err)
	}
	out, _, err = execute(t, "targets", "-f", e.file, "--jq", ".targets[0].reasons[0]")
	if err != nil || strings.TrimSpace(out) != "org:acme" {
		t.Fatalf("targets --jq: %q, %v", out, err)
	}

	out, errOut, err := execute(t, "plan", "-f", e.file, "--concurrency", "2")
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

	out, _, err = execute(t, "status", "add-file")
	if err != nil || !strings.Contains(out, "pr open") || !strings.Contains(out, "API quota: core 5,000/5,000") {
		t.Fatalf("status: %q, %v", out, err)
	}
	out, _, err = execute(t, "list", "--json", "--jq", ".[0].name")
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
