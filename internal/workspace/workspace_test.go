package workspace

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/rossedman/gh-fleet/internal/ghapi"
	"github.com/rossedman/gh-fleet/internal/testutil"
)

// failGitEnv names the git call the test binary fails when it runs as git:
// a subcommand ("fetch"), or a subcommand and its last argument
// ("checkout:origin/main").
const failGitEnv = "WORKSPACE_TEST_FAIL_GIT"

// TestMain lets the test binary stand in for git, so tests can fail one
// git call and reach Prepare's error paths on every platform.
func TestMain(m *testing.M) {
	if spec, ok := os.LookupEnv(failGitEnv); ok {
		os.Exit(fakeGit(spec, os.Args[1:]))
	}
	os.Exit(m.Run())
}

// fakeGit fails the git call that spec names and runs every other call
// with the real git, returning the exit code.
func fakeGit(spec string, args []string) int {
	sub, last, _ := strings.Cut(spec, ":")
	if len(args) > 0 && args[0] == sub && (last == "" || args[len(args)-1] == last) {
		_, _ = os.Stderr.WriteString("injected failure\n")
		return 1
	}
	cmd := exec.Command("git", args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return exit.ExitCode()
		}
		return 1
	}
	return 0
}

func TestPrepareNewAndExistingBranch(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	env := testutil.GitEnv(t)
	remote := testutil.NewRemote(t, env, map[string]string{"a.txt": "a\n"})
	m := &Manager{Root: t.TempDir(), Git: testutil.Git(env), CloneURL: func(ghapi.Repo) string { return remote.URL() }}
	repo := ghapi.Repo{Owner: "acme", Name: "api"}

	ws, err := m.Prepare(ctx, "c", repo, "main", "fleet/c")
	if err != nil {
		t.Fatal(err)
	}
	if ws.RemoteSHA != "" || ws.Reset || ws.Dir != filepath.Join(m.Root, "c", "acme", "api") {
		t.Fatalf("ws = %+v", ws)
	}
	branch, _ := m.Git.Run(ctx, ws.Dir, "branch", "--show-current")
	if branch != "fleet/c" {
		t.Fatalf("branch = %q", branch)
	}
	ws.Cleanup()
	if _, err := os.Stat(ws.Dir); !os.IsNotExist(err) {
		t.Fatal("cleanup left the workspace")
	}

	sha := remote.Commit("fleet/c", "old", map[string]string{"b.txt": "b\n"})
	remote.Commit("main", "newer", map[string]string{"c.txt": "c\n"})
	m.Keep = true
	ws, err = m.Prepare(ctx, "c", repo, "main", "fleet/c")
	if err != nil {
		t.Fatal(err)
	}
	if ws.RemoteSHA != sha || ws.Reset {
		t.Fatalf("ws = %+v", ws)
	}
	for _, f := range []string{"b.txt", "c.txt"} {
		if _, err := os.Stat(filepath.Join(ws.Dir, f)); err != nil {
			t.Fatalf("%s missing after rebase", f)
		}
	}
	ws.Cleanup()
	if _, err := os.Stat(ws.Dir); err != nil {
		t.Fatal("kept workspace removed")
	}

	// A conflicting branch is reset to the base.
	remote.Commit("fleet/c", "conflict", map[string]string{"a.txt": "branch\n"})
	remote.Commit("main", "conflict", map[string]string{"a.txt": "main\n"})
	ws, err = m.Prepare(ctx, "c", repo, "main", "fleet/c")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(ws.Dir, "a.txt"))
	if !ws.Reset || string(b) != "main\n" {
		t.Fatalf("not reset: %+v %q", ws, b)
	}

	ro, err := m.Clone(ctx, "exec", repo, "main")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(ro.Dir, "a.txt")); err != nil {
		t.Fatal("clone missing files")
	}
}

func TestPrepareErrors(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	env := testutil.GitEnv(t)
	m := &Manager{Root: t.TempDir(), Git: testutil.Git(env), CloneURL: func(ghapi.Repo) string { return "file:///nonexistent.git" }}
	repo := ghapi.Repo{Owner: "acme", Name: "api"}
	if _, err := m.Prepare(ctx, "c", repo, "main", "fleet/c"); err == nil {
		t.Fatal("expected clone error")
	}
	if _, err := m.Clone(ctx, "c", repo, "main"); err == nil {
		t.Fatal("expected clone error")
	}
	remote := testutil.NewRemote(t, env, map[string]string{"a": "a\n"})
	m.CloneURL = func(ghapi.Repo) string { return remote.URL() }
	if _, err := m.Prepare(ctx, "c", repo, "nope", "fleet/c"); err == nil {
		t.Fatal("expected missing base error")
	}
	ws, err := m.Prepare(ctx, "c", repo, "main", "bad..branch")
	if err == nil {
		t.Fatal("expected checkout error for an invalid branch name")
	}
	if ws != nil {
		t.Fatalf("ws = %+v, want nil on error", ws)
	}
	if _, statErr := os.Stat(m.Path("c", repo)); !os.IsNotExist(statErr) {
		t.Fatal("failed prepare left the workspace")
	}
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	m.Root = blocker
	if _, err := m.Prepare(ctx, "c", repo, "main", "fleet/c"); err == nil {
		t.Fatal("expected mkdir error")
	}
	if _, err := m.Clone(ctx, "c", repo, "main"); err == nil {
		t.Fatal("expected mkdir error")
	}
}

