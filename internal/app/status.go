package app

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rossedman/gh-fleet/internal/ghapi"
	"github.com/rossedman/gh-fleet/internal/render"
	"github.com/rossedman/gh-fleet/internal/state"
)

// StatusReport is the JSON form of gh fleet status.
type StatusReport struct {
	Campaign string            `json:"campaign"`
	Repos    []state.RepoState `json:"repos"`
	Counts   map[string]int    `json:"counts"`
	Quota    []ghapi.RateLimit `json:"quota,omitempty"`
}

// StatusOptions configures Status.
type StatusOptions struct {
	Common
	// Format is table, markdown or html.
	Format string
	// ExitStatus fails when a repo failed, and returns ErrPending when one
	// still needs work, after printing.
	ExitStatus bool
}

// Status refreshes every target's PR from GitHub and prints a table, or a
// Markdown or HTML report. PRs are looked up by branch (and by marker as a
// fallback), so the report is correct even when the local state file is
// gone.
func (a *App) Status(ctx context.Context, o StatusOptions) error {
	rows, err := a.status(ctx, o)
	if err != nil || !o.ExitStatus {
		return err
	}
	return exitStatus(rows)
}

// exitStatus fails when a repo failed and returns ErrPending while one
// still needs work. Merged, closed, no-op and skipped repos are done.
func exitStatus(rows []state.RepoState) error {
	failed, pending := 0, false
	for _, r := range rows {
		switch r.Status {
		case state.Failed:
			failed++
		case state.Merged, state.Closed, state.Noop, state.Skipped, state.Done:
		default:
			pending = true
		}
	}
	if failed > 0 {
		return &FailedError{N: failed}
	}
	if pending {
		return ErrPending
	}
	return nil
}

