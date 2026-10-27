package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rossedman/gh-fleet/internal/ghapi"
	"github.com/rossedman/gh-fleet/internal/ghapi/fake"
	"github.com/rossedman/gh-fleet/internal/state"
	"github.com/rossedman/gh-fleet/internal/target"
	"github.com/rossedman/gh-fleet/internal/testutil"
	"github.com/rossedman/gh-fleet/internal/ui"
	"github.com/rossedman/gh-fleet/internal/workspace"
)

// harness is an App wired to a fake GitHub and local bare repos.
type harness struct {
	t       *testing.T
	gh      *fake.Client
	remotes map[string]*testutil.Remote
	app     *App
	out     *bytes.Buffer
	errOut  *bytes.Buffer
	dir     string
	env     []string
	answer  bool
}

// newHarness creates repos acme/api, acme/web (push), acme/docs (no
// push), acme/old (archived) and a campaign directory.
func newHarness(t *testing.T) *harness {
	t.Helper()
	h := &harness{t: t, gh: fake.New(), remotes: map[string]*testutil.Remote{}, out: &bytes.Buffer{}, errOut: &bytes.Buffer{}, dir: t.TempDir()}
	h.env = testutil.GitEnv(t)
	for _, r := range []ghapi.Repo{
		{Owner: "acme", Name: "api", Topics: []string{"service"}, Language: "Go", CanPush: true},
		{Owner: "acme", Name: "web", Topics: []string{"service"}, Language: "TypeScript", CanPush: true},
		{Owner: "acme", Name: "docs", Topics: []string{"service"}, Language: "Go", CanPush: false},
		{Owner: "acme", Name: "old", Topics: []string{"service"}, Language: "Go", CanPush: true, Archived: true},
	} {
		h.gh.AddRepo(r)
		h.remotes[r.FullName()] = testutil.NewRemote(t, h.env, map[string]string{"README.md": "# " + r.Name + "\n"})
	}
	h.gh.Files["acme/api"] = []string{"README.md", "go.mod"}
	h.gh.Files["acme/web"] = []string{"README.md"}
	h.app = &App{
		TTY: true,
		Out: h.out, Err: h.errOut, In: strings.NewReader(""),
		NewClient: func(float64) (ghapi.Client, error) { return h.gh, nil },
		Store:     &state.Store{Dir: filepath.Join(t.TempDir(), "state")},
		Cache:     &target.Cache{Dir: filepath.Join(t.TempDir(), "cache")},
		Workspaces: &workspace.Manager{
			Root:     filepath.Join(t.TempDir(), "work"),
			CloneURL: func(r ghapi.Repo) string { return h.remotes[r.FullName()].URL() },
		},
		Git:     testutil.Git(h.env),
		Confirm: func(string) (bool, error) { return h.answer, nil },
	}
	return h
}

// campaign writes fleet.yaml (and extra files) and returns its path.
func (h *harness) campaign(yaml string, extra map[string]string) string {
	h.t.Helper()
	for name, content := range extra {
		p := filepath.Join(h.dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			h.t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o755); err != nil {
			h.t.Fatal(err)
		}
	}
	p := filepath.Join(h.dir, "fleet.yaml")
	if err := os.WriteFile(p, []byte(yaml), 0o644); err != nil {
		h.t.Fatal(err)
	}
	return p
}

// reset clears captured output.
func (h *harness) reset() {
	h.out.Reset()
	h.errOut.Reset()
}

// basic is a campaign appending a line to README.md in every Go or
// TypeScript service repo; the script reads LINE from the campaign env.
const basic = `apiVersion: fleet.gh/v1
name: add-line
targets:
  orgs: [acme]
  include:
    topics: [service]
change:
  steps:
    - run: ./change.sh
  env: {LINE: "managed by fleet"}
pr:
  title: "chore: add line to {{ .Repo.Name }}"
  body: ./body.md
  labels: [fleet]
  reviewers: [octocat, acme/platform]
`

// script appends $LINE to README.md unless it is already there.
const script = `#!/bin/sh
set -e
test "$FLEET_CAMPAIGN" = add-line
grep -qx "$LINE" README.md || echo "$LINE" >> README.md
`

