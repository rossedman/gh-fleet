package app

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/rossedman/gh-fleet/internal/ghapi"
	"github.com/rossedman/gh-fleet/internal/state"
)

func TestDecideMerge(t *testing.T) {
	t.Parallel()
	ready := ghapi.PR{MergeState: "CLEAN", Checks: "SUCCESS", Review: "APPROVED"}
	with := func(f func(*ghapi.PR)) ghapi.PR { p := ready; f(&p); return p }
	for name, tt := range map[string]struct {
		pr             ghapi.PR
		o              MergeOptions
		action, reason string
	}{
		"ready":              {ready, MergeOptions{}, mergeNow, ""},
		"has hooks":          {with(func(p *ghapi.PR) { p.MergeState = "HAS_HOOKS" }), MergeOptions{}, mergeNow, ""},
		"draft":              {with(func(p *ghapi.PR) { p.Draft = true }), MergeOptions{Auto: true}, mergeWait, "draft"},
		"auto already":       {with(func(p *ghapi.PR) { p.AutoMerge = true }), MergeOptions{}, mergeWait, "auto-merge already on"},
		"changes requested":  {with(func(p *ghapi.PR) { p.Review = "CHANGES_REQUESTED" }), MergeOptions{Auto: true}, mergeWait, "changes requested"},
		"conflict":           {with(func(p *ghapi.PR) { p.MergeState = "DIRTY" }), MergeOptions{}, mergeWait, "merge conflict; gh fleet apply rebases the branch"},
		"failing":            {with(func(p *ghapi.PR) { p.Checks = "FAILURE"; p.MergeState = "BLOCKED" }), MergeOptions{Auto: true}, mergeWait, "checks failing"},
		"unstable":           {with(func(p *ghapi.PR) { p.MergeState = "UNSTABLE" }), MergeOptions{}, mergeWait, "checks failing that aren't required"},
		"behind":             {with(func(p *ghapi.PR) { p.MergeState = "BEHIND" }), MergeOptions{}, mergeWait, "behind its base; pass --update-branch"},
		"behind, update":     {with(func(p *ghapi.PR) { p.MergeState = "BEHIND" }), MergeOptions{UpdateBranch: true}, mergeUpdate, ""},
		"running":            {with(func(p *ghapi.PR) { p.Checks = "PENDING"; p.MergeState = "BLOCKED" }), MergeOptions{}, mergeWait, "checks running"},
		"running, auto":      {with(func(p *ghapi.PR) { p.Checks = "PENDING"; p.MergeState = "BLOCKED" }), MergeOptions{Auto: true}, mergeAuto, ""},
		"needs review":       {with(func(p *ghapi.PR) { p.Review = "REVIEW_REQUIRED"; p.MergeState = "BLOCKED" }), MergeOptions{}, mergeWait, "needs review"},
		"needs review, auto": {with(func(p *ghapi.PR) { p.Review = "REVIEW_REQUIRED"; p.MergeState = "BLOCKED" }), MergeOptions{Auto: true}, mergeAuto, ""},
		"unreviewed":         {with(func(p *ghapi.PR) { p.Review = "" }), MergeOptions{Auto: true}, mergeWait, "not approved; the repo doesn't require review, pass --allow-unreviewed"},
		"unreviewed allowed": {with(func(p *ghapi.PR) { p.Review = "" }), MergeOptions{AllowUnreviewed: true}, mergeNow, ""},
		"no checks":          {with(func(p *ghapi.PR) { p.Checks = "" }), MergeOptions{Auto: true}, mergeWait, "no checks ran; pass --allow-no-checks"},
		"no checks allowed":  {with(func(p *ghapi.PR) { p.Checks = "" }), MergeOptions{AllowNoChecks: true}, mergeNow, ""},
		"blocked":            {with(func(p *ghapi.PR) { p.MergeState = "BLOCKED" }), MergeOptions{}, mergeWait, "blocked by branch protection"},
		"unknown":            {with(func(p *ghapi.PR) { p.MergeState = "UNKNOWN" }), MergeOptions{}, mergeWait, "GitHub hasn't worked out if it can merge yet; run again"},
		"unknown, auto":      {with(func(p *ghapi.PR) { p.MergeState = "" }), MergeOptions{Auto: true}, mergeAuto, ""},
	} {
		action, reason := decideMerge(tt.pr, tt.o)
		if action != tt.action || reason != tt.reason {
			t.Errorf("%s: got %s %q, want %s %q", name, action, reason, tt.action, tt.reason)
		}
	}
}

// openPRs applies the basic campaign and returns its PR numbers by repo.
func (h *harness) openPRs(f string) {
	h.t.Helper()
	if err := h.app.Apply(context.Background(), ApplyOptions{Common: Common{File: f, Yes: true}}); err != nil {
		h.t.Fatal(err)
	}
	for _, p := range h.gh.OpenPRs() {
		h.gh.SetPRState(p.Repo, p.Number, ghapi.StateOpen, "SUCCESS", "APPROVED")
		h.gh.SetMergeState(p.Repo, p.Number, "CLEAN")
	}
}

