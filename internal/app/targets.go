package app

import (
	"context"
	"fmt"
	"strings"
)

// Targets prints the repos a campaign will touch and why each matched.
func (a *App) Targets(ctx context.Context, c Common) error {
	camp, err := a.LoadCampaign(c)
	if err != nil {
		return err
	}
	client, err := a.NewClient(c.WriteRate)
	if err != nil {
		return err
	}
	res, err := a.resolve(ctx, client, camp.Targets, c.Repos, true)
	if err != nil {
		return err
	}
	p := a.printer(c)
	if p.Machine() {
		return p.PrintJSON(res)
	}
	rows := make([][]string, 0, len(res.Targets))
	for _, t := range res.Targets {
		rows = append(rows, []string{t.Repo.FullName(), strings.Join(t.Reasons, ", ")})
	}
	if len(rows) > 0 {
		p.Table([]string{"REPO", "MATCHED BY"}, rows, func(_, col int, s string) string {
			if col == 1 {
				return p.Styles.Dim.Render(s)
			}
			return s
		})
	}
	if len(res.Skipped) > 0 {
		fmt.Fprintln(p.Out)
		rows = rows[:0]
		for _, s := range res.Skipped {
			rows = append(rows, []string{s.Repo, s.Reason})
		}
		p.Table([]string{"SKIPPED", "REASON"}, rows, func(_, col int, s string) string {
			if col == 1 {
				return p.Styles.Warn.Render(s)
			}
			return s
		})
	}
	fmt.Fprintf(p.Out, "\n%s", p.Styles.Bold.Render(plural(len(res.Targets), "repo")))
	if len(res.Skipped) > 0 {
		fmt.Fprintf(p.Out, " %s", p.Styles.Dim.Render(fmt.Sprintf("(%d skipped)", len(res.Skipped))))
	}
	fmt.Fprintln(p.Out)
	return nil
}

// plural formats a count with a noun.
func plural(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return fmt.Sprintf("%d %ss", n, noun)
}