// files is the extra content of the basic campaign.
var files = map[string]string{"change.sh": script, "body.md": "Adds a line to {{ .Repo.FullName }}.\n"}

func TestTargets(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	f := h.campaign(basic, files)
	if err := h.app.Targets(context.Background(), Common{File: f}); err != nil {
		t.Fatal(err)
	}
	out := h.out.String() + h.errOut.String()
	for _, want := range []string{"acme/api", "topic:service", "acme/web", "SKIPPED", "acme/docs", "no push access", "2 repos (1 skipped)"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "acme/old") {
		t.Error("archived repo listed")
	}

	h.reset()
	if err := h.app.Targets(context.Background(), Common{File: f, JQ: ".targets | length"}); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(h.out.String()) != "2" {
		t.Fatalf("jq output = %q", h.out.String())
	}
	// The campaign filters on a topic, so each run asks repo search for
	// the matches rather than listing the org.
	if l, n := h.gh.CallCount("ListOwnerRepos"), h.gh.CallCount("SearchRepos"); l != 0 || n != 2 {
		t.Fatalf("ListOwnerRepos called %d times, SearchRepos %d", l, n)
	}

	h.reset()
	if err := h.app.Targets(context.Background(), Common{File: f, Repos: []string{"acme/nope"}}); err == nil {
		t.Fatal("expected error for a repo that is not a target")
	}
}

func TestPlanShowsDiffsWithoutWrites(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	// web already has the line, so it is a no-op.
	h.remotes["acme/web"].Commit("main", "add line", map[string]string{"README.md": "# web\nmanaged by fleet\n"})
	f := h.campaign(basic, files)
	if err := h.app.Plan(context.Background(), PlanOptions{Common: Common{File: f}}); err != nil {
		t.Fatal(err)
	}
	out := h.out.String()
	if !strings.Contains(out, "── acme/api") || !strings.Contains(out, "+managed by fleet") {
		t.Fatalf("diff missing:\n%s", out)
	}
	prog := h.errOut.String()
	if !strings.Contains(prog, "noop") || !strings.Contains(prog, "acme/web") || !strings.Contains(prog, "1 changed · 1 noop · 1 skipped") {
		t.Fatalf("progress:\n%s", prog)
	}
	if len(h.gh.PRs) != 0 || h.remotes["acme/api"].Branch("fleet/add-line") != "" {
		t.Fatal("plan wrote to GitHub or pushed")
	}
	for _, m := range []string{"CreatePR", "UpdatePR", "AddLabels", "Comment"} {
		if h.gh.CallCount(m) != 0 {
			t.Fatalf("plan called %s", m)
		}
	}

	h.reset()
	if err := h.app.Plan(context.Background(), PlanOptions{Common: Common{File: f, JSON: true}}); err != nil {
		t.Fatal(err)
	}
	var results []state.RepoState
	if err := json.Unmarshal(h.out.Bytes(), &results); err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 || results[0].Repo != "acme/api" || results[0].Diff == "" {
		t.Fatalf("json results: %+v", results)
	}
}

func TestPlanReportsDryRun(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	f := h.campaign(strings.Replace(basic, "./change.sh", `"test \"$FLEET_DRY_RUN\" = true && echo dry > dry.txt"`, 1), files)
	if err := h.app.Plan(context.Background(), PlanOptions{Common: Common{File: f, Repos: []string{"acme/api"}}}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.out.String(), "+dry") {
		t.Fatalf("FLEET_DRY_RUN not true in plan:\n%s", h.out.String())
	}
}