// status prints the status and returns the rows it printed.
func (a *App) status(ctx context.Context, o StatusOptions) (rows []state.RepoState, err error) {
	c := o.Common
	switch o.Format {
	case "", FormatTable:
	case FormatMarkdown, FormatHTML:
		if c.JSON || c.JQ != "" || c.Template != "" {
			return nil, fmt.Errorf("--format %s can't be combined with --json, --jq or --template", o.Format)
		}
	default:
		return nil, fmt.Errorf("--format must be table, markdown or html, not %q", o.Format)
	}
	camp, err := a.LoadCampaign(c)
	if err != nil {
		return nil, err
	}
	client, err := a.NewClient(c.WriteRate)
	if err != nil {
		return nil, err
	}
	res, err := a.resolve(ctx, c, client, camp, c.Repos, true)
	if err != nil {
		return nil, err
	}
	st, err := a.Store.Load(camp.Name)
	if err != nil {
		return nil, err
	}
	if st.File == "" {
		st.File = camp.Path
	}
	names := res.Names()
	inTargets := map[string]bool{}
	for _, n := range names {
		inTargets[n] = true
	}
	if len(c.Repos) == 0 {
		for _, r := range st.Sorted() {
			if !inTargets[r.Repo] && r.PRNumber > 0 && !excludedName(camp, r.Repo) {
				names = append(names, r.Repo)
				inTargets[r.Repo] = true
			}
		}
	}
	prs, err := indexPRs(ctx, client, camp, names)
	if err != nil {
		return nil, err
	}

	now := a.Store.Time()
	for _, n := range names {
		rs := state.RepoState{Repo: n, Status: state.Pending}
		if old := st.Get(n); old != nil {
			rs = *old
		}
		if p := prs.Existing(n); p != nil {
			rs.PRNumber, rs.PRURL, rs.Checks, rs.Review, rs.MergeState = p.Number, p.URL, p.Checks, p.Review, p.MergeState
			if p.HeadSHA != "" {
				rs.HeadSHA = p.HeadSHA
			}
			switch p.State {
			case ghapi.StateOpen:
				rs.Status = state.PROpen
			case ghapi.StateMerged:
				rs.Status = state.Merged
			case ghapi.StateClosed:
				rs.Status = state.Closed
			}
		} else if rs.PRNumber > 0 {
			// A PR recorded in state is gone from GitHub.
			rs.PRNumber, rs.PRURL, rs.Status = 0, "", state.Pending
		}
		st.Set(rs, now)
		rows = append(rows, *st.Get(n))
	}
	rows = append(rows, skippedStates(res)...)
	sortStates(rows)
	if err := a.Store.Save(st); err != nil {
		return nil, err
	}
	quota, _ := client.RateLimits(ctx)

	if o.Format == FormatMarkdown || o.Format == FormatHTML {
		return rows, WriteReport(a.Out, o.Format, camp, rows, now)
	}
	p := a.printer(c)
	counts := render.Count(rows)
	if p.Machine() {
		rep := StatusReport{Campaign: camp.Name, Repos: rows, Counts: map[string]int{}, Quota: quota}
		for k, v := range counts {
			rep.Counts[string(k)] = v
		}
		return rows, p.PrintJSON(rep)
	}
	table := make([][]string, len(rows))
	for i, r := range rows {
		prCol := "-"
		if r.PRNumber > 0 {
			prCol = r.PRURL
		}
		note := ""
		switch r.Status {
		case state.Failed:
			note = r.LastError
		case state.Noop, state.Skipped, state.Changed, state.Pending, state.Closed:
			note = r.Detail
		}
		table[i] = []string{r.Repo, render.Label(r.Status), prCol, orDash(strings.ToLower(r.Checks)), orDash(strings.ToLower(strings.ReplaceAll(r.Review, "_", " "))), firstLine(note)}
	}
	p.Table([]string{"REPO", "STATUS", "PR", "CHECKS", "REVIEW", "NOTE"}, table, func(row, col int, s string) string {
		switch col {
		case 1:
			return p.StatusStyle(rows[row].Status).Render(s)
		case 3:
			return checkStyle(p, s)
		case 5:
			if rows[row].Status == state.Failed {
				return p.Styles.Bad.Render(s)
			}
			return p.Styles.Dim.Render(s)
		}
		return s
	})
	// The summary is for people, so a pipe gets only the rows.
	e := a.errPrinter(c)
	fmt.Fprintf(e.Out, "\n%s\n", e.Summary(counts))
	if len(quota) > 0 {
		fmt.Fprintln(e.Out, e.Styles.Dim.Render(quotaLine(quota, now)))
	}
	return rows, nil
}

// checkStyle colors a checks rollup state.
func checkStyle(p render.Printer, s string) string {
	switch s {
	case "success":
		return p.Styles.OK.Render(s)
	case "failure", "error":
		return p.Styles.Bad.Render(s)
	case "pending", "expected":
		return p.Styles.Warn.Render(s)
	}
	return p.Styles.Dim.Render(s)
}

// quotaLine formats remaining API quota.
func quotaLine(q []ghapi.RateLimit, now time.Time) string {
	sort.Slice(q, func(i, j int) bool { return q[i].Resource < q[j].Resource })
	parts := make([]string, 0, len(q))
	for _, r := range q {
		if r.Resource != "core" && r.Resource != "graphql" {
			continue
		}
		part := fmt.Sprintf("%s %s/%s", r.Resource, comma(r.Remaining), comma(r.Limit))
		if reset := r.Reset.Sub(now).Round(time.Minute); reset > 0 {
			part += fmt.Sprintf(" (resets in %s)", strings.TrimSuffix(reset.String(), "0s"))
		}
		parts = append(parts, part)
	}
	return "API quota: " + strings.Join(parts, " · ")
}

// comma formats an integer with thousands separators.
func comma(n int) string {
	s := strconv.Itoa(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

// orDash returns s, or "-" when empty.
func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// firstLine returns the first line of s.
func firstLine(s string) string {
	l, _, _ := strings.Cut(s, "\n")
	return l
}
