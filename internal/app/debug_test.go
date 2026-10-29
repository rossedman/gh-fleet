package app

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// timingNames returns the names the app's recorder has seen.
func timingNames(a *App) map[string]int {
	got := map[string]int{}
	for _, s := range a.Timings.Stats() {
		got[s.Name] = s.Count
	}
	return got
}

func TestDebugTimesPlanAndApply(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.app.EnableDebug()
	if h.app.Git.Timings != h.app.Timings || h.app.Log == nil {
		t.Fatal("EnableDebug didn't wire git timings and the logger")
	}
	f := h.campaign(basic, files)
	if err := h.app.Plan(context.Background(), PlanOptions{Common: Common{File: f}}); err != nil {
		t.Fatal(err)
	}
	got := timingNames(h.app)
	for name, n := range map[string]int{
		"load campaign": 1, "compile starlark": 1, "resolve targets": 1, "pick wave": 1,
		"run change in every repo": 1, "save state": 1,
		"repo: total": 2, "repo: clone": 2, "repo: step #1": 2, "repo: diff": 2, "repo: cleanup": 2,
	} {
		if got[name] != n {
			t.Errorf("%s recorded %d times, want %d (all: %v)", name, got[name], n, got)
		}
	}
	if got["git clone"] != 2 {
		t.Errorf("git clone recorded %d times", got["git clone"])
	}
	logs := h.errOut.String()
	if !strings.Contains(logs, "resolve targets: starting") || !strings.Contains(logs, "repo=acme/api") {
		t.Fatalf("debug logs:\n%s", logs)
	}

	h.answer = true
	if err := h.app.Apply(context.Background(), ApplyOptions{Common: Common{File: f, Yes: true}}); err != nil {
		t.Fatal(err)
	}
	got = timingNames(h.app)
	for _, name := range []string{"index existing PRs", "skip unchanged repos", "repo: push", "repo: sync PR"} {
		if got[name] == 0 {
			t.Errorf("%s not recorded (all: %v)", name, got)
		}
	}

	h.reset()
	if err := h.app.TimingSummary(); err != nil {
		t.Fatal(err)
	}
	if s := h.errOut.String(); !strings.Contains(s, "Timings") || !strings.Contains(s, "repo: clone") {
		t.Fatalf("summary:\n%s", s)
	}
}

func TestNoDebugRecordsNothing(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	if err := h.app.Plan(context.Background(), PlanOptions{Common: Common{File: h.campaign(basic, files)}}); err != nil {
		t.Fatal(err)
	}
	if h.app.Timings.Stats() != nil || strings.Contains(h.errOut.String(), "starting") {
		t.Fatalf("recorded without --debug:\n%s", h.errOut.String())
	}
	if err := h.app.TimingSummary(); err != nil {
		t.Fatal(err)
	}
}

func TestNoteWriterSplitsLines(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	w := noteWriter{h.app}
	n, err := w.Write([]byte("one\ntwo\n"))
	if err != nil || n != 8 || h.errOut.String() != "one\ntwo\n" {
		t.Fatalf("wrote %d, %v: %q", n, err, h.errOut.String())
	}
}

func TestRunsLookUpCampaignBranchesInBatches(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.app.EnableDebug()
	f := h.campaign(basic, files)
	ctx := context.Background()
	if err := h.app.Apply(ctx, ApplyOptions{Common: Common{File: f, Yes: true}}); err != nil {
		t.Fatal(err)
	}
	// The second run finds the branches the first pushed, in one query.
	h.remotes["acme/api"].Commit("main", "moved", map[string]string{"NEW": "1\n"})
	if err := h.app.Plan(ctx, PlanOptions{Common: Common{File: f}}); err != nil {
		t.Fatal(err)
	}
	if err := h.app.Apply(ctx, ApplyOptions{Common: Common{File: f, Yes: true}, Recheck: true}); err != nil {
		t.Fatal(err)
	}
	got := timingNames(h.app)
	if got["git ls-remote"] != 0 || got["look up campaign branches"] != 3 {
		t.Fatalf("timings = %v", got)
	}
	if h.remotes["acme/api"].Show("fleet/add-line", "NEW") != "1" {
		t.Fatal("existing branch not rebased onto the moved base")
	}
}

func TestRunsLeaveNoClonesBehind(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.app.EnableDebug()
	if err := h.app.Plan(context.Background(), PlanOptions{Common: Common{File: h.campaign(basic, files)}}); err != nil {
		t.Fatal(err)
	}
	// Clones are deleted in the background, and all of them by the time
	// the run returns.
	if got := timingNames(h.app)["cleanup (background)"]; got != 2 {
		t.Fatalf("background cleanups = %d", got)
	}
	var left []string
	_ = filepath.WalkDir(h.app.Workspaces.Root, func(p string, d fs.DirEntry, err error) error {
		if err == nil && p != h.app.Workspaces.Root {
			left = append(left, p)
		}
		return nil
	})
	// Only the campaign and owner directories may remain.
	for _, p := range left {
		if rel, _ := filepath.Rel(h.app.Workspaces.Root, p); strings.Count(rel, string(filepath.Separator)) > 1 {
			t.Fatalf("left behind: %v", left)
		}
	}
}

func TestRunsCloneThroughTheGitCache(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.app.Workspaces.CacheDir = filepath.Join(t.TempDir(), "git")
	h.app.EnableDebug()
	f := h.campaign(basic, files)
	ctx := context.Background()
	if err := h.app.Plan(ctx, PlanOptions{Common: Common{File: f}}); err != nil {
		t.Fatal(err)
	}
	// Apply after the base moved still sees the new commit, and pushes
	// from a workspace that borrows the cache's files.
	h.remotes["acme/api"].Commit("main", "moved", map[string]string{"NEW": "1\n"})
	if err := h.app.Apply(ctx, ApplyOptions{Common: Common{File: f, Yes: true}}); err != nil {
		t.Fatal(err)
	}
	if h.remotes["acme/api"].Show("fleet/add-line", "NEW") != "1" || len(h.gh.PRs) != 2 {
		t.Fatalf("branch NEW %q, %d PRs", h.remotes["acme/api"].Show("fleet/add-line", "NEW"), len(h.gh.PRs))
	}
	got := timingNames(h.app)
	if got["git clone"] != 0 || got["git fetch"] != 4 {
		t.Fatalf("timings = %v", got)
	}
	if entries, _ := os.ReadDir(filepath.Join(h.app.Workspaces.CacheDir, "acme")); len(entries) != 2 {
		t.Fatalf("cache = %v", entries)
	}

	// --no-git-cache clones from GitHub, and a cache that can't be used
	// is noted and cloned around.
	h.reset()
	if err := h.app.Plan(ctx, PlanOptions{Common: Common{File: f, NoGitCache: true}}); err != nil {
		t.Fatal(err)
	}
	if got := timingNames(h.app)["git clone"]; got != 2 {
		t.Fatalf("--no-git-cache clones = %d", got)
	}
	h.app.Workspaces.CacheDir = filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(h.app.Workspaces.CacheDir, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	h.reset()
	if err := h.app.Plan(ctx, PlanOptions{Common: Common{File: f}}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.errOut.String(), "Couldn't use the git cache for acme/api (update git cache:") {
		t.Fatalf("stderr:\n%s", h.errOut)
	}
}
