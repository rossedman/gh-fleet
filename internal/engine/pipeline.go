package engine

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/rossedman/gh-fleet/internal/campaign"
	"github.com/rossedman/gh-fleet/internal/ghapi"
	"github.com/rossedman/gh-fleet/internal/gitops"
	"github.com/rossedman/gh-fleet/internal/pr"
	"github.com/rossedman/gh-fleet/internal/runner"
	"github.com/rossedman/gh-fleet/internal/state"
	"github.com/rossedman/gh-fleet/internal/target"
	"github.com/rossedman/gh-fleet/internal/workspace"
)

// Mode selects what the pipeline does with a change.
type Mode int

// Pipeline modes.
const (
	// Plan runs the change and reports the diff without pushing.
	Plan Mode = iota
	// Apply runs the change, pushes the branch and opens or updates PRs.
	Apply
	// Exec runs a read-only command and reports its output.
	Exec
)

// Pipeline is the per-repo state machine: check PR, clone, run, detect
// changes, then commit, push and sync the PR. Any step can end the repo
// as noop or failed without affecting other repos.
type Pipeline struct {
	Mode       Mode
	Campaign   *campaign.Campaign
	Workspaces *workspace.Manager
	Git        gitops.Git
	// PRs is required in Apply mode.
	PRs *pr.Manager
	// LogPath returns the per-repo log file; nil disables logs.
	LogPath func(repo string) string
	// Reopen reopens PRs that were closed without merging.
	Reopen bool
	// CloseNoop closes open PRs whose change is already on the base.
	CloseNoop bool
	// Argv is the command run in Exec mode.
	Argv []string
}

// NoopComment is posted on a PR closed because its change already landed.
const NoopComment = "The change this pull request makes is already on the base branch, so gh fleet is closing it."

// Process runs one repo through the pipeline. In-flight work is not
// interrupted by cancellation; the engine simply stops starting new repos.
func (p *Pipeline) Process(ctx context.Context, t target.Target) (state.RepoState, error) {
	ctx = context.WithoutCancel(ctx)
	repo := t.Repo
	rs := state.RepoState{Repo: repo.FullName()}
	fail := func(err error) (state.RepoState, error) {
		rs.Status = state.Failed
		rs.LastError = err.Error()
		if errors.Is(err, ghapi.ErrQuotaLow) {
			return rs, err
		}
		return rs, nil
	}

	var existing *ghapi.PR
	if p.Mode == Apply {
		existing = p.PRs.Existing(repo.FullName())
		if existing != nil {
			rs.PRNumber, rs.PRURL = existing.Number, existing.URL
			switch {
			case existing.State == ghapi.StateMerged:
				rs.Status, rs.Detail = state.Merged, "PR already merged"
				return rs, nil
			case existing.State == ghapi.StateClosed && !p.Reopen:
				rs.Status, rs.Detail = state.Closed, "PR closed without merging; use --reopen to reopen it"
				return rs, nil
			}
		}
	}

	base := p.Campaign.PR.Base
	if base == "" {
		base = repo.DefaultBranch
	}
	if base == "" {
		return fail(errors.New("repository has no default branch (is it empty?)"))
	}

	var ws *workspace.Workspace
	var err error
	if p.Mode == Exec {
		ws, err = p.Workspaces.Clone(ctx, p.Campaign.Name, repo, base)
	} else {
		ws, err = p.Workspaces.Prepare(ctx, p.Campaign.Name, repo, base, p.Campaign.PR.Branch)
	}
	if err != nil {
		return fail(err)
	}
	defer ws.Cleanup()

	logw, closeLog := p.openLog(repo.FullName())
	defer closeLog()

	res, err := runner.Run(ctx, ws.Dir, p.spec(), p.env(repo), logw)
	if p.Mode == Exec {
		rs.Output = string(res.Stdout)
		if err != nil {
			return fail(err)
		}
		rs.Status = state.Done
		return rs, nil
	}
	if err != nil {
		return fail(err)
	}

	if p.Mode == Plan {
		diff, err := p.Git.StagedDiff(ctx, ws.Dir, "origin/"+base)
		if err != nil {
			return fail(err)
		}
		if diff == "" {
			rs.Status = state.Noop
			return rs, nil
		}
		rs.Status, rs.Diff = state.Changed, diff
		if ws.RemoteSHA != "" {
			rs.Detail = "updates existing branch"
		}
		return rs, nil
	}

	return p.apply(ctx, t, ws, res, existing, rs, base)
}

