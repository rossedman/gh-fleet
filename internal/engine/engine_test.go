package engine

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rossedman/gh-fleet/internal/campaign"
	"github.com/rossedman/gh-fleet/internal/ghapi"
	"github.com/rossedman/gh-fleet/internal/ghapi/fake"
	"github.com/rossedman/gh-fleet/internal/pr"
	"github.com/rossedman/gh-fleet/internal/state"
	"github.com/rossedman/gh-fleet/internal/target"
	"github.com/rossedman/gh-fleet/internal/testutil"
	"github.com/rossedman/gh-fleet/internal/timing"
	"github.com/rossedman/gh-fleet/internal/workspace"
)

// targets returns n targets named acme/r0..r(n-1).
func targets(n int) []target.Target {
	out := make([]target.Target, n)
	for i := range out {
		out[i] = target.Target{Repo: ghapi.Repo{Owner: "acme", Name: fmt.Sprintf("r%d", i), DefaultBranch: "main"}}
	}
	return out
}

func TestRunRespectsConcurrencyAndOrder(t *testing.T) {
	t.Parallel()
	var running, peak atomic.Int64
	proc := ProcessorFunc(func(_ context.Context, tg target.Target) (state.RepoState, error) {
		n := running.Add(1)
		for {
			p := peak.Load()
			if n <= p || peak.CompareAndSwap(p, n) {
				break
			}
		}
		time.Sleep(2 * time.Millisecond)
		running.Add(-1)
		status := state.Changed
		if strings.HasSuffix(tg.Repo.Name, "3") {
			status = state.Failed
		}
		return state.RepoState{Status: status}, nil
	})
	var seen atomic.Int64
	results, err := Run(context.Background(), targets(20), proc, Options{Concurrency: 4, OnResult: func(state.RepoState) { seen.Add(1) }})
	if err != nil {
		t.Fatal(err)
	}
	if peak.Load() > 4 || peak.Load() < 2 {
		t.Fatalf("peak concurrency = %d", peak.Load())
	}
	if seen.Load() != 20 || results[5].Repo != "acme/r5" || results[3].Status != state.Failed || results[4].Status != state.Changed {
		t.Fatalf("results = %+v", results[:6])
	}
}