func TestApplyTwiceIsIdempotent(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	f := h.campaign(basic, files)
	ctx := context.Background()
	if err := h.app.Apply(ctx, ApplyOptions{Common: Common{File: f, Yes: true}}); err != nil {
		t.Fatal(err)
	}
	if len(h.gh.PRs) != 2 {
		t.Fatalf("want 2 PRs, got %d", len(h.gh.PRs))
	}
	api := h.remotes["acme/api"]
	if got := api.Show("fleet/add-line", "README.md"); got != "# api\nmanaged by fleet" {
		t.Fatalf("branch content = %q", got)
	}
	if got := api.Log("fleet/add-line")[0]; got != "chore: add line to api" {
		t.Fatalf("commit message = %q", got)
	}
	pr := h.gh.PRs[0]
	if pr.Repo != "acme/api" {
		pr = h.gh.PRs[1]
	}
	if pr.Title != "chore: add line to api" || !strings.Contains(pr.Body, "Adds a line to acme/api.") ||
		!strings.Contains(pr.Body, "<!-- gh-fleet:campaign=add-line -->") || pr.BaseRef != "main" || pr.Labels[0] != "fleet" {
		t.Fatalf("PR = %+v", pr)
	}
	if got := h.gh.Reviewers["acme/api#1"]; len(got) != 2 {
		t.Fatalf("reviewers = %v", got)
	}
	if !strings.Contains(h.errOut.String(), "2 pr open · 1 skipped") {
		t.Fatalf("summary:\n%s", h.errOut.String())
	}
	sha := api.Branch("fleet/add-line")

	// Second run: nothing changed, so no new PRs and no new push.
	h.reset()
	if err := h.app.Apply(ctx, ApplyOptions{Common: Common{File: f, Yes: true}}); err != nil {
		t.Fatal(err)
	}
	if len(h.gh.PRs) != 2 || api.Branch("fleet/add-line") != sha {
		t.Fatalf("second apply duplicated PRs or re-pushed: %d PRs", len(h.gh.PRs))
	}
	if !strings.Contains(h.errOut.String(), "unchanged") {
		t.Fatalf("progress:\n%s", h.errOut.String())
	}

	// Third run with an edited campaign: the existing PRs are updated.
	h.reset()
	edited := strings.Replace(basic, "chore: add line", "docs: add line", 1)
	edited = strings.Replace(edited, "managed by fleet", "owned by platform", 1)
	h.campaign(edited, files)
	if err := h.app.Apply(ctx, ApplyOptions{Common: Common{File: f, Yes: true}}); err != nil {
		t.Fatal(err)
	}
	if len(h.gh.PRs) != 2 || !strings.HasPrefix(h.gh.PRs[0].Title, "docs: add line to ") {
		t.Fatalf("PR not updated: %+v", h.gh.PRs[0])
	}
	if got := api.Show("fleet/add-line", "README.md"); !strings.Contains(got, "owned by platform") {
		t.Fatalf("branch not updated: %q", got)
	}
	if !strings.Contains(h.errOut.String(), "updated") {
		t.Fatalf("progress:\n%s", h.errOut.String())
	}

	// State records the PRs.
	st, err := h.app.Store.Load("add-line")
	if err != nil {
		t.Fatal(err)
	}
	if rs := st.Get("acme/api"); rs == nil || rs.Status != state.PROpen || rs.PRNumber != 1 || rs.HeadSHA == "" {
		t.Fatalf("state = %+v", rs)
	}
	if st.File != f {
		t.Fatalf("state file = %q", st.File)
	}
}