func TestMerge(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	f := h.campaign(basic, files)
	ctx := context.Background()
	if err := h.app.Merge(ctx, MergeOptions{Common: Common{File: f, Yes: true}, Method: "squash"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.errOut.String(), "No open campaign PRs.") {
		t.Fatalf("output:\n%s", h.errOut)
	}
	h.openPRs(f)
	h.gh.SetPRState("acme/web", 1, ghapi.StateOpen, "PENDING", "APPROVED")
	h.gh.SetMergeState("acme/web", 1, "BLOCKED")

	// Dry run: nothing written, no prompt needed.
	h.reset()
	if err := h.app.Merge(ctx, MergeOptions{Common: Common{File: f}, Method: "squash", DryRun: true}); err != nil {
		t.Fatal(err)
	}
	if len(h.gh.Merges) != 0 || !strings.Contains(h.errOut.String(), "acme/api") || !strings.Contains(h.errOut.String(), "would merge") ||
		!strings.Contains(h.errOut.String(), "waiting: checks running") || !strings.Contains(h.errOut.String(), "1 would merge · 1 waiting") {
		t.Fatalf("dry run:\n%s %v", h.errOut, h.gh.Merges)
	}

	// Without --yes and no terminal, it refuses.
	if err := h.app.Merge(ctx, MergeOptions{Common: Common{File: f}, Method: "squash"}); err == nil {
		t.Fatal("merge without --yes should fail")
	}

	h.reset()
	if err := h.app.Merge(ctx, MergeOptions{Common: Common{File: f, Yes: true}, Method: "squash"}); err != nil {
		t.Fatal(err)
	}
	if strings.Join(h.gh.Merges, ",") != "acme/api#1:squash" || strings.Join(h.gh.DeletedBranches, ",") != "acme/api:fleet/add-line" {
		t.Fatalf("merges %v, deleted %v", h.gh.Merges, h.gh.DeletedBranches)
	}
	if !strings.Contains(h.errOut.String(), "1 merged · 1 waiting") {
		t.Fatalf("output:\n%s", h.errOut)
	}
	st, _ := h.app.Store.Load("add-line")
	if st.Get("acme/api").Status != state.Merged || st.Get("acme/web").Status != state.PROpen {
		t.Fatalf("state: %+v %+v", st.Get("acme/api"), st.Get("acme/web"))
	}

	// --auto turns on auto-merge for the one still running checks.
	h.reset()
	if err := h.app.Merge(ctx, MergeOptions{Common: Common{File: f, Yes: true, JSON: true}, Method: "rebase", Auto: true, KeepBranches: true}); err != nil {
		t.Fatal(err)
	}
	var res []MergeResult
	if err := json.Unmarshal(h.out.Bytes(), &res); err != nil || len(res) != 1 || res[0].Action != mergeAuto || !res[0].Done || h.gh.CallCount("EnableAutoMerge") != 1 {
		t.Fatalf("json = %s, %v", h.out, err)
	}
	h.reset()
	if err := h.app.Merge(ctx, MergeOptions{Common: Common{File: f, Yes: true}, Method: "rebase", Auto: true}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.errOut.String(), "waiting: auto-merge already on") {
		t.Fatalf("output:\n%s", h.errOut)
	}
}

func TestMergeMaxUpdateAndErrors(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	f := h.campaign(basic, files)
	ctx := context.Background()
	h.openPRs(f)
	for _, bad := range []MergeOptions{{Method: "fast-forward"}, {Method: "merge", Max: -1}} {
		bad.Common = Common{File: f, Yes: true}
		if err := h.app.Merge(ctx, bad); err == nil {
			t.Errorf("%+v accepted", bad)
		}
	}

	// --max 1 merges the first repo by name only.
	h.reset()
	if err := h.app.Merge(ctx, MergeOptions{Common: Common{File: f, Yes: true}, Method: "merge", Max: 1}); err != nil {
		t.Fatal(err)
	}
	if strings.Join(h.gh.Merges, ",") != "acme/api#1:merge" || !strings.Contains(h.errOut.String(), "waiting: --max 1 reached") {
		t.Fatalf("merges %v\n%s", h.gh.Merges, h.errOut)
	}

	// A branch behind its base is updated with --update-branch.
	h.gh.SetMergeState("acme/web", 1, "BEHIND")
	h.reset()
	if err := h.app.Merge(ctx, MergeOptions{Common: Common{File: f, Yes: true}, Method: "merge", UpdateBranch: true}); err != nil {
		t.Fatal(err)
	}
	if strings.Join(h.gh.UpdatedBranches, ",") != "acme/web#1" || !strings.Contains(h.errOut.String(), "branch updated") {
		t.Fatalf("updated %v\n%s", h.gh.UpdatedBranches, h.errOut)
	}

	// A failed merge fails the run; a failed branch delete after a merge
	// is only noted.
	h.gh.SetMergeState("acme/web", 1, "CLEAN")
	h.gh.Errors["MergePR:acme/web"] = errors.New("405 not mergeable")
	h.reset()
	if err := h.app.Merge(ctx, MergeOptions{Common: Common{File: f, Yes: true}, Method: "merge"}); !errors.Is(err, ErrFailed) {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(h.errOut.String(), "405 not mergeable") || !strings.Contains(h.errOut.String(), "1 failed") {
		t.Fatalf("output:\n%s", h.errOut)
	}
	delete(h.gh.Errors, "MergePR:acme/web")
	h.gh.Errors["DeleteBranch"] = errors.New("protected")
	h.reset()
	if err := h.app.Merge(ctx, MergeOptions{Common: Common{File: f, Yes: true}, Method: "merge"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.errOut.String(), "branch not deleted: protected") || !strings.Contains(h.errOut.String(), "1 merged") {
		t.Fatalf("output:\n%s", h.errOut)
	}

	h.gh.Errors["PRsByHead"] = errors.New("graphql down")
	if err := h.app.Merge(ctx, MergeOptions{Common: Common{File: f, Yes: true}, Method: "merge"}); err == nil {
		t.Fatal("lookup error swallowed")
	}
}