func TestRunStopsOnFatalAndCancel(t *testing.T) {
	t.Parallel()
	boom := errors.New("quota")
	proc := ProcessorFunc(func(_ context.Context, tg target.Target) (state.RepoState, error) {
		if tg.Repo.Name == "r1" {
			return state.RepoState{Status: state.Failed}, boom
		}
		return state.RepoState{Status: state.Changed}, nil
	})
	results, err := Run(context.Background(), targets(10), proc, Options{Concurrency: 1})
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
	if results[0].Status != state.Changed || results[9].Status != state.Pending || results[9].Detail != "not started" {
		t.Fatalf("results = %+v", results)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	results, err = Run(ctx, targets(3), proc, Options{})
	if err != nil || results[0].Status != state.Pending {
		t.Fatalf("canceled run: %+v, %v", results, err)
	}
}

// benchmarkEngine runs 500 fake repo steps of ~200µs each.
func benchmarkEngine(b *testing.B, concurrency int) {
	ts := targets(500)
	proc := ProcessorFunc(func(context.Context, target.Target) (state.RepoState, error) {
		time.Sleep(200 * time.Microsecond)
		return state.RepoState{Status: state.Changed}, nil
	})
	b.ReportAllocs()
	for b.Loop() {
		if _, err := Run(context.Background(), ts, proc, Options{Concurrency: concurrency}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEngineConcurrency1(b *testing.B)  { benchmarkEngine(b, 1) }
func BenchmarkEngineConcurrency8(b *testing.B)  { benchmarkEngine(b, 8) }
func BenchmarkEngineConcurrency32(b *testing.B) { benchmarkEngine(b, 32) }

// fixture is a pipeline over one local bare repo.
type fixture struct {
	remote *testutil.Remote
	pipe   *Pipeline
	gh     *fake.Client
	target target.Target
	logs   string
}

// newFixture builds a pipeline in mode with the given inline script.
func newFixture(t *testing.T, mode Mode, run string) *fixture {
	t.Helper()
	env := testutil.GitEnv(t)
	remote := testutil.NewRemote(t, env, map[string]string{"README.md": "hello\n"})
	gh := fake.New()
	camp := &campaign.Campaign{Name: "fx", Change: campaign.Change{Steps: []campaign.Step{{Run: run}}}, PR: campaign.PR{Title: "t {{ .Repo.Name }}", Labels: []string{"fleet"}}}
	camp.ApplyDefaults()
	logs := t.TempDir()
	f := &fixture{
		remote: remote, gh: gh, logs: logs,
		target: target.Target{Repo: ghapi.Repo{Owner: "acme", Name: "api", DefaultBranch: "main"}},
		pipe: &Pipeline{
			Mode: mode, Campaign: camp, Git: testutil.Git(env), PRs: pr.NewManager(gh, 0),
			Workspaces: &workspace.Manager{Root: t.TempDir(), Git: testutil.Git(env), CloneURL: func(ghapi.Repo) string { return remote.URL() }},
			LogPath:    func(repo string) string { return filepath.Join(logs, strings.ReplaceAll(repo, "/", "_")+".log") },
		},
	}
	return f
}

func TestPipelineFailures(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	f := newFixture(t, Apply, "exit 0")
	f.target.Repo.DefaultBranch = ""
	if rs, _ := f.pipe.Process(ctx, f.target); rs.Status != state.Failed || !strings.Contains(rs.LastError, "no default branch") {
		t.Fatalf("rs = %+v", rs)
	}

	f = newFixture(t, Plan, "exit 0")
	f.pipe.Workspaces.CloneURL = func(ghapi.Repo) string { return "file:///nonexistent/repo.git" }
	if rs, _ := f.pipe.Process(ctx, f.target); rs.Status != state.Failed || !strings.Contains(rs.LastError, "git clone") {
		t.Fatalf("rs = %+v", rs)
	}

	f = newFixture(t, Plan, "echo out; echo err >&2; exit 2")
	rs, err := f.pipe.Process(ctx, f.target)
	if err != nil || rs.Status != state.Failed || !strings.Contains(rs.LastError, "status 2: err") {
		t.Fatalf("rs = %+v, %v", rs, err)
	}
	if b, _ := os.ReadFile(filepath.Join(f.logs, "acme_api.log")); !strings.Contains(string(b), "out") || !strings.Contains(string(b), "err") {
		t.Fatalf("log = %q", b)
	}

	f = newFixture(t, Apply, "echo x >> README.md")
	f.pipe.Campaign.PR.Title = "{{ .Nope }}"
	f.pipe.Campaign.PR.Commit.Message = "fixed message"
	if rs, _ := f.pipe.Process(ctx, f.target); rs.Status != state.Failed || !strings.Contains(rs.LastError, "pr.title") {
		t.Fatalf("rs = %+v", rs)
	}
	if f.pipe.PRs.OpenCount() != 0 {
		t.Fatal("slot not released")
	}

	f = newFixture(t, Apply, "echo x >> README.md")
	f.pipe.Campaign.PR.Commit.Message = "{{ .Nope }}"
	if rs, _ := f.pipe.Process(ctx, f.target); rs.Status != state.Failed || !strings.Contains(rs.LastError, "pr.commit") {
		t.Fatalf("rs = %+v", rs)
	}

	f = newFixture(t, Apply, "echo x >> README.md")
	f.pipe.Campaign.PR.Body = "{{ .Nope }}"
	if rs, _ := f.pipe.Process(ctx, f.target); rs.Status != state.Failed || !strings.Contains(rs.LastError, "pr.body") {
		t.Fatalf("rs = %+v", rs)
	}

	f = newFixture(t, Apply, "echo x >> README.md")
	f.gh.Errors["AddLabels"] = errors.New("label boom")
	rs, _ = f.pipe.Process(ctx, f.target)
	if rs.Status != state.Failed || rs.PRNumber != 1 || f.pipe.PRs.OpenCount() != 1 {
		t.Fatalf("created PR should be recorded even when labeling fails: %+v", rs)
	}

	// A fatal quota error is returned to stop the run.
	f = newFixture(t, Apply, "echo x >> README.md")
	f.gh.Errors["CreatePR"] = &ghapi.QuotaError{Resource: "core"}
	if _, err := f.pipe.Process(ctx, f.target); !errors.Is(err, ghapi.ErrQuotaLow) {
		t.Fatalf("err = %v", err)
	}
}

func TestPipelineDraftAndBase(t *testing.T) {
	t.Parallel()
	f := newFixture(t, Apply, "echo x >> README.md")
	f.remote.Commit("release", "release branch", map[string]string{"VERSION": "1\n"})
	f.pipe.Campaign.PR.Base = "release"
	f.pipe.Campaign.PR.Draft = true
	rs, err := f.pipe.Process(context.Background(), f.target)
	if err != nil || rs.Status != state.PROpen {
		t.Fatalf("rs = %+v, %v", rs, err)
	}
	p := f.gh.PRs[0]
	if p.BaseRef != "release" || !p.Draft || p.Title != "t api" {
		t.Fatalf("pr = %+v", p)
	}
	if f.remote.Show("fleet/fx", "VERSION") != "1" {
		t.Fatal("branch not based on pr.base")
	}
}

func TestPipelineExecAndPlanEnv(t *testing.T) {
	t.Parallel()
	f := newFixture(t, Exec, "")
	f.pipe.Argv = []string{"sh", "-c", `echo "$FLEET_REPO $FLEET_CAMPAIGN $FLEET_DEFAULT_BRANCH $FLEET_DRY_RUN"`}
	rs, err := f.pipe.Process(context.Background(), f.target)
	if err != nil || rs.Status != state.Done || strings.TrimSpace(rs.Output) != "acme/api fx main true" {
		t.Fatalf("rs = %+v, %v", rs, err)
	}
	f.pipe.Argv = []string{"false"}
	if rs, _ := f.pipe.Process(context.Background(), f.target); rs.Status != state.Failed {
		t.Fatalf("rs = %+v", rs)
	}

	f = newFixture(t, Plan, "echo x >> README.md")
	f.remote.Commit("fleet/fx", "old", map[string]string{"OLD": "1\n"})
	rs, _ = f.pipe.Process(context.Background(), f.target)
	if rs.Status != state.Changed || rs.Detail != "updates existing branch" || !strings.Contains(rs.Diff, "OLD") {
		t.Fatalf("rs = %+v", rs)
	}

	// Without a log path, output is discarded; with a bad one, too.
	f.pipe.LogPath = nil
	if _, err := f.pipe.Process(context.Background(), f.target); err != nil {
		t.Fatal(err)
	}
	f.pipe.LogPath = func(string) string { return "/dev/null/x/y.log" }
	if _, err := f.pipe.Process(context.Background(), f.target); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	f.pipe.LogPath = func(string) string { return dir }
	if _, err := f.pipe.Process(context.Background(), f.target); err != nil {
		t.Fatal(err)
	}
}

func TestPipelinePushRejected(t *testing.T) {
	t.Parallel()
	f := newFixture(t, Apply, "echo x >> README.md")
	// Someone else pushes the branch after the clone saw it missing: the
	// lease protects their work.
	f.pipe.Workspaces.CloneURL = func(ghapi.Repo) string {
		return f.remote.URL()
	}
	hook := filepath.Join(f.remote.Bare, "hooks", "pre-receive")
	if err := os.WriteFile(hook, []byte("#!/bin/sh\necho denied >&2\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	rs, _ := f.pipe.Process(context.Background(), f.target)
	if rs.Status != state.Failed || !strings.Contains(rs.LastError, "git push") || f.pipe.PRs.OpenCount() != 0 {
		t.Fatalf("rs = %+v", rs)
	}
}

func TestPipelineSteps(t *testing.T) {
	t.Parallel()
	f := newFixture(t, Apply, `echo "from=first" > "$FLEET_OUTPUT/vars"; echo "first message" > "$FLEET_OUTPUT/commit-message"; echo a >> README.md`)
	camp := f.pipe.Campaign
	camp.PR.Title = "t {{ .Vars.from }} {{ .Vars.seen }}"
	camp.Change.Steps = append(camp.Change.Steps,
		campaign.Step{Name: "second", Run: `echo "seen=$FLEET_VAR_FROM" > "$FLEET_OUTPUT/vars"; echo b >> README.md`},
		campaign.Step{Run: `echo "last message" > "$FLEET_OUTPUT/commit-message"`},
	)
	camp.ApplyDefaults()
	rs, err := f.pipe.Process(context.Background(), f.target)
	if err != nil || rs.Status != state.PROpen {
		t.Fatalf("rs = %+v, %v", rs, err)
	}
	if p := f.gh.PRs[0]; p.Title != "t first first" {
		t.Fatalf("title = %q", p.Title)
	}
	if got := f.remote.Show("fleet/fx", "README.md"); got != "hello\na\nb" {
		t.Fatalf("README = %q", got)
	}
	if msg := strings.Join(f.remote.Log("fleet/fx"), "\n"); !strings.Contains(msg, "last message") {
		t.Fatalf("commit message = %q", msg)
	}
	if b, _ := os.ReadFile(filepath.Join(f.logs, "acme_api.log")); !strings.Contains(string(b), "## step second") || !strings.Contains(string(b), "## step #3") {
		t.Fatalf("log does not name steps:\n%s", b)
	}

	f = newFixture(t, Plan, "echo a >> README.md")
	f.pipe.Campaign.Change.Steps = append(f.pipe.Campaign.Change.Steps, campaign.Step{Name: "boom", Run: "exit 3"})
	f.pipe.Campaign.ApplyDefaults()
	if rs, _ := f.pipe.Process(context.Background(), f.target); rs.Status != state.Failed || !strings.HasPrefix(rs.LastError, "step boom: ") {
		t.Fatalf("rs = %+v", rs)
	}
	f = newFixture(t, Plan, "")
	f.pipe.Campaign.Change.Steps = []campaign.Step{{Starlark: "x.star"}}
	if rs, _ := f.pipe.Process(context.Background(), f.target); rs.Status != state.Failed || !strings.Contains(rs.LastError, "not compiled") {
		t.Fatalf("rs = %+v", rs)
	}
}

func TestRepoFingerprint(t *testing.T) {
	t.Parallel()
	tg := target.Target{Repo: ghapi.Repo{Owner: "acme", Name: "api", Topics: []string{"a"}}, Vars: map[string]any{"node": 20}}
	fp := RepoFingerprint("camp", tg)
	if len(fp) != 64 || RepoFingerprint("camp", tg) != fp {
		t.Fatalf("fingerprint %q", fp)
	}
	if RepoFingerprint("", tg) != "" {
		t.Error("no campaign fingerprint, but a repo one")
	}
	other := tg
	other.Vars = map[string]any{"node": 22}
	topics := tg
	topics.Repo.Topics = []string{"b"}
	for _, o := range []target.Target{other, topics} {
		if RepoFingerprint("camp", o) == fp {
			t.Errorf("%+v kept the fingerprint", o)
		}
	}
	if RepoFingerprint("camp2", tg) == fp {
		t.Error("campaign change kept the fingerprint")
	}
}

func TestPipelineUnchanged(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	for _, mode := range []Mode{Plan, Apply} {
		f := newFixture(t, mode, "true")
		f.pipe.Fingerprint = "camp"
		rs, err := f.pipe.Process(ctx, f.target)
		if err != nil || rs.Status != state.Noop {
			t.Fatalf("mode %d: %+v, %v", mode, rs, err)
		}
		// The record says which base commit the change found nothing on.
		if rs.BaseSHA != f.remote.Branch("main") || rs.Fingerprint != RepoFingerprint("camp", f.target) {
			t.Errorf("mode %d: base %q fingerprint %q", mode, rs.BaseSHA, rs.Fingerprint)
		}
		// Without a campaign fingerprint, there's nothing to record.
		f.pipe.Fingerprint = ""
		if rs, _ := f.pipe.Process(ctx, f.target); rs.Status != state.Noop || rs.BaseSHA != "" || rs.Fingerprint != "" {
			t.Errorf("mode %d without fingerprint: %+v", mode, rs)
		}
	}
}

func TestPipelineExistingPRs(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	withPR := func(run, state string) *fixture {
		f := newFixture(t, Apply, run)
		f.gh.PRs = append(f.gh.PRs, &ghapi.PR{Repo: "acme/api", Number: 7, State: state, HeadRef: "fleet/fx"})
		if err := f.pipe.PRs.Index(ctx, []string{"acme/api"}, "fleet/fx", "", nil); err != nil {
			t.Fatal(err)
		}
		return f
	}
	api := target.Target{Repo: ghapi.Repo{Owner: "acme", Name: "api", DefaultBranch: "main"}}
	if rs, _ := withPR("true", ghapi.StateMerged).pipe.Process(ctx, api); rs.Status != state.Merged || rs.PRNumber != 7 {
		t.Errorf("merged: %+v", rs)
	}
	if rs, _ := withPR("true", ghapi.StateClosed).pipe.Process(ctx, api); rs.Status != state.Closed || !strings.Contains(rs.Detail, "--reopen") {
		t.Errorf("closed: %+v", rs)
	}
	// An open PR whose change already landed is left, or closed on request.
	f := withPR("true", ghapi.StateOpen)
	if rs, _ := f.pipe.Process(ctx, f.target); rs.Status != state.Noop || !strings.Contains(rs.Detail, "--close-noop") || rs.Fingerprint != "" {
		t.Errorf("open noop: %+v", rs)
	}
	f.pipe.CloseNoop = true
	if rs, _ := f.pipe.Process(ctx, f.target); rs.Status != state.Closed || f.gh.PRs[0].State != ghapi.StateClosed {
		t.Errorf("close noop: %+v", rs)
	}
	f = withPR("true", ghapi.StateOpen)
	f.pipe.CloseNoop = true
	f.gh.Errors["Comment"] = errors.New("boom")
	if rs, _ := f.pipe.Process(ctx, f.target); rs.Status != state.Failed {
		t.Errorf("close noop error: %+v", rs)
	}
	// With every PR slot taken, a change isn't pushed.
	f = newFixture(t, Apply, "echo x >> README.md")
	f.pipe.PRs = pr.NewManager(f.gh, 1)
	f.pipe.PRs.Reserve()
	if rs, _ := f.pipe.Process(ctx, f.target); rs.Status != state.Changed || !strings.Contains(rs.Detail, "maxOpenPRs reached (1)") {
		t.Errorf("max open: %+v", rs)
	}
}

func TestPipelineReviewersFrom(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	setup := func() *fixture {
		f := newFixture(t, Apply, "echo x >> README.md")
		f.remote.Commit("main", "owners", map[string]string{
			".github/CODEOWNERS": "* @alice @acme/team\n",
			"catalog-info.yaml":  "apiVersion: backstage.io/v1alpha1\nkind: Component\nmetadata: {name: api}\nspec: {owner: group:payments}\n",
		})
		f.pipe.Campaign.PR.Reviewers = []string{"alice"}
		f.pipe.Campaign.PR.ReviewersFrom = []string{"codeowners", "catalog"}
		return f
	}
	f := setup()
	rs, err := f.pipe.Process(ctx, f.target)
	if err != nil || rs.Status != state.PROpen {
		t.Fatalf("rs = %+v, %v", rs, err)
	}
	// alice was already asked; the owners are asked once, together.
	got := f.gh.Reviewers["acme/api#1"]
	if strings.Join(got, ",") != "alice,acme/team,acme/payments" {
		t.Fatalf("reviewers = %v", got)
	}
	if b, _ := os.ReadFile(filepath.Join(f.logs, "acme_api.log")); !strings.Contains(string(b), "## reviewers from codeowners, catalog: acme/team, acme/payments") {
		t.Fatalf("log:\n%s", b)
	}

	// Owners GitHub refuses are skipped and logged, not a failure.
	f = setup()
	f.gh.Errors["RequestReviewers"] = errors.New("not a collaborator")
	f.pipe.Campaign.PR.Reviewers = nil
	rs, _ = f.pipe.Process(ctx, f.target)
	b, _ := os.ReadFile(filepath.Join(f.logs, "acme_api.log"))
	if rs.Status != state.PROpen || !strings.Contains(string(b), "reviewer acme/team skipped: not a collaborator") {
		t.Fatalf("rs = %+v, log:\n%s", rs, b)
	}
}

// gitCalls returns how many times each git subcommand ran.
func gitCalls(rec *timing.Recorder) map[string]int {
	got := map[string]int{}
	for _, s := range rec.Stats() {
		got[s.Name] = s.Count
	}
	return got
}

func TestPipelineUsesBranchHeads(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	f := newFixture(t, Apply, "echo x >> README.md")
	rec := timing.New(nil)
	f.pipe.Git.Timings, f.pipe.Workspaces.Git.Timings = rec, rec

	// A branch the lookup didn't find is created, without asking origin.
	f.pipe.BranchHeads = map[string]string{}
	rs, err := f.pipe.Process(ctx, f.target)
	if err != nil || rs.Status != state.PROpen {
		t.Fatalf("new branch: %+v, %v", rs, err)
	}
	head := f.remote.Branch("fleet/fx")
	if head == "" || rs.HeadSHA != head {
		t.Fatalf("pushed %q, branch at %q", rs.HeadSHA, head)
	}

	// A branch the lookup found is updated on top of where it is.
	f.remote.Commit("main", "base moved", map[string]string{"BASE": "1\n"})
	f.pipe.BranchHeads = map[string]string{"acme/api": head}
	f.pipe.Campaign.Change.Steps[0].Run = "echo y >> README.md"
	rs, err = f.pipe.Process(ctx, f.target)
	if err != nil || rs.Status != state.PROpen || f.remote.Show("fleet/fx", "BASE") != "1" || !strings.Contains(f.remote.Show("fleet/fx", "README.md"), "x\ny") {
		t.Fatalf("existing branch: %+v, %v; README %q", rs, err, f.remote.Show("fleet/fx", "README.md"))
	}
	if n := gitCalls(rec)["git ls-remote"]; n != 0 {
		t.Fatalf("asked origin %d times with BranchHeads set", n)
	}

	// A stale lookup can't overwrite the branch: the push's lease fails.
	moved := f.remote.Branch("fleet/fx")
	f.pipe.BranchHeads = map[string]string{}
	rs, err = f.pipe.Process(ctx, f.target)
	if err != nil || rs.Status != state.Failed || !strings.Contains(rs.LastError, "git push") || f.remote.Branch("fleet/fx") != moved {
		t.Fatalf("stale lookup: %+v, %v", rs, err)
	}

	// Plan with BranchHeads still shows what the PR would contain.
	f = newFixture(t, Plan, "echo x >> README.md")
	rec = timing.New(nil)
	f.pipe.Git.Timings, f.pipe.Workspaces.Git.Timings = rec, rec
	f.remote.Commit("fleet/fx", "old", map[string]string{"OLD": "1\n"})
	f.pipe.BranchHeads = map[string]string{"acme/api": f.remote.Branch("fleet/fx")}
	rs, _ = f.pipe.Process(ctx, f.target)
	if rs.Status != state.Changed || rs.Detail != "updates existing branch" || !strings.Contains(rs.Diff, "OLD") || gitCalls(rec)["git ls-remote"] != 0 {
		t.Fatalf("plan: %+v, calls %v", rs, gitCalls(rec))
	}
}
