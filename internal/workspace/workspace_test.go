package workspace

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rossedman/gh-fleet/internal/ghapi"
	"github.com/rossedman/gh-fleet/internal/testutil"
)

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