func TestApplyOneFailureDoesNotStopOthers(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	f := h.campaign(basic, map[string]string{
		"change.sh": "#!/bin/sh\nif [ \"$FLEET_REPO\" = acme/web ]; then echo boom >&2; exit 3; fi\necho x >> README.md\n",
		"body.md":   "b",
	})
	err := h.app.Apply(context.Background(), ApplyOptions{Common: Common{File: f, Yes: true}})
	var fe *FailedError
	if !errors.As(err, &fe) || fe.N != 1 || !errors.Is(err, ErrFailed) {
		t.Fatalf("err = %v", err)
	}
	if len(h.gh.PRs) != 1 || h.gh.PRs[0].Repo != "acme/api" {
		t.Fatalf("PRs = %v", h.gh.PRs)
	}
	if !strings.Contains(h.errOut.String(), "exited with status 3: boom") {
		t.Fatalf("progress:\n%s", h.errOut.String())
	}
	st, _ := h.app.Store.Load("add-line")
	if rs := st.Get("acme/web"); rs.Status != state.Failed || rs.LastError == "" {
		t.Fatalf("state = %+v", rs)
	}
	log, err := os.ReadFile(h.app.Store.LogPath("add-line", "acme/web"))
	if err != nil || !strings.Contains(string(log), "boom") {
		t.Fatalf("log = %q, %v", log, err)
	}

	// Retry re-runs only the failed repo once the script is fixed.
	h.reset()
	h.campaign(basic, files)
	if err := h.app.Retry(context.Background(), ApplyOptions{Common: Common{File: f, Yes: true}}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(h.errOut.String(), "acme/api") || !strings.Contains(h.errOut.String(), "acme/web") {
		t.Fatalf("retry touched the wrong repos:\n%s", h.errOut.String())
	}
	if len(h.gh.PRs) != 2 {
		t.Fatalf("PRs = %d", len(h.gh.PRs))
	}

	// Nothing is left to retry.
	h.reset()
	if err := h.app.Retry(context.Background(), ApplyOptions{Common: Common{File: f, Yes: true}}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.errOut.String(), "Nothing to do.") {
		t.Fatalf("retry output:\n%s", h.errOut.String())
	}
}

func TestApplyNeedsConfirmation(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	f := h.campaign(basic, files)
	err := h.app.Apply(context.Background(), ApplyOptions{Common: Common{File: f}})
	if err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("err = %v", err)
	}
	h.app.Interactive = true
	h.answer = false
	if err := h.app.Apply(context.Background(), ApplyOptions{Common: Common{File: f}}); !errors.Is(err, ui.ErrNotConfirmed) {
		t.Fatalf("err = %v", err)
	}
	if len(h.gh.PRs) != 0 {
		t.Fatal("PRs opened without confirmation")
	}
	h.answer = true
	if err := h.app.Apply(context.Background(), ApplyOptions{Common: Common{File: f}}); err != nil {
		t.Fatal(err)
	}
	if len(h.gh.PRs) != 2 {
		t.Fatalf("PRs = %d", len(h.gh.PRs))
	}
}

func TestApplyMergedClosedAndReopen(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	f := h.campaign(basic, files)
	ctx := context.Background()
	if err := h.app.Apply(ctx, ApplyOptions{Common: Common{File: f, Yes: true}}); err != nil {
		t.Fatal(err)
	}
	h.gh.SetPRState("acme/api", 1, ghapi.StateMerged, "", "")
	h.gh.SetPRState("acme/web", 1, ghapi.StateClosed, "", "")
	h.reset()
	if err := h.app.Apply(ctx, ApplyOptions{Common: Common{File: f, Yes: true}}); err != nil {
		t.Fatal(err)
	}
	out := h.errOut.String()
	if !strings.Contains(out, "PR already merged") || !strings.Contains(out, "--reopen") {
		t.Fatalf("progress:\n%s", out)
	}
	h.reset()
	if err := h.app.Apply(ctx, ApplyOptions{Common: Common{File: f, Yes: true}, Reopen: true}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.errOut.String(), "reopened") {
		t.Fatalf("progress:\n%s", h.errOut.String())
	}
	if prs := h.gh.OpenPRs(); len(prs) != 1 || prs[0].Repo != "acme/web" {
		t.Fatalf("open PRs = %v", prs)
	}
}

func TestApplyCloseNoop(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	f := h.campaign(basic, files)
	ctx := context.Background()
	if err := h.app.Apply(ctx, ApplyOptions{Common: Common{File: f, Yes: true, Repos: []string{"acme/api"}}}); err != nil {
		t.Fatal(err)
	}
	// The change lands on main another way.
	h.remotes["acme/api"].Commit("main", "manual", map[string]string{"README.md": "# api\nmanaged by fleet\n"})
	h.reset()
	if err := h.app.Apply(ctx, ApplyOptions{Common: Common{File: f, Yes: true, Repos: []string{"acme/api"}}}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.errOut.String(), "--close-noop") {
		t.Fatalf("progress:\n%s", h.errOut.String())
	}
	h.reset()
	if err := h.app.Apply(ctx, ApplyOptions{Common: Common{File: f, Yes: true, Repos: []string{"acme/api"}}, CloseNoop: true}); err != nil {
		t.Fatal(err)
	}
	if len(h.gh.OpenPRs()) != 0 || len(h.gh.Comments["acme/api#1"]) != 1 || len(h.gh.DeletedBranches) != 1 {
		t.Fatalf("PR not closed: %v %v", h.gh.Comments, h.gh.DeletedBranches)
	}
}

