package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/rossedman/gh-fleet/internal/campaign"
	"github.com/rossedman/gh-fleet/internal/ghapi"
	"github.com/rossedman/gh-fleet/internal/gitops"
	"github.com/rossedman/gh-fleet/internal/owners"
	"github.com/rossedman/gh-fleet/internal/pr"
	"github.com/rossedman/gh-fleet/internal/runner"
	"github.com/rossedman/gh-fleet/internal/star"
	"github.com/rossedman/gh-fleet/internal/state"
	"github.com/rossedman/gh-fleet/internal/target"
	"github.com/rossedman/gh-fleet/internal/timing"
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
	// ExecTimeout limits Argv in each repo; zero means none.
	ExecTimeout time.Duration
	// Scripts holds the compiled script of each starlark step, by index
	// into change.steps.
	Scripts map[int]*star.Script
	// Fingerprint is the campaign's fingerprint. When set, a repo found
	// unchanged reports its base commit and RepoFingerprint, so a later
	// apply can skip it.
	Fingerprint string
	// Timings, when set, records how long each repo spends in each stage.
	Timings *timing.Recorder
	// BranchHeads holds the campaign branch's commit per repo full name,
	// looked up in batches; a repo without the branch is absent. When nil,
	// each repo asks its git server instead.
	BranchHeads map[string]string
}

