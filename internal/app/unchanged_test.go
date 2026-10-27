package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestApplySkipsUnchangedRepos(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	now := time.Date(2026, 10, 26, 9, 0, 0, 0, time.UTC)
	h.app.Store.Now = func() time.Time { return now }
	h.app.Version = "v1.0.0"
	// The change only touches acme/api; acme/web is left as it is.
	only := "#!/bin/sh\nset -e\nif [ \"$FLEET_REPO\" = acme/api ]; then echo x >> README.md; fi\n"
	f := h.campaign(basic, map[string]string{"change.sh": only, "body.md": "Adds a line.\n"})
	ctx := context.Background()
	heads := func() {
		h.gh.Heads = map[string]string{}
		for _, r := range []string{"acme/api", "acme/web"} {
			h.gh.Heads[r+":main"] = h.remotes[r].Branch("main")
		}
	}
	heads()
	detail := func(repo string) string {
		t.Helper()
		st, err := h.app.Store.Load("add-line")
		if err != nil {
			t.Fatal(err)
		}
		if rs := st.Get(repo); rs != nil {
			return rs.Detail
		}
		return ""
	}
	apply := func(o ApplyOptions) {
		t.Helper()
		h.reset()
		o.File, o.Yes = f, true
		if err := h.app.Apply(ctx, o); err != nil {
			t.Fatal(err)
		}
	}

	if err := h.app.Plan(ctx, PlanOptions{Common: Common{File: f}}); err != nil {
		t.Fatal(err)
	}
	// apply opens api's PR and skips web, which the plan found unchanged.
	apply(ApplyOptions{})
	if !strings.Contains(h.errOut.String(), "Skipping 1 repo the last run found nothing to change in") {
		t.Fatalf("output:\n%s", h.errOut)
	}
	if d := detail("acme/web"); d != "unchanged since 9:00AM" {
		t.Fatalf("web detail = %q", d)
	}
	if len(h.gh.PRs) != 1 || h.gh.PRs[0].Repo != "acme/api" {
		t.Fatalf("PRs = %+v", h.gh.PRs)
	}

	// Skipping doesn't renew the record, so a day after the plan web runs
	// again, and that run renews it.
	now = now.Add(25 * time.Hour)
	apply(ApplyOptions{})
	if d := detail("acme/web"); d != "" {
		t.Fatalf("web skipped after a day: %q", d)
	}
	apply(ApplyOptions{})
	if d := detail("acme/web"); !strings.HasPrefix(d, "unchanged since") {
		t.Fatalf("web not skipped after a fresh run: %q", d)
	}

	// --recheck runs it anyway.
	apply(ApplyOptions{Recheck: true})
	if d := detail("acme/web"); d != "" {
		t.Fatalf("--recheck skipped web: %q", d)
	}

	// A new commit on its base means running it again.
	h.remotes["acme/web"].Commit("main", "docs", map[string]string{"NOTES.md": "x\n"})
	heads()
	apply(ApplyOptions{})
	if d := detail("acme/web"); d != "" {
		t.Fatalf("web skipped after its base moved: %q", d)
	}
	apply(ApplyOptions{})
	if d := detail("acme/web"); !strings.HasPrefix(d, "unchanged since") {
		t.Fatalf("web not skipped: %q", d)
	}

	// So does a change to the campaign's files, or a new gh fleet.
	if err := os.WriteFile(filepath.Join(h.dir, "body.md"), []byte("New body.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	apply(ApplyOptions{})
	if d := detail("acme/web"); d != "" {
		t.Fatalf("web skipped after the campaign changed: %q", d)
	}
	h.app.Version = "v1.1.0"
	apply(ApplyOptions{})
	if d := detail("acme/web"); d != "" {
		t.Fatalf("web skipped after an upgrade: %q", d)
	}

	// When the base can't be checked, every repo runs.
	apply(ApplyOptions{})
	h.gh.Errors["BranchHeads"] = errors.New("boom")
	apply(ApplyOptions{})
	if d := detail("acme/web"); d != "" || !strings.Contains(h.errOut.String(), "Couldn't check which repos are unchanged (boom)") {
		t.Fatalf("web detail %q, output:\n%s", d, h.errOut)
	}
}

func TestApplyAllUnchanged(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	// A change that edits nothing: every repo is unchanged.
	f := h.campaign(basic, map[string]string{"change.sh": "#!/bin/sh\ntrue\n", "body.md": "b\n"})
	ctx := context.Background()
	if err := h.app.Plan(ctx, PlanOptions{Common: Common{File: f}}); err != nil {
		t.Fatal(err)
	}
	h.gh.Heads = map[string]string{}
	for _, r := range []string{"acme/api", "acme/web"} {
		h.gh.Heads[r+":main"] = h.remotes[r].Branch("main")
	}
	h.reset()
	if err := h.app.Apply(ctx, ApplyOptions{Common: Common{File: f, Yes: true}}); err != nil {
		t.Fatal(err)
	}
	// Nothing is left to run, so there is nothing to confirm or clone.
	if !strings.Contains(h.errOut.String(), "2 noop") || strings.Contains(h.errOut.String(), "Applying") {
		t.Fatalf("output:\n%s", h.errOut)
	}
	h.reset()
	if err := h.app.Apply(ctx, ApplyOptions{Common: Common{File: f, Yes: true, JSON: true}}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.out.String(), `"unchanged since`) {
		t.Fatalf("json:\n%s", h.out)
	}
}