func TestApplyMaxOpenPRs(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	f := h.campaign(basic+"limits:\n  maxOpenPRs: 1\n  concurrency: 1\n", files)
	if err := h.app.Apply(context.Background(), ApplyOptions{Common: Common{File: f, Yes: true}}); err != nil {
		t.Fatal(err)
	}
	if len(h.gh.PRs) != 1 {
		t.Fatalf("PRs = %d", len(h.gh.PRs))
	}
	if !strings.Contains(h.errOut.String(), "maxOpenPRs reached (1); not pushed") {
		t.Fatalf("progress:\n%s", h.errOut.String())
	}
	if h.remotes["acme/web"].Branch("fleet/add-line") != "" {
		t.Fatal("branch pushed past the limit")
	}
}

func TestApplyRebasesAndResetsExistingBranch(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	f := h.campaign(basic, files)
	ctx := context.Background()
	api := h.remotes["acme/api"]
	// An old campaign branch with a manual commit that rebases cleanly.
	api.Commit("fleet/add-line", "manual tweak", map[string]string{"NOTES.md": "keep me\n"})
	api.Commit("main", "unrelated", map[string]string{"main.go": "package main\n"})
	if err := h.app.Apply(ctx, ApplyOptions{Common: Common{File: f, Yes: true, Repos: []string{"acme/api"}}}); err != nil {
		t.Fatal(err)
	}
	if api.Show("fleet/add-line", "NOTES.md") != "keep me" || api.Show("fleet/add-line", "main.go") != "package main" {
		t.Fatal("rebased branch lost content")
	}
	// Now make the branch conflict with main: it is reset and re-run.
	api.Commit("fleet/add-line", "conflict", map[string]string{"main.go": "package other\n"})
	api.Commit("main", "change main.go", map[string]string{"main.go": "package changed\n"})
	h.reset()
	if err := h.app.Apply(ctx, ApplyOptions{Common: Common{File: f, Yes: true, Repos: []string{"acme/api"}}}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.errOut.String(), "branch reset after rebase conflict") {
		t.Fatalf("progress:\n%s", h.errOut.String())
	}
	if api.Show("fleet/add-line", "main.go") != "package changed" {
		t.Fatal("branch not reset to main")
	}
	if len(h.gh.PRs) != 1 {
		t.Fatalf("PRs = %d", len(h.gh.PRs))
	}
}

func TestApplyCommitMessageOverride(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	f := h.campaign(basic, map[string]string{
		"change.sh": "#!/bin/sh\necho y >> README.md\necho 'custom: message' > \"$FLEET_OUTPUT/commit-message\"\n",
		"body.md":   "b",
	})
	if err := h.app.Apply(context.Background(), ApplyOptions{Common: Common{File: f, Yes: true, Repos: []string{"acme/api"}}}); err != nil {
		t.Fatal(err)
	}
	if got := h.remotes["acme/api"].Log("fleet/add-line")[0]; got != "custom: message" {
		t.Fatalf("commit = %q", got)
	}
}

func TestApplyStopsAtQuota(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	f := h.campaign(basic+"limits: {concurrency: 1}\n", files)
	h.gh.Errors["CreatePR"] = &ghapi.QuotaError{Resource: "core", Remaining: 10, Limit: 5000}
	err := h.app.Apply(context.Background(), ApplyOptions{Common: Common{File: f, Yes: true}})
	if !errors.Is(err, ghapi.ErrQuotaLow) {
		t.Fatalf("err = %v", err)
	}
	st, _ := h.app.Store.Load("add-line")
	if rs := st.Get("acme/web"); rs == nil || rs.Status != state.Pending {
		t.Fatalf("second repo should not have started: %+v", rs)
	}
}

func TestApplyPRErrors(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	f := h.campaign(basic, files)
	h.gh.Errors["CreatePR:acme/web"] = errors.New("validation failed")
	err := h.app.Apply(context.Background(), ApplyOptions{Common: Common{File: f, Yes: true}})
	if !errors.Is(err, ErrFailed) {
		t.Fatalf("err = %v", err)
	}
	h.gh.Errors = map[string]error{"PRsByHead": errors.New("graphql down")}
	if err := h.app.Apply(context.Background(), ApplyOptions{Common: Common{File: f, Yes: true}}); err == nil {
		t.Fatal("expected index error")
	}
}

