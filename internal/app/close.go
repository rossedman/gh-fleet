package app

import (
	"context"
	"fmt"
	"sort"

	"github.com/rossedman/gh-fleet/internal/engine"
	"github.com/rossedman/gh-fleet/internal/ghapi"
	"github.com/rossedman/gh-fleet/internal/render"
	"github.com/rossedman/gh-fleet/internal/state"
	"github.com/rossedman/gh-fleet/internal/target"
)

// CloseOptions configures Close.
type CloseOptions struct {
	Common
	// Comment is posted on each PR before closing it.
	Comment string
	// KeepBranches leaves the campaign branches in place.
	KeepBranches bool
}

// Close closes every open campaign PR and deletes its branch.
func (a *App) Close(ctx context.Context, o CloseOptions) error {
	camp, err := a.LoadCampaign(o.Common)
	if err != nil {
		return err
	}
	client, err := a.NewClient(o.WriteRate)
	if err != nil {
		return err
	}
	st, err := a.Store.Load(camp.Name)
	if err != nil {
		return err
	}
	res, err := a.resolve(ctx, client, camp, o.Repos, true)
	if err != nil {
		return err
	}
	names := res.Names()
	seen := map[string]bool{}
	for _, n := range names {
		seen[n] = true
	}
	if len(o.Repos) == 0 {
		for _, r := range st.Sorted() {
			if !seen[r.Repo] && r.PRNumber > 0 && !excludedName(camp, r.Repo) {
				names = append(names, r.Repo)
			}
		}
	}
	prs, err := indexPRs(ctx, client, camp, names)
	if err != nil {
		return err
	}
	var open []ghapi.PR
	for _, p := range prs.All() {
		if p.State == ghapi.StateOpen {
			open = append(open, p)
		}
	}
	sort.Slice(open, func(i, j int) bool { return open[i].Repo < open[j].Repo })

	p := a.printer(o.Common)
	progress := a.errPrinter(o.Common)
	if len(open) == 0 {
		if p.Machine() {
			return p.PrintJSON([]state.RepoState{})
		}
		fmt.Fprintln(progress.Out, "No open campaign PRs.")
		return nil
	}
	what := "and delete their branches"
	if o.KeepBranches {
		what = "and keep their branches"
	}
	if err := a.confirm(o.Common, fmt.Sprintf("Close %s %s?", plural(len(open), "open PR"), what)); err != nil {
		return err
	}

	byRepo := map[string]ghapi.PR{}
	targets := make([]target.Target, len(open))
	for i, pr := range open {
		byRepo[pr.Repo] = pr
		owner, name, _ := ghapi.SplitRepo(pr.Repo)
		targets[i] = target.Target{Repo: ghapi.Repo{Owner: owner, Name: name}}
	}
	proc := engine.ProcessorFunc(func(ctx context.Context, t target.Target) (state.RepoState, error) {
		pr := byRepo[t.Repo.FullName()]
		rs := state.RepoState{Repo: pr.Repo, PRNumber: pr.Number, PRURL: pr.URL}
		if err := prs.Close(context.WithoutCancel(ctx), pr, o.Comment, !o.KeepBranches); err != nil {
			rs.Status, rs.LastError = state.Failed, err.Error()
			return rs, nil
		}
		rs.Status = state.Closed
		return rs, nil
	})
	results, fatal := engine.Run(ctx, targets, proc, engine.Options{
		Concurrency: concurrency(o.Common, camp),
		OnResult: func(rs state.RepoState) {
			if rs.Status == state.Closed {
				st.Set(rs, a.Store.Time())
			}
			if !p.Machine() {
				progress.Progress(rs)
			}
		},
	})
	if err := a.Store.Save(st); err != nil {
		return err
	}
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