func TestDefaults(t *testing.T) {
	t.Parallel()
	m := &Manager{}
	repo := ghapi.Repo{Owner: "acme", Name: "api"}
	if !strings.HasPrefix(m.Path("c", repo), DefaultRoot()) {
		t.Fatal("default root")
	}
	if m.url(repo) != "https://github.com/acme/api.git" {
		t.Fatal(m.url(repo))
	}
	repo.URL = "https://ghe.example.com/acme/api"
	if m.url(repo) != "https://ghe.example.com/acme/api.git" {
		t.Fatal(m.url(repo))
	}
}

func TestPrepareGitFailures(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		fail string
		// conflict makes the existing branch conflict with main, so the
		// rebase fails and Prepare falls back to resetting to main.
		conflict bool
	}{
		{name: "ls-remote", fail: "ls-remote"},
		{name: "fetch", fail: "fetch"},
		{name: "checkout existing branch", fail: "checkout:origin/fleet/c"},
		{name: "reset after conflict", fail: "checkout:origin/main", conflict: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			env := testutil.GitEnv(t)
			remote := testutil.NewRemote(t, env, map[string]string{"a.txt": "a\n"})
			remote.Commit("fleet/c", "branch", map[string]string{"a.txt": "branch\n"})
			if tc.conflict {
				remote.Commit("main", "main", map[string]string{"a.txt": "main\n"})
			}
			m := &Manager{
				Root:     t.TempDir(),
				Git:      testutil.Git(slices.Concat(env, []string{failGitEnv + "=" + tc.fail})),
				CloneURL: func(ghapi.Repo) string { return remote.URL() },
			}
			m.Git.Bin = os.Args[0]
			repo := ghapi.Repo{Owner: "acme", Name: "api"}
			ws, err := m.Prepare(ctx, "c", repo, "main", "fleet/c")
			if err == nil || !strings.Contains(err.Error(), "injected failure") {
				t.Fatalf("err = %v, want the injected failure", err)
			}
			if ws != nil {
				t.Fatalf("ws = %+v, want nil on error", ws)
			}
			if _, statErr := os.Stat(m.Path("c", repo)); !os.IsNotExist(statErr) {
				t.Fatal("failed prepare left the workspace")
			}
		})
	}
}

func TestRemoveError(t *testing.T) {
	t.Parallel()
	// A NUL byte makes every path operation fail, before any git runs.
	m := &Manager{Root: t.TempDir()}
	repo := ghapi.Repo{Owner: "acme", Name: "api"}
	if _, err := m.Prepare(t.Context(), "bad\x00", repo, "main", "fleet/c"); err == nil {
		t.Fatal("Prepare: expected an error for an invalid path")
	}
	if _, err := m.Clone(t.Context(), "bad\x00", repo, "main"); err == nil {
		t.Fatal("Clone: expected an error for an invalid path")
	}
}

func TestPrepareAt(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	env := testutil.GitEnv(t)
	remote := testutil.NewRemote(t, env, map[string]string{"a.txt": "a\n"})
	m := &Manager{Root: t.TempDir(), Git: testutil.Git(env), CloneURL: func(ghapi.Repo) string { return remote.URL() }}
	repo := ghapi.Repo{Owner: "acme", Name: "api"}

	// No branch: checked out fresh from the base.
	ws, err := m.PrepareAt(ctx, "c", repo, "main", "fleet/c", "")
	if err != nil {
		t.Fatal(err)
	}
	cur, _ := m.Git.Run(ctx, ws.Dir, "rev-parse", "--abbrev-ref", "HEAD")
	if ws.RemoteSHA != "" || ws.Branch != "fleet/c" || cur != "fleet/c" {
		t.Fatalf("new: %+v on %q", ws, cur)
	}
	ws.Cleanup()

	// A branch at the given commit is fetched and rebased onto the base.
	remote.Commit("fleet/c", "old", map[string]string{"old.txt": "1\n"})
	remote.Commit("main", "moved", map[string]string{"b.txt": "b\n"})
	sha := remote.Branch("fleet/c")
	ws, err = m.PrepareAt(ctx, "c", repo, "main", "fleet/c", sha)
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Cleanup()
	for _, f := range []string{"old.txt", "b.txt"} {
		if _, err := os.Stat(filepath.Join(ws.Dir, f)); err != nil {
			t.Fatalf("%s missing after rebase: %v", f, err)
		}
	}
	if ws.RemoteSHA != sha || ws.Reset {
		t.Fatalf("existing: %+v", ws)
	}
}
