package app

import (
	"context"
	"fmt"

	"github.com/rossedman/gh-fleet/internal/campaign"
	"github.com/rossedman/gh-fleet/internal/engine"
	"github.com/rossedman/gh-fleet/internal/ghapi"
	"github.com/rossedman/gh-fleet/internal/pr"
	"github.com/rossedman/gh-fleet/internal/render"
	"github.com/rossedman/gh-fleet/internal/state"
	"github.com/rossedman/gh-fleet/internal/target"
	"github.com/rossedman/gh-fleet/internal/workspace"
)

// ApplyOptions configures Apply and Retry.
type ApplyOptions struct {
	Common
	// Reopen reopens campaign PRs that were closed without merging.
	Reopen bool
	// CloseNoop closes open PRs whose change is already on the base.
	CloseNoop bool
}

// saveEvery is how many finished repos pass between state saves.
const saveEvery = 25

// workspaces returns the workspace manager with --keep-workdirs applied.
func (a *App) workspaces(c Common) *workspace.Manager {
	m := *a.Workspaces
	m.Git = a.Git
	m.Keep = m.Keep || c.KeepWorkdirs
	return &m
}

// Plan runs the change in every target and shows the diffs, without
// pushing or writing anything to GitHub.
func (a *App) Plan(ctx context.Context, c Common) error {
	camp, err := a.LoadCampaign(c)
	if err != nil {
		return err
	}
	script, err := a.script(ctx, camp)
	if err != nil {
		return err
	}
	client, err := a.NewClient(c.WriteRate)
	if err != nil {
		return err
	}
	res, err := a.resolve(ctx, client, camp, c.Repos, true)
	if err != nil {
		return err
	}
	p := a.printer(c)
	progress := a.errPrinter(c)
	pipe := &engine.Pipeline{
		Mode: engine.Plan, Campaign: camp, Workspaces: a.workspaces(c), Git: a.Git, Script: script,
		LogPath: func(repo string) string { return a.Store.LogPath(camp.Name, repo) },
	}
	results, fatal := engine.Run(ctx, res.Targets, pipe, engine.Options{
		Concurrency: concurrency(c, camp),
		OnResult: func(rs state.RepoState) {
			if p.Machine() {
				return
			}
			if rs.Status == state.Changed {
				p.Diff(rs.Repo, rs.Diff)
				return
			}
			progress.Progress(rs)
		},
	})
	results = append(results, skippedStates(res)...)
	sortStates(results)
	if p.Machine() {
		if err := p.PrintJSON(results); err != nil {
			return err
		}
	} else {
		fmt.Fprintln(progress.Out, progress.Summary(render.Count(results)))
	}
	if fatal != nil {
		return fatal
	}
	return failures(results, statusHint)
}

// Apply runs the change, pushes branches and opens or updates PRs.
func (a *App) Apply(ctx context.Context, o ApplyOptions) error {
	return a.apply(ctx, o, nil)
}

// Retry re-runs only repos that failed, never ran, or were not pushed.
func (a *App) Retry(ctx context.Context, o ApplyOptions) error {
	return a.apply(ctx, o, func(st *state.Campaign, t target.Target) bool {
		rs := st.Get(t.Repo.FullName())
		if rs == nil {
			return true
		}
		switch rs.Status {
		case state.Failed, state.Pending, state.Changed:
			return true
		}
		return false
	})
}

// apply is Apply with an optional filter over targets.
func (a *App) apply(ctx context.Context, o ApplyOptions, keep func(*state.Campaign, target.Target) bool) error {
	camp, err := a.LoadCampaign(o.Common)
	if err != nil {
		return err
	}
	script, err := a.script(ctx, camp)
	if err != nil {
		return err
	}
	client, err := a.NewClient(o.WriteRate)
	if err != nil {
		return err
	}
	res, err := a.resolve(ctx, client, camp, o.Repos, true)
	if err != nil {
		return err
	}
	st, err := a.Store.Load(camp.Name)
	if err != nil {
		return err
	}
	st.File = camp.Path
	if keep != nil {
		var kept []target.Target
		for _, t := range res.Targets {
			if keep(st, t) {
				kept = append(kept, t)
			}
		}
		res.Targets = kept
	}
	p := a.printer(o.Common)
	progress := a.errPrinter(o.Common)
	if len(res.Targets) == 0 {
		if p.Machine() {
			return p.PrintJSON([]state.RepoState{})
		}
		fmt.Fprintln(progress.Out, "Nothing to do.")
		return nil
	}

	title := fmt.Sprintf("Push %s and open or update PRs in %s?", camp.PR.Branch, plural(len(res.Targets), "repo"))
	if err := a.confirm(o.Common, title); err != nil {
		return err
	}

	prs := pr.NewManager(client, camp.Limits.MaxOpenPRs)
	if err := prs.Index(ctx, res.Names(), camp.PR.Branch, camp.Marker(), searchScope(camp)); err != nil {
		return err
	}
	pipe := &engine.Pipeline{
		Mode: engine.Apply, Campaign: camp, Workspaces: a.workspaces(o.Common), Git: a.Git, PRs: prs, Script: script,
		LogPath: func(repo string) string { return a.Store.LogPath(camp.Name, repo) },
		Reopen:  o.Reopen, CloseNoop: o.CloseNoop,
	}
	done := 0
	results, fatal := engine.Run(ctx, res.Targets, pipe, engine.Options{
		Concurrency: concurrency(o.Common, camp),
		OnResult: func(rs state.RepoState) {
			st.Set(rs, a.Store.Time())
			done++
			if done%saveEvery == 0 {
				_ = a.Store.Save(st)
			}
			if !p.Machine() {
				progress.Progress(rs)
			}
		},
	})
	for _, r := range results {
		if r.Status == state.Pending {
			if old := st.Get(r.Repo); old != nil {
				r.PRNumber, r.PRURL = old.PRNumber, old.PRURL
			}
			st.Set(r, a.Store.Time())
		}
	}
	for _, s := range skippedStates(res) {
		st.Set(s, a.Store.Time())
		results = append(results, s)
	}
	if err := a.Store.Save(st); err != nil {
		return err
	}
	sortStates(results)
	if p.Machine() {
		if err := p.PrintJSON(results); err != nil {
			return err
		}
	} else {
		fmt.Fprintln(progress.Out, progress.Summary(render.Count(results)))
	}
	if fatal != nil {
		return fatal
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return failures(results, statusHint)
}

// indexPRs loads a campaign's PRs for repos.
func indexPRs(ctx context.Context, client ghapi.Client, camp *campaign.Campaign, repos []string) (*pr.Manager, error) {
	m := pr.NewManager(client, 0)
	if err := m.Index(ctx, repos, camp.PR.Branch, camp.Marker(), searchScope(camp)); err != nil {
		return nil, err
	}
	return m, nil
}