// apply commits, pushes and syncs the PR for a repo whose script ran.
func (p *Pipeline) apply(ctx context.Context, t target.Target, ws *workspace.Workspace, res runner.Result, existing *ghapi.PR, rs state.RepoState, base string) (state.RepoState, error) {
	repo := t.Repo
	data := campaign.RepoData{Owner: repo.Owner, Name: repo.Name, FullName: repo.FullName(), DefaultBranch: repo.DefaultBranch}
	fail := func(err error) (state.RepoState, error) {
		rs.Status = state.Failed
		rs.LastError = err.Error()
		if errors.Is(err, ghapi.ErrQuotaLow) {
			return rs, err
		}
		return rs, nil
	}

	dirty, err := p.Git.Dirty(ctx, ws.Dir)
	if err != nil {
		return fail(err)
	}
	if dirty {
		msg := res.CommitMessage
		if msg == "" {
			if msg, err = p.Campaign.RenderCommitMessage(data); err != nil {
				return fail(err)
			}
		}
		if err := p.Git.CommitAll(ctx, ws.Dir, msg, p.Campaign.PR.Commit.Signoff); err != nil {
			return fail(err)
		}
	}
	same, err := p.Git.SameTree(ctx, ws.Dir, "HEAD", "origin/"+base)
	if err != nil {
		return fail(err)
	}
	if same {
		if existing != nil && existing.State == ghapi.StateOpen {
			if p.CloseNoop {
				if err := p.PRs.Close(ctx, *existing, NoopComment, true); err != nil {
					return fail(err)
				}
				rs.Status, rs.Detail = state.Closed, "closed: change already on "+base
				return rs, nil
			}
			rs.Status, rs.Detail = state.Noop, "PR open but change already on "+base+"; use --close-noop to close it"
			return rs, nil
		}
		rs.Status = state.Noop
		return rs, nil
	}

	needsSlot := existing == nil || existing.State != ghapi.StateOpen
	if needsSlot && !p.PRs.Reserve() {
		rs.Status = state.Changed
		rs.Detail = "maxOpenPRs reached (" + strconv.Itoa(p.PRs.MaxOpen) + "); not pushed"
		return rs, nil
	}
	release := func() {
		if needsSlot {
			p.PRs.Release()
		}
	}

	head, err := p.Git.Rev(ctx, ws.Dir, "HEAD")
	if err != nil {
		release()
		return fail(err)
	}
	if head != ws.RemoteSHA {
		if err := p.Git.Push(ctx, ws.Dir, p.Campaign.PR.Branch, ws.RemoteSHA); err != nil {
			release()
			return fail(err)
		}
	}
	rs.HeadSHA = head

	title, err := p.Campaign.RenderTitle(data)
	if err != nil {
		release()
		return fail(err)
	}
	body, err := p.Campaign.RenderBody(data)
	if err != nil {
		release()
		return fail(err)
	}
	d := pr.Desired{
		Repo: repo.FullName(), Title: title, Body: body, Head: p.Campaign.PR.Branch, Base: base,
		Draft: p.Campaign.PR.Draft, Labels: p.Campaign.PR.Labels, Reviewers: p.Campaign.PR.Reviewers,
	}
	got, action, err := p.PRs.Sync(ctx, d, existing, p.Reopen)
	if got.Number != 0 {
		rs.PRNumber, rs.PRURL = got.Number, got.URL
	}
	if err != nil {
		if got.Number == 0 {
			release()
		}
		return fail(err)
	}
	rs.Status = state.PROpen
	rs.Checks, rs.Review = got.Checks, got.Review
	rs.Detail = string(action)
	if ws.Reset {
		rs.Detail += "; branch reset after rebase conflict"
	}
	return rs, nil
}

// spec builds the runner spec for the current mode.
func (p *Pipeline) spec() runner.Spec {
	c := p.Campaign.Change
	s := runner.Spec{Shell: c.Shell, Env: c.Env, Timeout: c.Timeout.Std()}
	switch {
	case p.Mode == Exec:
		s.Argv = p.Argv
	case p.Campaign.ScriptFile() != "":
		s.File = p.Campaign.ScriptFile()
	default:
		s.Script = c.Run
	}
	return s
}

// env returns the FLEET_* variables for a repo.
func (p *Pipeline) env(repo ghapi.Repo) []string {
	return []string{
		"FLEET_REPO=" + repo.FullName(),
		"FLEET_CAMPAIGN=" + p.Campaign.Name,
		"FLEET_DEFAULT_BRANCH=" + repo.DefaultBranch,
		"FLEET_DRY_RUN=" + strconv.FormatBool(p.Mode != Apply),
	}
}

// openLog opens the per-repo log, truncating any previous run's log.
func (p *Pipeline) openLog(repo string) (io.Writer, func()) {
	if p.LogPath == nil {
		return io.Discard, func() {}
	}
	path := p.LogPath(repo)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return io.Discard, func() {}
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return io.Discard, func() {}
	}
	fmt.Fprintf(f, "# gh fleet %s\n", repo)
	return f, func() { _ = f.Close() }
}