func TestStatusAfterStateLoss(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	f := h.campaign(basic, files)
	ctx := context.Background()
	if err := h.app.Apply(ctx, ApplyOptions{Common: Common{File: f, Yes: true}}); err != nil {
		t.Fatal(err)
	}
	h.gh.SetPRState("acme/api", 1, ghapi.StateMerged, "SUCCESS", "APPROVED")
	h.gh.SetPRState("acme/web", 1, ghapi.StateOpen, "FAILURE", "REVIEW_REQUIRED")
	h.gh.Limits = []ghapi.RateLimit{{Resource: "core", Limit: 5000, Remaining: 4812}, {Resource: "graphql", Limit: 5000, Remaining: 4950}, {Resource: "search", Limit: 30, Remaining: 30}}
	if err := os.RemoveAll(h.app.Store.Dir); err != nil {
		t.Fatal(err)
	}
	h.reset()
	if err := h.app.Status(ctx, StatusOptions{Common: Common{File: f}}); err != nil {
		t.Fatal(err)
	}
	out := h.out.String() + h.errOut.String()
	for _, want := range []string{"acme/api", "merged", "success", "approved", "acme/web", "pr open", "failure", "review required", "acme/docs", "skipped", "1 pr open · 1 merged · 1 skipped", "core 4,812/5,000", "graphql 4,950/5,000"} {
		if !strings.Contains(out, want) {
			t.Errorf("status missing %q:\n%s", want, out)
		}
	}
	h.reset()
	if err := h.app.Status(ctx, StatusOptions{Common: Common{File: f, JSON: true}}); err != nil {
		t.Fatal(err)
	}
	var rep StatusReport
	if err := json.Unmarshal(h.out.Bytes(), &rep); err != nil {
		t.Fatal(err)
	}
	if rep.Counts["merged"] != 1 || rep.Counts["pr_open"] != 1 || rep.Campaign != "add-line" {
		t.Fatalf("report = %+v", rep)
	}
}

func TestStatusFindsPRsByMarker(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	f := h.campaign(basic, files)
	// A PR on another branch that carries the campaign marker.
	h.gh.PRs = append(h.gh.PRs, &ghapi.PR{Repo: "acme/api", Number: 9, URL: "https://github.com/acme/api/pull/9", State: ghapi.StateOpen, HeadRef: "old-branch", Body: "x\n<!-- gh-fleet:campaign=add-line -->"})
	if err := h.app.Status(context.Background(), StatusOptions{Common: Common{File: f}}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.out.String(), "pull/9") {
		t.Fatalf("status:\n%s", h.out.String())
	}
	if h.gh.CallCount("SearchCampaignPRs") != 1 {
		t.Fatal("marker search not used")
	}
}

func TestStatusDropsVanishedPR(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	f := h.campaign(basic, files)
	st, _ := h.app.Store.Load("add-line")
	st.Set(state.RepoState{Repo: "acme/api", Status: state.PROpen, PRNumber: 3, PRURL: "u"}, h.app.Store.Time())
	st.Set(state.RepoState{Repo: "acme/gone", Status: state.PROpen, PRNumber: 4, PRURL: "u"}, h.app.Store.Time())
	if err := h.app.Store.Save(st); err != nil {
		t.Fatal(err)
	}
	if err := h.app.Status(context.Background(), StatusOptions{Common: Common{File: f}}); err != nil {
		t.Fatal(err)
	}
	st, _ = h.app.Store.Load("add-line")
	if rs := st.Get("acme/api"); rs.PRNumber != 0 || rs.Status != state.Pending {
		t.Fatalf("state = %+v", rs)
	}
}

