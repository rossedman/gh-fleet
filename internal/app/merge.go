package app

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/rossedman/gh-fleet/internal/engine"
	"github.com/rossedman/gh-fleet/internal/ghapi"
	"github.com/rossedman/gh-fleet/internal/state"
	"github.com/rossedman/gh-fleet/internal/target"
)

// MergeOptions configures Merge.
type MergeOptions struct {
	Common
	// Method is merge, squash or rebase.
	Method string
	// Auto turns on auto-merge for PRs that aren't ready yet, so GitHub
	// merges them when their checks and reviews pass.
	Auto bool
	// UpdateBranch merges the base into PRs that are behind it.
	UpdateBranch bool
	// Max limits how many PRs are merged or set to auto-merge; zero means
	// no limit.
	Max int
	// AllowUnreviewed merges PRs in repos that don't require a review.
	AllowUnreviewed bool
	// AllowNoChecks merges PRs whose head has no checks.
	AllowNoChecks bool
	// KeepBranches leaves branches of merged PRs in place.
	KeepBranches bool
	// DryRun reports what would happen without writing.
	DryRun bool
}

// Merge actions.
const (
	mergeNow    = "merge"
	mergeAuto   = "auto-merge"
	mergeUpdate = "update-branch"
	mergeWait   = "wait"
)

// MergeResult is what Merge did, or would do, with one PR.
type MergeResult struct {
	Repo   string `json:"repo"`
	Number int    `json:"number"`
	URL    string `json:"url"`
	// Action is merge, auto-merge, update-branch or wait.
	Action string `json:"action"`
	// Reason says why a PR waits.
	Reason string `json:"reason,omitempty"`
	// Done is false for a dry run or a failed write.
	Done  bool   `json:"done"`
	Error string `json:"error,omitempty"`
	// Note reports a problem after a successful merge, such as a branch
	// that couldn't be deleted.
	Note string `json:"note,omitempty"`
}

// decideMerge picks what to do with an open campaign PR: merge it now when
// GitHub reports it ready and it is approved and green, otherwise update a
// branch that is behind, turn on auto-merge, or say what it waits on.
func decideMerge(p ghapi.PR, o MergeOptions) (action, reason string) {
	switch {
	case p.Draft:
		return mergeWait, "draft"
	case p.AutoMerge:
		return mergeWait, "auto-merge already on"
	case p.Review == "CHANGES_REQUESTED":
		return mergeWait, "changes requested"
	case p.MergeState == "DIRTY":
		return mergeWait, "merge conflict; gh fleet apply rebases the branch"
	case p.Checks == "FAILURE" || p.Checks == "ERROR":
		return mergeWait, "checks failing"
	case p.MergeState == "UNSTABLE":
		return mergeWait, "checks failing that aren't required"
	case p.MergeState == "BEHIND":
		if o.UpdateBranch {
			return mergeUpdate, ""
		}
		return mergeWait, "behind its base; pass --update-branch"
	}
	reviewed := p.Review == "APPROVED" || (p.Review == "" && o.AllowUnreviewed)
	checked := p.Checks == "SUCCESS" || (p.Checks == "" && o.AllowNoChecks)
	if (p.MergeState == "CLEAN" || p.MergeState == "HAS_HOOKS") && reviewed && checked {
		return mergeNow, ""
	}
	var why string
	switch {
	case p.Checks == "PENDING" || p.Checks == "EXPECTED":
		why = "checks running"
	case p.Review == "REVIEW_REQUIRED":
		why = "needs review"
	case p.Review == "":
		why = "not approved; the repo doesn't require review, pass --allow-unreviewed"
	case p.Checks == "":
		why = "no checks ran; pass --allow-no-checks"
	case p.MergeState == "BLOCKED":
		why = "blocked by branch protection"
	default:
		why = "GitHub hasn't worked out if it can merge yet; run again"
	}
	// Auto-merge waits for what branch protection requires; it can't make
	// up for a missing review or missing checks in a repo that requires
	// neither, so those stay waiting.
	if o.Auto && (p.Review != "" || o.AllowUnreviewed) && (p.Checks != "" || o.AllowNoChecks) {
		return mergeAuto, ""
	}
	return mergeWait, why
}

