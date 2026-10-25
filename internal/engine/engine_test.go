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
	camp := &campaign.Campaign{Name: "fx", Change: campaign.Change{Run: run}, PR: campaign.PR{Title: "t {{ .Repo.Name }}", Labels: []string{"fleet"}}}
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
