package app

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

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

// PlanOptions configures Plan.
type PlanOptions struct {
	Common
	// Ungrouped prints each repo's diff as it finishes, instead of each
	// distinct diff once at the end with the repos that get it.
	Ungrouped bool
}

// Plan runs the change in every target and shows the diffs, without
// pushing or writing anything to GitHub.
func (a *App) Plan(ctx context.Context, o PlanOptions) error {
	c := o.Common
	camp, err := a.LoadCampaign(c)
	if err != nil {
		return err
	}
	scripts, err := a.scripts(ctx, camp)
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
		Mode: engine.Plan, Campaign: camp, Workspaces: a.workspaces(c), Git: a.Git, Scripts: scripts,
		LogPath: func(repo string) string { return a.Store.LogPath(camp.Name, repo) },
	}
	results, fatal := engine.Run(ctx, res.Targets, pipe, engine.Options{
		Concurrency: concurrency(c, camp),
		OnResult: func(rs state.RepoState) {
			if p.Machine() {
				return
			}
			if rs.Status == state.Changed && o.Ungrouped {
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
		if !o.Ungrouped {
			for _, g := range groupDiffs(results) {
				p.Diff(groupHeader(g.repos), g.diff)
			}
		}
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
	scripts, err := a.scripts(ctx, camp)
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
	// Waves are worked out over every target, before retry narrows them.
	all := res.Names()
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

	prs := pr.NewManager(client, camp.Limits.MaxOpenPRs)
	if err := prs.Index(ctx, all, camp.PR.Branch, camp.Marker(), searchScope(camp)); err != nil {
		return err
	}

	// Waves hold back repos until the waves before theirs are done. An
	// explicit --repo is a deliberate choice and isn't held.
	var held []state.RepoState
	if len(camp.Limits.Waves) > 0 && len(o.Repos) == 0 {
		gate := gateWaves(camp.Limits.Waves, all, func(r string) bool {
			if p := prs.Existing(r); p != nil && p.State == ghapi.StateMerged {
				return true
			}
			rs := st.Get(r)
			return rs != nil && rs.Status == state.Noop
		}, func(r string) bool {
			p := prs.Existing(r)
			return p != nil && p.State == ghapi.StateOpen
		})
		var kept []target.Target
		for _, t := range res.Targets {
			name := t.Repo.FullName()
			if gate.Allowed[name] {
				kept = append(kept, t)
				continue
			}
			rs := state.RepoState{Repo: name, Status: state.Pending, Detail: gate.Held[name]}
			if old := st.Get(name); old != nil {
				rs.PRNumber, rs.PRURL = old.PRNumber, old.PRURL
			}
			held = append(held, rs)
		}
		res.Targets = kept
		if !p.Machine() {
			verb := "wait"
			if len(held) == 1 {
				verb = "waits"
			}
			fmt.Fprintf(progress.Out, "Wave %d of %d: %s now; %s %s for it.\n", gate.Current, gate.Waves, plural(len(kept), "repo"), plural(len(held), "repo"), verb)
		}
	}
	if len(res.Targets) == 0 {
		for _, h := range held {
			st.Set(h, a.Store.Time())
		}
		if err := a.Store.Save(st); err != nil {
			return err
		}
		if p.Machine() {
			return p.PrintJSON(held)
		}
		fmt.Fprintln(progress.Out, "Nothing to do until the current wave is done.")
		return nil
	}

	title := fmt.Sprintf("Push %s and open or update PRs in %s?", camp.PR.Branch, plural(len(res.Targets), "repo"))
	if err := a.confirm(o.Common, title); err != nil {
		return err
	}
	pipe := &engine.Pipeline{
		Mode: engine.Apply, Campaign: camp, Workspaces: a.workspaces(o.Common), Git: a.Git, PRs: prs, Scripts: scripts,
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
	for _, h := range held {
		if !p.Machine() {
			progress.Progress(h)
		}
	}
	for _, s := range append(skippedStates(res), held...) {
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

// diffGroup is a diff and the repos whose change it is.
type diffGroup struct {
	diff  string
	repos []string
}

// hunkHeader matches a hunk header, whose line numbers differ between
// repos making the same change.
var hunkHeader = regexp.MustCompile(`(?m)^@@ -\d+(?:,\d+)? \+\d+(?:,\d+)? @@`)

// diffKey is a diff without what differs between repos making the same
// change: blob hashes and hunk line numbers.
func diffKey(diff string) string {
	lines := strings.Split(diff, "\n")
	out := lines[:0:0]
	for _, l := range lines {
		if !strings.HasPrefix(l, "index ") {
			out = append(out, l)
		}
	}
	return hunkHeader.ReplaceAllString(strings.Join(out, "\n"), "@@")
}

// groupDiffs groups changed repos by diff, largest group first, so a
// change made the same way in many repos is read once.
func groupDiffs(results []state.RepoState) []diffGroup {
	byKey := map[string]*diffGroup{}
	var groups []*diffGroup
	for _, r := range results {
		if r.Status != state.Changed || r.Diff == "" {
			continue
		}
		k := diffKey(r.Diff)
		g, ok := byKey[k]
		if !ok {
			g = &diffGroup{diff: r.Diff}
			byKey[k] = g
			groups = append(groups, g)
		}
		g.repos = append(g.repos, r.Repo)
	}
	sort.SliceStable(groups, func(i, j int) bool { return len(groups[i].repos) > len(groups[j].repos) })
	out := make([]diffGroup, len(groups))
	for i, g := range groups {
		out[i] = *g
	}
	return out
}

// groupHeader names the repos of a diff group: all of them up to ten,
// else the first ten and how many more.
func groupHeader(repos []string) string {
	if len(repos) == 1 {
		return repos[0]
	}
	shown := repos
	more := ""
	if len(repos) > 10 {
		shown, more = repos[:10], fmt.Sprintf(" and %d more", len(repos)-10)
	}
	return fmt.Sprintf("%d repos: %s%s", len(repos), strings.Join(shown, ", "), more)
}