// Merge merges, or sets to auto-merge, the campaign's open PRs that are
// ready, and reports what the others wait on.
func (a *App) Merge(ctx context.Context, o MergeOptions) error {
	if !slices.Contains([]string{ghapi.MergeMethodMerge, ghapi.MergeMethodSquash, ghapi.MergeMethodRebase}, o.Method) {
		return fmt.Errorf("--method must be merge, squash or rebase, not %q", o.Method)
	}
	if o.Max < 0 {
		return fmt.Errorf("--max must be zero or more")
	}
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
	res, err := a.resolve(ctx, o.Common, client, camp, o.Repos, false)
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
	open, err := freshPRs(ctx, client, prs.All())
	if err != nil {
		return err
	}

	p := a.printer(o.Common)
	progress := a.errPrinter(o.Common)
	if len(open) == 0 {
		if p.Machine() {
			return p.PrintJSON([]MergeResult{})
		}
		fmt.Fprintln(progress.Out, "No open campaign PRs.")
		return nil
	}

	// Decide up front, so --max picks the same PRs whatever order the
	// workers finish in, and the prompt can say how many will merge.
	plans := make([]MergeResult, len(open))
	writes := 0
	for i, pr := range open {
		action, reason := decideMerge(pr, o)
		if (action == mergeNow || action == mergeAuto) && o.Max > 0 && writes >= o.Max {
			action, reason = mergeWait, fmt.Sprintf("--max %d reached", o.Max)
		}
		if action == mergeNow || action == mergeAuto {
			writes++
		}
		plans[i] = MergeResult{Repo: pr.Repo, Number: pr.Number, URL: pr.URL, Action: action, Reason: reason}
	}
	counts := map[string]int{}
	for _, pl := range plans {
		counts[pl.Action]++
	}
	var todo []string
	for _, k := range []string{mergeNow, mergeAuto, mergeUpdate} {
		if n := counts[k]; n > 0 {
			todo = append(todo, fmt.Sprintf(map[string]string{mergeNow: "merge %s", mergeAuto: "turn on auto-merge for %s", mergeUpdate: "update the branch of %s"}[k], plural(n, "PR")))
		}
	}
	if !o.DryRun && len(todo) > 0 {
		if err := a.confirm(o.Common, "Going to "+strings.Join(todo, ", ")+" ("+o.Method+"). Continue?"); err != nil {
			return err
		}
	}

	byRepo := map[string]int{}
	targets := make([]target.Target, len(open))
	for i, pr := range open {
		byRepo[pr.Repo] = i
		owner, name, _ := ghapi.SplitRepo(pr.Repo)
		targets[i] = target.Target{Repo: ghapi.Repo{Owner: owner, Name: name}}
	}
	var mu sync.Mutex
	proc := engine.ProcessorFunc(func(ctx context.Context, t target.Target) (state.RepoState, error) {
		i := byRepo[t.Repo.FullName()]
		pr, plan := open[i], plans[i]
		if !o.DryRun {
			wctx := context.WithoutCancel(ctx)
			var err error
			switch plan.Action {
			case mergeNow:
				if err = client.MergePR(wctx, pr.Repo, pr.Number, o.Method, pr.HeadSHA); err == nil && !o.KeepBranches && pr.HeadRef != "" {
					if derr := client.DeleteBranch(wctx, pr.Repo, pr.HeadRef); derr != nil {
						plan.Note = "branch not deleted: " + derr.Error()
					}
				}
			case mergeAuto:
				err = client.EnableAutoMerge(wctx, pr, o.Method)
			case mergeUpdate:
				err = client.UpdateBranch(wctx, pr.Repo, pr.Number, pr.HeadSHA)
			}
			if err != nil {
				plan.Error = err.Error()
			} else {
				plan.Done = plan.Action != mergeWait
			}
		}
		mu.Lock()
		plans[i] = plan
		mu.Unlock()
		rs := state.RepoState{Repo: pr.Repo, PRNumber: pr.Number, PRURL: pr.URL, Status: state.PROpen, HeadSHA: pr.HeadSHA, Checks: pr.Checks, Review: pr.Review}
		switch {
		case plan.Error != "":
			rs.Status, rs.LastError = state.Failed, plan.Error
		case plan.Action == mergeWait:
			rs.Detail = "waiting: " + plan.Reason
		case o.DryRun:
			rs.Detail = map[string]string{mergeNow: "would merge", mergeAuto: "would turn on auto-merge", mergeUpdate: "would update the branch"}[plan.Action]
		case plan.Action == mergeNow:
			rs.Status, rs.Detail = state.Merged, plan.Note
		case plan.Action == mergeAuto:
			rs.Detail = "auto-merge on"
		case plan.Action == mergeUpdate:
			rs.Detail = "branch updated"
		}
		return rs, nil
	})
	tr, stop := a.track(o.Common, "Merging in", len(targets), concurrency(o.Common, camp), true)
	results, fatal := engine.Run(ctx, targets, proc, engine.Options{
		Concurrency: concurrency(o.Common, camp),
		OnResult: func(rs state.RepoState) {
			if rs.Status == state.Merged {
				st.Set(rs, a.Store.Time())
			}
			tr.Result(rs)
		},
	})
	stop()
	if !o.DryRun {
		if err := a.Store.Save(st); err != nil {
			return err
		}
	}
	if p.Machine() {
		if err := p.PrintJSON(plans); err != nil {
			return err
		}
	} else {
		fmt.Fprintln(progress.Out, mergeSummary(plans, o.DryRun))
	}
	if fatal != nil {
		return fatal
	}
	return failures(results, "")
}

// freshPRs reloads the open PRs with their checks, reviews and merge
// state, grouped by head branch, sorted by repo.
func freshPRs(ctx context.Context, client ghapi.Client, all []ghapi.PR) ([]ghapi.PR, error) {
	byHead := map[string][]string{}
	for _, p := range all {
		if p.State == ghapi.StateOpen {
			byHead[p.HeadRef] = append(byHead[p.HeadRef], p.Repo)
		}
	}
	var open []ghapi.PR
	for head, repos := range byHead {
		got, err := client.PRsByHead(ctx, repos, head)
		if err != nil {
			return nil, err
		}
		for _, p := range got {
			if p.State == ghapi.StateOpen {
				open = append(open, p)
			}
		}
	}
	sort.Slice(open, func(i, j int) bool { return open[i].Repo < open[j].Repo })
	return open, nil
}

// mergeSummary counts what Merge did.
func mergeSummary(plans []MergeResult, dryRun bool) string {
	n := map[string]int{}
	failed := 0
	for _, p := range plans {
		if p.Error != "" {
			failed++
			continue
		}
		n[p.Action]++
	}
	var parts []string
	verbs := map[string]string{mergeNow: "merged", mergeAuto: "set to auto-merge", mergeUpdate: "branches updated", mergeWait: "waiting"}
	if dryRun {
		verbs = map[string]string{mergeNow: "would merge", mergeAuto: "would auto-merge", mergeUpdate: "would update branch", mergeWait: "waiting"}
	}
	for _, k := range []string{mergeNow, mergeAuto, mergeUpdate, mergeWait} {
		if n[k] > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", n[k], verbs[k]))
		}
	}
	if failed > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", failed))
	}
	return strings.Join(parts, " · ")
}
