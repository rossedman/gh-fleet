package app

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/rossedman/gh-fleet/internal/campaign"
	"github.com/rossedman/gh-fleet/internal/target"
)

// Targets prints the repos a campaign will touch and why each matched.
func (a *App) Targets(ctx context.Context, c Common) error {
	camp, err := a.LoadCampaign(c)
	if err != nil {
		return err
	}
	// Check the change script now, so a broken one shows up before plan.
	if _, err := a.scripts(ctx, camp); err != nil {
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
	if p.Machine() {
		return p.PrintJSON(res)
	}
	showVars := false
	for _, t := range res.Targets {
		showVars = showVars || len(t.Vars) > 0
	}
	headers := []string{"REPO", "MATCHED BY"}
	if showVars {
		headers = append(headers, "VARS")
	}
	rows := make([][]string, 0, len(res.Targets))
	for _, t := range res.Targets {
		row := []string{t.Repo.FullName(), strings.Join(t.Reasons, ", ")}
		if showVars {
			row = append(row, formatVars(t.Vars))
		}
		rows = append(rows, row)
	}
	if len(rows) > 0 {
		p.Table(headers, rows, func(_, col int, s string) string {
			if col == 1 {
				return p.Styles.Dim.Render(s)
			}
			return s
		})
	}
	// Skipped repos and the summary are for people: a pipe gets only the
	// targets on stdout.
	e := a.errPrinter(c)
	if len(res.Skipped) > 0 {
		sp := p
		if !p.TTY {
			sp = e
		}
		fmt.Fprintln(sp.Out)
		rows = rows[:0]
		for _, s := range res.Skipped {
			rows = append(rows, []string{s.Repo, s.Reason})
		}
		sp.Table([]string{"SKIPPED", "REASON"}, rows, func(_, col int, s string) string {
			if col == 1 {
				return sp.Styles.Warn.Render(s)
			}
			return s
		})
	}
	fmt.Fprintf(e.Out, "\n%s", e.Styles.Bold.Render(plural(len(res.Targets), "repo")))
	if len(res.Skipped) > 0 {
		fmt.Fprintf(e.Out, " %s", e.Styles.Dim.Render(fmt.Sprintf("(%d skipped)", len(res.Skipped))))
	}
	fmt.Fprintln(e.Out)
	if extra := unmatchedRows(camp, res); len(extra) > 0 && len(c.Repos) == 0 {
		what := "is not a target"
		if len(extra) > 1 {
			what = "are not targets"
		}
		fmt.Fprintf(e.Out, "%s %s names %s that %s: %s\n",
			e.Styles.Warn.Render("warning:"), camp.Vars.File, plural(len(extra), "repo"), what, strings.Join(extra, ", "))
	}
	return nil
}

// formatVars renders variables as sorted name=value pairs.
func formatVars(vars map[string]any) string {
	parts := make([]string, 0, len(vars))
	for _, k := range slices.Sorted(maps.Keys(vars)) {
		parts = append(parts, k+"="+campaign.FormatValue(vars[k]))
	}
	return strings.Join(parts, " ")
}

// unmatchedRows returns the vars.file repos that are not targets, which is
// usually a typo or a renamed repo.
func unmatchedRows(camp *campaign.Campaign, res *target.Result) []string {
	targets := map[string]bool{}
	for _, t := range res.Targets {
		targets[strings.ToLower(t.Repo.FullName())] = true
	}
	for _, s := range res.Skipped {
		targets[strings.ToLower(s.Repo)] = true
	}
	var out []string
	for _, r := range camp.Vars.RowRepos() {
		if !targets[r] {
			out = append(out, r)
		}
	}
	return out
}

// plural formats a count with a noun.
func plural(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return fmt.Sprintf("%d %ss", n, noun)
}