func TestClose(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	f := h.campaign(basic, files)
	ctx := context.Background()
	if err := h.app.Close(ctx, CloseOptions{Common: Common{File: f, Yes: true}}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.errOut.String(), "No open campaign PRs.") {
		t.Fatalf("close output:\n%s", h.errOut.String())
	}
	if err := h.app.Apply(ctx, ApplyOptions{Common: Common{File: f, Yes: true}}); err != nil {
		t.Fatal(err)
	}
	if err := h.app.Close(ctx, CloseOptions{Common: Common{File: f}}); err == nil {
		t.Fatal("close without --yes and no terminal should fail")
	}
	h.reset()
	if err := h.app.Close(ctx, CloseOptions{Common: Common{File: f, Yes: true}, Comment: "bye"}); err != nil {
		t.Fatal(err)
	}
	if len(h.gh.OpenPRs()) != 0 || len(h.gh.DeletedBranches) != 2 || h.gh.Comments["acme/web#1"][0] != "bye" {
		t.Fatalf("not closed: %v %v", h.gh.DeletedBranches, h.gh.Comments)
	}
	if !strings.Contains(h.errOut.String(), "2 closed") {
		t.Fatalf("close output:\n%s", h.errOut.String())
	}
	st, _ := h.app.Store.Load("add-line")
	if st.Get("acme/api").Status != state.Closed {
		t.Fatal("state not updated")
	}
}

func TestCloseErrorsAndJSON(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	f := h.campaign(basic, files)
	ctx := context.Background()
	if err := h.app.Apply(ctx, ApplyOptions{Common: Common{File: f, Yes: true}}); err != nil {
		t.Fatal(err)
	}
	h.gh.Errors["UpdatePR:acme/web"] = errors.New("nope")
	h.reset()
	err := h.app.Close(ctx, CloseOptions{Common: Common{File: f, Yes: true, JSON: true}, KeepBranches: true})
	if !errors.Is(err, ErrFailed) {
		t.Fatalf("err = %v", err)
	}
	var results []state.RepoState
	if err := json.Unmarshal(h.out.Bytes(), &results); err != nil || len(results) != 2 {
		t.Fatalf("json = %s, %v", h.out.String(), err)
	}
	if len(h.gh.DeletedBranches) != 0 {
		t.Fatal("branches deleted with --keep-branches")
	}
}

func TestExec(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	ctx := context.Background()
	o := ExecOptions{Argv: []string{"head", "-1", "README.md"}, Orgs: []string{"acme"}, Topics: []string{"service"}}
	if err := h.app.Exec(ctx, o); err != nil {
		t.Fatal(err)
	}
	out := h.out.String()
	// docs is read-only but still included; old is archived.
	for _, want := range []string{"REPOS", "# api", "# docs", "# web"} {
		if !strings.Contains(out, want) {
			t.Errorf("exec output missing %q:\n%s", want, out)
		}
	}
	if !strings.Contains(h.errOut.String(), "3 done") {
		t.Fatalf("summary: %s", h.errOut.String())
	}

	h.reset()
	o.Raw = true
	if err := h.app.Exec(ctx, o); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.out.String(), "acme/api: # api") {
		t.Fatalf("raw output:\n%s", h.out.String())
	}

	h.reset()
	o = ExecOptions{Common: Common{JSON: true, Repos: []string{"acme/api", "acme/web"}}, Argv: []string{"sh", "-c", "echo same; test -f go.mod"}}
	h.remotes["acme/api"].Commit("main", "go", map[string]string{"go.mod": "module x\n"})
	err := h.app.Exec(ctx, o)
	if !errors.Is(err, ErrFailed) {
		t.Fatalf("err = %v", err)
	}
	var rep ExecReport
	if err := json.Unmarshal(h.out.Bytes(), &rep); err != nil {
		t.Fatal(err)
	}
	if len(rep.Tally) != 1 || rep.Tally[0].Count != 2 || rep.Tally[0].Line != "same" {
		t.Fatalf("tally = %+v", rep.Tally)
	}

	if err := h.app.Exec(ctx, ExecOptions{}); err == nil {
		t.Fatal("expected error without a command")
	}
	if err := h.app.Exec(ctx, ExecOptions{Argv: []string{"true"}}); err == nil {
		t.Fatal("expected error without targets")
	}
}

func TestExecFromCampaign(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	f := h.campaign(basic, files)
	if err := h.app.Exec(context.Background(), ExecOptions{Campaign: f, Common: Common{Repos: []string{"acme/web"}}, Argv: []string{"cat", "README.md"}}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.out.String(), "# web") || strings.Contains(h.out.String(), "# api") {
		t.Fatalf("exec output:\n%s", h.out.String())
	}
}

