package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/rossedman/gh-fleet/internal/campaign"
	"github.com/rossedman/gh-fleet/internal/ghapi"
	"github.com/rossedman/gh-fleet/internal/state"
)

func TestSplitWaves(t *testing.T) {
	t.Parallel()
	names := []string{"o/j", "o/b", "o/c", "o/d", "o/e", "o/f", "o/g", "o/h", "o/i", "o/a"}
	waves := []campaign.Wave{
		{Repos: []string{"o/h", "o/missing"}},
		{Size: campaign.WaveSize{N: 2}},
		{Repos: []string{"o/j"}, Size: campaign.WaveSize{N: 25, Percent: true}},
		{Repos: []string{"o/h"}},
	}
	got := fmt.Sprint(splitWaves(waves, names))
	if want := "[[o/h] [o/a o/b] [o/c o/d o/e o/j] [o/f o/g o/i]]"; got != want {
		t.Fatalf("waves = %s, want %s", got, want)
	}
	// A pattern takes every remaining target it matches.
	if got := fmt.Sprint(splitWaves([]campaign.Wave{{Repos: []string{"o/[a-c]"}}, {Repos: []string{"O/*"}, Size: campaign.WaveSize{N: 1}}}, names[:4])); got != "[[o/b o/c] [o/d o/j]]" {
		t.Errorf("pattern waves = %s", got)
	}
	if got := fmt.Sprint(splitWaves([]campaign.Wave{{Size: campaign.WaveSize{N: 50}}}, names[:3])); got != "[[o/b o/c o/j]]" {
		t.Errorf("oversized wave = %s", got)
	}
	if (campaign.WaveSize{N: 1, Percent: true}).Of(10) != 1 || (campaign.WaveSize{N: 30, Percent: true}).Of(10) != 3 || (campaign.WaveSize{N: 4}).Of(100) != 4 {
		t.Error("WaveSize.Of")
	}
}

func TestGateWaves(t *testing.T) {
	t.Parallel()
	names := []string{"o/a", "o/b", "o/c", "o/d"}
	waves := []campaign.Wave{{Repos: []string{"o/a"}}, {Size: campaign.WaveSize{N: 2}}}
	set := func(xs ...string) func(string) bool {
		return func(r string) bool { return strings.Contains(strings.Join(xs, ","), r) }
	}
	g := gateWaves(waves, names, set(), set())
	if g.Current != 1 || g.Waves != 3 || !g.Allowed["o/a"] || g.Allowed["o/b"] || g.Held["o/d"] != "wave 3 of 3: starts when wave 1 is done (0 of 1 merged or unchanged)" {
		t.Fatalf("start: %+v", g)
	}
	// An open PR in a later wave keeps being updated.
	g = gateWaves(waves, names, set(), set("o/c"))
	if !g.Allowed["o/c"] || g.Held["o/c"] != "" {
		t.Fatalf("open later PR: %+v", g)
	}
	g = gateWaves(waves, names, set("o/a", "o/b"), set())
	if g.Current != 2 || !g.Allowed["o/c"] || g.Allowed["o/d"] || !strings.Contains(g.Held["o/d"], "(1 of 2 merged or unchanged)") {
		t.Fatalf("wave 2: %+v", g)
	}
	g = gateWaves(waves, names, set("o/a", "o/b", "o/c", "o/d"), set())
	if g.Current != 3 || len(g.Held) != 0 || len(g.Allowed) != 4 {
		t.Fatalf("all done: %+v", g)
	}
}

func TestApplyWaves(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	f := h.campaign(strings.Replace(basic, "pr:\n", "limits:\n  waves:\n    - repos: [acme/web]\npr:\n", 1), files)
	ctx := context.Background()
	if err := h.app.Apply(ctx, ApplyOptions{Common: Common{File: f, Yes: true}}); err != nil {
		t.Fatal(err)
	}
	if prs := h.gh.OpenPRs(); len(prs) != 1 || prs[0].Repo != "acme/web" {
		t.Fatalf("wave 1 opened %+v", prs)
	}
	if !strings.Contains(h.errOut.String(), "Wave 1 of 2: 1 repo now; 1 repo waits for it.") || !strings.Contains(h.errOut.String(), "pending  acme/api") {
		t.Fatalf("output:\n%s", h.errOut)
	}
	st, _ := h.app.Store.Load("add-line")
	if rs := st.Get("acme/api"); rs.Status != state.Pending || !strings.Contains(rs.Detail, "wave 2 of 2: starts when wave 1 is done") {
		t.Fatalf("held state = %+v", rs)
	}

	// Nothing moves while the canary is open; --repo goes around waves.
	h.reset()
	if err := h.app.Apply(ctx, ApplyOptions{Common: Common{File: f, Yes: true}}); err != nil {
		t.Fatal(err)
	}
	if len(h.gh.OpenPRs()) != 1 {
		t.Fatal("wave 2 started early")
	}

	// retry keeps only the held repo, which still waits.
	h.reset()
	if err := h.app.Retry(ctx, ApplyOptions{Common: Common{File: f, Yes: true}}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.errOut.String(), "Nothing to do until the current wave is done.") || len(h.gh.OpenPRs()) != 1 {
		t.Fatalf("retry during wave 1:\n%s", h.errOut)
	}

	// Merge the canary: the next apply opens wave 2.
	h.gh.SetPRState("acme/web", 1, ghapi.StateMerged, "SUCCESS", "APPROVED")
	h.reset()
	if err := h.app.Apply(ctx, ApplyOptions{Common: Common{File: f, Yes: true}}); err != nil {
		t.Fatal(err)
	}
	if prs := h.gh.OpenPRs(); len(prs) != 1 || prs[0].Repo != "acme/api" || !strings.Contains(h.errOut.String(), "Wave 2 of 2") {
		t.Fatalf("wave 2: %+v\n%s", prs, h.errOut)
	}

	// A canary closed without merging stops the rollout.
	h2 := newHarness(t)
	f2 := h2.campaign(strings.Replace(basic, "pr:\n", "limits:\n  waves:\n    - repos: [acme/nope]\n      size: 1\npr:\n", 1), files)
	if err := h2.app.Apply(ctx, ApplyOptions{Common: Common{File: f2, Yes: true}}); err != nil {
		t.Fatal(err)
	}
	h2.gh.SetPRState("acme/api", 1, ghapi.StateClosed, "", "")
	h2.reset()
	if err := h2.app.Apply(ctx, ApplyOptions{Common: Common{File: f2, Yes: true}}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h2.errOut.String(), "pending  acme/web wave 2 of 2") || len(h2.gh.OpenPRs()) != 0 {
		t.Fatalf("closed canary should stop the rollout:\n%s", h2.errOut)
	}
	if err := h2.app.Apply(ctx, ApplyOptions{Common: Common{File: f2, Yes: true, Repos: []string{"acme/web"}}}); err != nil || len(h2.gh.OpenPRs()) != 1 {
		t.Fatalf("--repo should bypass waves: %v %+v", err, h2.gh.OpenPRs())
	}
}