// RepoFingerprint is the fingerprint of the change in one repo: the
// campaign's, plus everything about the repo the change can read besides
// its files. It is "" when the campaign has no fingerprint.
func RepoFingerprint(campaign string, t target.Target) string {
	if campaign == "" {
		return ""
	}
	b, _ := json.Marshal(struct {
		Campaign string
		Repo     ghapi.Repo
		Vars     map[string]any
	}{campaign, t.Repo, t.Vars})
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// unchanged marks rs as a repo the change left alone: its base commit and
// fingerprint, for a later apply to skip it.
func (p *Pipeline) unchanged(ctx context.Context, ws *workspace.Workspace, base string, t target.Target, rs *state.RepoState) {
	rs.Status = state.Noop
	fp := RepoFingerprint(p.Fingerprint, t)
	if fp == "" {
		return
	}
	sha, err := p.Git.Rev(ctx, ws.Dir, "origin/"+base)
	if err != nil {
		return
	}
	rs.BaseSHA, rs.Fingerprint = sha, fp
}

// NoopComment is posted on a PR closed because its change already landed.
const NoopComment = "The change this pull request makes is already on the base branch, so gh fleet is closing it."

// Process runs one repo through the pipeline. In-flight work is not
// interrupted by cancellation; the engine simply stops starting new repos.
func (p *Pipeline) Process(ctx context.Context, t target.Target) (state.RepoState, error) {
	ctx = context.WithoutCancel(ctx)
	repo := t.Repo
	rs := state.RepoState{Repo: repo.FullName()}
	defer p.Timings.Start("repo: total", "repo", repo.FullName())()
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
	stop := p.Timings.Start("repo: clone", "repo", repo.FullName())
	branch := p.Campaign.PR.Branch
	switch {
	case p.Mode == Exec:
		ws, err = p.Workspaces.Clone(ctx, p.Campaign.Name, repo, base)
	case p.BranchHeads != nil:
		ws, err = p.Workspaces.PrepareAt(ctx, p.Campaign.Name, repo, base, branch, p.BranchHeads[repo.FullName()])
	default:
		ws, err = p.Workspaces.Prepare(ctx, p.Campaign.Name, repo, base, branch)
	}
	stop()
	if err != nil {
		return fail(err)
	}
	defer func() {
		defer p.Timings.Start("repo: cleanup", "repo", repo.FullName())()
		ws.Cleanup()
	}()

	logw, closeLog := p.openLog(repo.FullName())
	defer closeLog()

	if p.Mode == Exec {
		spec := runner.Spec{Argv: p.Argv, Env: p.Campaign.Change.Env, Timeout: p.ExecTimeout}
		res, err := runner.Run(ctx, ws.Dir, spec, p.env(t), logw)
		rs.Output = string(res.Stdout)
		if err != nil {
			return fail(err)
		}
		rs.Status = state.Done
		return rs, nil
	}
	res, extra, err := p.runChange(ctx, ws.Dir, t, logw)
	if err != nil {
		return fail(err)
	}

	if p.Mode == Plan {
		stop := p.Timings.Start("repo: diff", "repo", repo.FullName())
		diff, err := p.Git.StagedDiff(ctx, ws.Dir, "origin/"+base)
		stop()
		if err != nil {
			return fail(err)
		}
		if diff == "" {
			p.unchanged(ctx, ws, base, t, &rs)
			return rs, nil
		}
		rs.Status, rs.Diff = state.Changed, diff
		if ws.RemoteSHA != "" {
			rs.Detail = "updates existing branch"
		}
		// Render the PR now, so a missing variable shows up in plan.
		if _, err := p.content(t, res, extra); err != nil {
			return fail(err)
		}
		return rs, nil
	}

	return p.apply(ctx, t, ws, res, extra, existing, rs, base, logw)
}

// apply commits, pushes and syncs the PR for a repo whose script ran.
func (p *Pipeline) apply(ctx context.Context, t target.Target, ws *workspace.Workspace, res runner.Result, extra map[string]any, existing *ghapi.PR, rs state.RepoState, base string, logw io.Writer) (state.RepoState, error) {
	repo := t.Repo
	fail := func(err error) (state.RepoState, error) {
		rs.Status = state.Failed
		rs.LastError = err.Error()
		if errors.Is(err, ghapi.ErrQuotaLow) {
			return rs, err
		}
		return rs, nil
	}

	// Render everything before committing, so a template error never
	// leaves a pushed branch behind.
	c, err := p.content(t, res, extra)
	if err != nil {
		return fail(err)
	}
	dirty, err := p.Git.Dirty(ctx, ws.Dir)
	if err != nil {
		return fail(err)
	}
	if dirty {
		if err := p.Git.CommitAll(ctx, ws.Dir, c.commit, p.Campaign.PR.Commit.Signoff); err != nil {
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
		p.unchanged(ctx, ws, base, t, &rs)
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
		stop := p.Timings.Start("repo: push", "repo", repo.FullName())
		err := p.Git.Push(ctx, ws.Dir, p.Campaign.PR.Branch, ws.RemoteSHA)
		stop()
		if err != nil {
			release()
			return fail(err)
		}
	}
	rs.HeadSHA = head

	d := pr.Desired{
		Repo: repo.FullName(), Title: c.title, Body: c.body, Head: p.Campaign.PR.Branch, Base: base,
		Draft: p.Campaign.PR.Draft, Labels: c.labels, Reviewers: c.reviewers,
	}
	stop := p.Timings.Start("repo: sync PR", "repo", repo.FullName())
	got, action, err := p.PRs.Sync(ctx, d, existing, p.Reopen)
	stop()
	if got.Number != 0 {
		rs.PRNumber, rs.PRURL = got.Number, got.URL
	}
	if err != nil {
		if got.Number == 0 {
			release()
		}
		return fail(err)
	}
	if action == pr.Created && len(p.Campaign.PR.ReviewersFrom) > 0 {
		p.requestOwners(ctx, ws.Dir, repo, base, got.Number, c.reviewers, logw)
	}
	rs.Status = state.PROpen
	rs.Checks, rs.Review = got.Checks, got.Review
	rs.Detail = string(action)
	if ws.Reset {
		rs.Detail += "; branch reset after rebase conflict"
	}
	return rs, nil
}

// runChange runs change.steps in order in the clone. Variables a step
// sets are visible to the steps after it, and the last commit message a
// step writes wins. It returns that commit message and every variable the
// steps set.
func (p *Pipeline) runChange(ctx context.Context, dir string, t target.Target, log io.Writer) (runner.Result, map[string]any, error) {
	extra := map[string]any{}
	var out runner.Result
	cur := t
	for i, st := range p.Campaign.Change.Steps {
		cur.Vars = maps.Clone(t.Vars)
		if cur.Vars == nil {
			cur.Vars = map[string]any{}
		}
		maps.Copy(cur.Vars, extra)
		fmt.Fprintf(log, "## step %s\n", st.Label(i))
		stop := p.Timings.Start("repo: step "+st.Label(i), "repo", t.Repo.FullName())
		vars, msg, err := p.runStep(ctx, dir, i, st, cur, log)
		stop()
		if err != nil {
			if len(p.Campaign.Change.Steps) > 1 || st.Name != "" {
				err = fmt.Errorf("step %s: %w", st.Label(i), err)
			}
			return runner.Result{}, nil, err
		}
		maps.Copy(extra, vars)
		if msg != "" {
			out.CommitMessage = msg
		}
	}
	return out, extra, nil
}

// runStep runs one step and returns the variables and commit message it
// set.
func (p *Pipeline) runStep(ctx context.Context, dir string, i int, st campaign.Step, t target.Target, log io.Writer) (map[string]any, string, error) {
	if st.Starlark != "" {
		script := p.Scripts[i]
		if script == nil {
			return nil, "", fmt.Errorf("starlark %s was not compiled", st.Starlark)
		}
		repo := t.Repo
		info := star.RepoInfo{
			Owner: repo.Owner, Name: repo.Name, DefaultBranch: repo.DefaultBranch,
			Topics: repo.Topics, Language: repo.Language, Properties: repo.Properties, Vars: t.Vars,
		}
		r, err := script.Run(ctx, dir, info, log, st.Timeout.Std())
		if err != nil {
			return nil, "", err
		}
		return r.Vars, "", nil
	}
	spec := runner.Spec{Shell: st.Shell, Env: p.Campaign.StepEnv(st), Timeout: st.Timeout.Std()}
	if f := p.Campaign.ScriptFile(st); f != "" {
		spec.File = f
	} else {
		spec.Script = st.Run
	}
	res, err := runner.Run(ctx, dir, spec, p.env(t), log)
	if err != nil {
		return nil, "", err
	}
	vars := make(map[string]any, len(res.Vars))
	for k, v := range res.Vars {
		vars[k] = v
	}
	return vars, res.CommitMessage, nil
}

// env returns the FLEET_* variables for a repo: the fixed ones, then
// FLEET_PROP_<NAME> per custom property and FLEET_VAR_<NAME> per variable,
// in name order.
func (p *Pipeline) env(t target.Target) []string {
	repo := t.Repo
	env := []string{
		"FLEET_REPO=" + repo.FullName(),
		"FLEET_CAMPAIGN=" + p.Campaign.Name,
		"FLEET_DEFAULT_BRANCH=" + repo.DefaultBranch,
		"FLEET_DRY_RUN=" + strconv.FormatBool(p.Mode != Apply),
	}
	for _, k := range slices.Sorted(maps.Keys(repo.Properties)) {
		env = append(env, "FLEET_PROP_"+campaign.EnvName(k)+"="+repo.Properties[k])
	}
	for _, k := range slices.Sorted(maps.Keys(t.Vars)) {
		env = append(env, "FLEET_VAR_"+campaign.EnvName(k)+"="+campaign.FormatValue(t.Vars[k]))
	}
	return env
}

// prContent is everything rendered from the campaign's PR templates for
// one repo.
type prContent struct {
	title, body, commit string
	labels, reviewers   []string
}

// content renders the PR title, body, commit message, labels and
// reviewers for a repo. extra (values the change set with repo.set_var or
// wrote to $FLEET_OUTPUT/vars) overrides the campaign's variables, and a
// commit message the script wrote overrides the template.
func (p *Pipeline) content(t target.Target, res runner.Result, extra map[string]any) (prContent, error) {
	repo := t.Repo
	vars := maps.Clone(t.Vars)
	if vars == nil {
		vars = map[string]any{}
	}
	maps.Copy(vars, extra)
	data := campaign.RepoData{
		Owner: repo.Owner, Name: repo.Name, FullName: repo.FullName(), DefaultBranch: repo.DefaultBranch,
		Topics: repo.Topics, Language: repo.Language, Properties: repo.Properties, Vars: vars,
	}
	var c prContent
	var err error
	if c.title, err = p.Campaign.RenderTitle(data); err != nil {
		return c, err
	}
	if c.body, err = p.Campaign.RenderBody(data); err != nil {
		return c, err
	}
	c.commit = res.CommitMessage
	if c.commit == "" {
		if c.commit, err = p.Campaign.RenderCommitMessage(data); err != nil {
			return c, err
		}
	}
	if c.labels, err = p.Campaign.RenderList("labels", p.Campaign.PR.Labels, data); err != nil {
		return c, err
	}
	if c.reviewers, err = p.Campaign.RenderList("reviewers", p.Campaign.PR.Reviewers, data); err != nil {
		return c, err
	}
	return c, nil
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

// Local is the outcome of running a change in a local directory.
type Local struct {
	// Vars are the variables the steps set.
	Vars map[string]any
	// Title, Body and CommitMessage are set by RenderLocal.
	Title, Body, CommitMessage string

	res runner.Result
}

// RunLocal runs every step of the change in dir as if dir were t's clone.
// Nothing is staged, committed or pushed.
func (p *Pipeline) RunLocal(ctx context.Context, dir string, t target.Target, log io.Writer) (Local, error) {
	res, extra, err := p.runChange(ctx, dir, t, log)
	if err != nil {
		return Local{}, err
	}
	return Local{Vars: extra, res: res}, nil
}

// RenderLocal renders the PR a local run would open, as apply does for a
// repo the change edited.
func (p *Pipeline) RenderLocal(t target.Target, l Local) (Local, error) {
	c, err := p.content(t, l.res, l.Vars)
	if err != nil {
		return l, fmt.Errorf("pr: %w", err)
	}
	l.Title, l.Body, l.CommitMessage = c.title, c.body, c.commit
	return l, nil
}

// requestOwners asks the repo's own owners, from CODEOWNERS or the
// Backstage catalog, to review a new PR. Owners GitHub won't accept, such
// as the author or a team without access, are skipped and logged rather
// than failing the repo.
func (p *Pipeline) requestOwners(ctx context.Context, dir string, repo ghapi.Repo, base string, number int, already []string, log io.Writer) {
	var want []string
	for _, from := range p.Campaign.PR.ReviewersFrom {
		switch from {
		case "codeowners":
			for _, f := range owners.CodeownersPaths {
				b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(f)))
				if err != nil {
					continue
				}
				changed, err := p.Git.Run(ctx, dir, "diff", "--name-only", "origin/"+base, "HEAD")
				if err == nil {
					want = append(want, owners.ParseCodeowners(string(b)).Owners(strings.Fields(changed))...)
				}
				break
			}
		case "catalog":
			for _, f := range owners.CatalogPaths {
				if b, err := os.ReadFile(filepath.Join(dir, f)); err == nil {
					want = append(want, owners.CatalogOwners(b, repo.Owner)...)
					break
				}
			}
		}
	}
	seen := map[string]bool{}
	for _, r := range already {
		seen[strings.ToLower(r)] = true
	}
	var ask []string
	for _, w := range want {
		if !seen[strings.ToLower(w)] {
			seen[strings.ToLower(w)] = true
			ask = append(ask, w)
		}
	}
	if len(ask) == 0 {
		return
	}
	fmt.Fprintf(log, "## reviewers from %s: %s\n", strings.Join(p.Campaign.PR.ReviewersFrom, ", "), strings.Join(ask, ", "))
	if err := p.PRs.Client.RequestReviewers(ctx, repo.FullName(), number, ask); err == nil {
		return
	}
	// One bad owner fails the whole request; ask for each alone.
	for _, r := range ask {
		if err := p.PRs.Client.RequestReviewers(ctx, repo.FullName(), number, []string{r}); err != nil {
			fmt.Fprintf(log, "reviewer %s skipped: %v\n", r, err)
		}
	}
}