func TestListAndCampaignByName(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	if err := h.app.List(Common{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.errOut.String(), "No campaigns yet") || h.out.Len() != 0 {
		t.Fatalf("list: %q %q", h.out, h.errOut)
	}
	h.reset()
	if err := h.app.List(Common{JSON: true}); err != nil || strings.TrimSpace(h.out.String()) != "[]" {
		t.Fatalf("list json = %q, %v", h.out.String(), err)
	}
	f := h.campaign(basic, files)
	if err := h.app.Apply(context.Background(), ApplyOptions{Common: Common{File: f, Yes: true}}); err != nil {
		t.Fatal(err)
	}
	h.reset()
	if err := h.app.List(Common{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.out.String(), "add-line") || !strings.Contains(h.out.String(), "2 pr open · 1 skipped") {
		t.Fatalf("list:\n%s", h.out.String())
	}
	// A campaign name from state works in place of --file.
	h.reset()
	if err := h.app.Targets(context.Background(), Common{Campaign: "add-line"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.errOut.String(), "2 repos") {
		t.Fatalf("targets by name:\n%s", h.errOut.String())
	}
}

func TestLoadCampaignErrors(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	if _, err := h.app.LoadCampaign(Common{Campaign: "nope"}); err == nil || !strings.Contains(err.Error(), "gh fleet list") {
		t.Fatalf("err = %v", err)
	}
	st, _ := h.app.Store.Load("nofile")
	if err := h.app.Store.Save(st); err != nil {
		t.Fatal(err)
	}
	if _, err := h.app.LoadCampaign(Common{Campaign: "nofile"}); err == nil {
		t.Fatal("expected error for state without a file")
	}
	f := h.campaign(basic, files)
	if c, err := h.app.LoadCampaign(Common{Campaign: f}); err != nil || c.Name != "add-line" {
		t.Fatalf("by path: %v", err)
	}
}

func TestInit(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	f := filepath.Join(h.dir, "sub", "fleet.yaml")
	if err := h.app.Init(InitOptions{Name: "Pin Actions!", File: f, Org: "acme"}); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(f)
	if err != nil || !strings.Contains(string(b), "name: pin-actions") || !strings.Contains(string(b), "orgs: [acme]") {
		t.Fatalf("file = %s, %v", b, err)
	}
	if !strings.Contains(h.errOut.String(), "Wrote") {
		t.Fatalf("output: %s", h.errOut.String())
	}
	if err := h.app.Init(InitOptions{File: f}); err == nil || !strings.Contains(err.Error(), "--force") {
		t.Fatalf("err = %v", err)
	}
	if err := h.app.Init(InitOptions{File: f, Force: true}); err != nil {
		t.Fatal(err)
	}
	b, _ = os.ReadFile(f)
	if !strings.Contains(string(b), "name: sub") {
		t.Fatalf("name from directory not used:\n%s", b)
	}
	if c, err := h.app.LoadCampaign(Common{File: f}); err != nil || c.Name != "sub" {
		t.Fatalf("init wrote an invalid file: %v", err)
	}
}

func TestSlugName(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]string{"Pin Actions": "pin-actions", "--": "my-campaign", strings.Repeat("a", 70): strings.Repeat("a", 63), "a_b.c": "a-b-c"} {
		if got := SlugName(in); got != want {
			t.Errorf("SlugName(%q) = %q", in, got)
		}
	}
}

func TestHelpers(t *testing.T) {
	t.Parallel()
	if (&FailedError{N: 1, Hint: statusHint}).Error() != "1 repo failed; run gh fleet status for details" || (&FailedError{N: 2}).Error() != "2 repos failed" {
		t.Fatal("singular message")
	}
	if comma(1234567) != "1,234,567" || comma(12) != "12" {
		t.Fatal("comma")
	}
	if orDash("") != "-" || orDash("x") != "x" {
		t.Fatal("orDash")
	}
	if plural(1, "repo") != "1 repo" {
		t.Fatal("plural")
	}
}

// readFile returns a file's content as a string.
func readFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	return string(b), err
}
