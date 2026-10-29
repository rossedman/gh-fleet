package gitops_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rossedman/gh-fleet/internal/gitops"
	"github.com/rossedman/gh-fleet/internal/testutil"
	"github.com/rossedman/gh-fleet/internal/timing"
)

func TestCloneCommitPush(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	env := testutil.GitEnv(t)
	remote := testutil.NewRemote(t, env, map[string]string{"a.txt": "a\n"})
	g := testutil.Git(env)
	dir := filepath.Join(t.TempDir(), "clone")
	if err := g.Clone(ctx, remote.URL(), dir, "main"); err != nil {
		t.Fatal(err)
	}
	if shallow, _ := g.Run(ctx, dir, "rev-parse", "--is-shallow-repository"); shallow != "true" {
		t.Fatal("clone is not shallow")
	}
	// The files come with the clone, not in a second fetch at checkout.
	if promisor, _ := g.Run(ctx, dir, "config", "--get", "remote.origin.promisor"); promisor != "" {
		t.Fatal("clone is a partial clone")
	}
	if spec, _ := g.Run(ctx, dir, "config", "--get", "remote.origin.fetch"); spec != "+refs/heads/main:refs/remotes/origin/main" {
		t.Fatalf("clone is not single-branch: fetches %q", spec)
	}
	if sha, err := g.RemoteBranchSHA(ctx, dir, "fleet/x"); err != nil || sha != "" {
		t.Fatalf("missing branch: %q, %v", sha, err)
	}
	if err := g.Checkout(ctx, dir, "fleet/x", "origin/main"); err != nil {
		t.Fatal(err)
	}
	if dirty, _ := g.Dirty(ctx, dir); dirty {
		t.Fatal("fresh clone dirty")
	}
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("ignored\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ignored"), []byte("x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if dirty, _ := g.Dirty(ctx, dir); !dirty {
		t.Fatal("untracked file not detected")
	}
	diff, err := g.StagedDiff(ctx, dir, "origin/main")
	if err != nil || !strings.Contains(diff, "+b") || strings.Contains(diff, "ignored\n+x") {
		t.Fatalf("diff = %q, %v", diff, err)
	}
	if err := g.CommitAll(ctx, dir, "add b", true); err != nil {
		t.Fatal(err)
	}
	msg, _ := g.Run(ctx, dir, "log", "-1", "--format=%B")
	if !strings.Contains(msg, "Signed-off-by: Fleet Test <fleet@example.com>") {
		t.Fatalf("no signoff: %q", msg)
	}
	if same, _ := g.SameTree(ctx, dir, "HEAD", "origin/main"); same {
		t.Fatal("trees should differ")
	}
	if err := g.Push(ctx, dir, "fleet/x", ""); err != nil {
		t.Fatal(err)
	}
	head, _ := g.Rev(ctx, dir, "HEAD")
	if remote.Branch("fleet/x") != head {
		t.Fatal("push did not land")
	}
	// A stale lease is rejected.
	remote.Commit("fleet/x", "someone else", map[string]string{"c.txt": "c\n"})
	if err := g.Push(ctx, dir, "fleet/x", head); err == nil {
		t.Fatal("push with stale lease succeeded")
	}
	if sha, _ := g.RemoteBranchSHA(ctx, dir, "fleet/x"); sha == "" || sha == head {
		t.Fatalf("remote sha = %q", sha)
	}
	if err := g.FetchBranches(ctx, dir, "main", "fleet/x"); err != nil {
		t.Fatal(err)
	}
	if err := g.FetchBranches(ctx, dir, "main"); err != nil {
		t.Fatalf("fetch on a full clone: %v", err)
	}
}

func TestRebaseConflictAborts(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	env := testutil.GitEnv(t)
	remote := testutil.NewRemote(t, env, map[string]string{"a.txt": "a\n"})
	remote.Commit("feature", "feature", map[string]string{"a.txt": "feature\n"})
	remote.Commit("main", "main", map[string]string{"a.txt": "main\n"})
	g := testutil.Git(env)
	dir := filepath.Join(t.TempDir(), "clone")
	if err := g.Clone(ctx, remote.URL(), dir, "main"); err != nil {
		t.Fatal(err)
	}
	if err := g.FetchBranches(ctx, dir, "main", "feature"); err != nil {
		t.Fatal(err)
	}
	if err := g.Checkout(ctx, dir, "feature", "origin/feature"); err != nil {
		t.Fatal(err)
	}
	if err := g.Rebase(ctx, dir, "origin/main"); err == nil {
		t.Fatal("expected conflict")
	}
	if dirty, _ := g.Dirty(ctx, dir); dirty {
		t.Fatal("rebase not aborted")
	}
}

func TestErrors(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	g := gitops.Git{Bin: "git"}
	dir := t.TempDir()
	if _, err := g.Run(ctx, dir, "status"); err == nil || !strings.Contains(err.Error(), "git status:") {
		t.Fatalf("err = %v", err)
	}
	if _, err := (gitops.Git{Bin: "/nonexistent/git"}).Run(ctx, dir, "status"); err == nil {
		t.Fatal("expected exec error")
	}
	if _, err := g.RemoteBranchSHA(ctx, dir, "x"); err == nil {
		t.Fatal("expected error outside a repo")
	}
	if _, err := g.Dirty(ctx, dir); err == nil {
		t.Fatal("expected error outside a repo")
	}
	if _, err := g.StagedDiff(ctx, dir, "HEAD"); err == nil {
		t.Fatal("expected error outside a repo")
	}
	if err := g.CommitAll(ctx, dir, "m", false); err == nil {
		t.Fatal("expected error outside a repo")
	}
	if _, err := g.SameTree(ctx, dir, "a", "b"); err == nil {
		t.Fatal("expected error outside a repo")
	}
	env := testutil.GitEnv(t)
	remote := testutil.NewRemote(t, env, map[string]string{"a": "a\n"})
	tg := testutil.Git(env)
	clone := filepath.Join(t.TempDir(), "c")
	if err := tg.Clone(ctx, remote.URL(), clone, "main"); err != nil {
		t.Fatal(err)
	}
	if _, err := tg.SameTree(ctx, clone, "HEAD", "nope"); err == nil {
		t.Fatal("expected bad rev error")
	}
}

func TestRunRecordsTimings(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	env := testutil.GitEnv(t)
	remote := testutil.NewRemote(t, env, map[string]string{"a.txt": "a\n"})
	g := testutil.Git(env)
	g.Timings = timing.New(nil)
	dir := filepath.Join(t.TempDir(), "clone")
	if err := g.Clone(ctx, remote.URL(), dir, "main"); err != nil {
		t.Fatal(err)
	}
	if _, err := g.StagedDiff(ctx, dir, "origin/main"); err != nil {
		t.Fatal(err)
	}
	_, _ = g.Run(ctx, dir, "no-such-command")
	got := map[string]int{}
	for _, s := range g.Timings.Stats() {
		got[s.Name] = s.Count
	}
	want := map[string]int{"git clone": 1, "git add": 1, "git diff": 1, "git no-such-command": 1}
	if len(got) != len(want) {
		t.Fatalf("timings = %v, want %v", got, want)
	}
	for k, n := range want {
		if got[k] != n {
			t.Fatalf("timings = %v, want %v", got, want)
		}
	}
}