func TestPlanWaves(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	f := h.campaign(strings.Replace(basic, "pr:\n", "limits:\n  waves:\n    - repos: [acme/web]\npr:\n", 1), files)
	ctx := context.Background()
	plan := func(o PlanOptions) []state.RepoState {
		t.Helper()
		h.reset()
		o.File, o.JSON = f, true
		if err := h.app.Plan(ctx, o); err != nil {
			t.Fatal(err)
		}
		var out []state.RepoState
		if err := json.Unmarshal(h.out.Bytes(), &out); err != nil {
			t.Fatalf("%v: %s", err, h.out)
		}
		return out
	}
	repos := func(rs []state.RepoState) string {
		var names []string
		for _, r := range rs {
			if r.Status != state.Skipped {
				names = append(names, r.Repo)
			}
		}
		return strings.Join(names, ",")
	}

	// Before any apply, wave 1 is current: only the canary is planned.
	if got := repos(plan(PlanOptions{})); got != "acme/web" {
		t.Fatalf("current wave planned %s", got)
	}
	if got := repos(plan(PlanOptions{Wave: 2})); got != "acme/api" {
		t.Fatalf("--wave 2 planned %s", got)
	}
	if got := repos(plan(PlanOptions{AllWaves: true})); got != "acme/api,acme/web" {
		t.Fatalf("--all-waves planned %s", got)
	}
	// Once the canary merges, wave 2 is current, as apply sees it.
	if err := h.app.Apply(ctx, ApplyOptions{Common: Common{File: f, Yes: true}}); err != nil {
		t.Fatal(err)
	}
	h.gh.SetPRState("acme/web", 1, ghapi.StateMerged, "SUCCESS", "APPROVED")
	if got := repos(plan(PlanOptions{})); got != "acme/api" {
		t.Fatalf("after the canary merged, planned %s", got)
	}

	// People see which wave is planned.
	h.reset()
	if err := h.app.Plan(ctx, PlanOptions{Common: Common{File: f}, Wave: 1}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.errOut.String(), "Wave 1 of 2: planning 1 repo; 1 repo in other waves (--wave N or --all-waves plans them)") {
		t.Fatalf("output:\n%s", h.errOut)
	}

	// Mistakes are errors.
	for _, tc := range []struct {
		o    PlanOptions
		want string
	}{
		{PlanOptions{Wave: 3}, "--wave 3: the campaign's targets make 2 waves"},
		{PlanOptions{Wave: 1, AllWaves: true}, "--wave and --all-waves can't be combined"},
		{PlanOptions{Common: Common{Repos: []string{"acme/api"}}, Wave: 1}, "--repo plans the repos you name"},
	} {
		tc.o.File = f
		if err := h.app.Plan(ctx, tc.o); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%+v: err = %v", tc.o, err)
		}
	}
	// Without waves, --wave is an error and plan covers every target.
	plain := h.campaign(basic, files)
	if err := h.app.Plan(ctx, PlanOptions{Common: Common{File: plain}, Wave: 1}); err == nil || !strings.Contains(err.Error(), "--wave needs limits.waves") {
		t.Errorf("err = %v", err)
	}
	f = plain
	if got := repos(plan(PlanOptions{})); got != "acme/api,acme/web" {
		t.Fatalf("no waves planned %s", got)
	}
	// A failed PR lookup fails the plan.
	f = h.campaign(strings.Replace(basic, "pr:\n", "limits:\n  waves:\n    - repos: [acme/web]\npr:\n", 1), files)
	h.gh.Errors["PRsByHead"] = errors.New("boom")
	if err := h.app.Plan(ctx, PlanOptions{Common: Common{File: f}}); err == nil {
		t.Error("PR lookup error swallowed")
	}
}
