package app

import (
	"fmt"
	"time"

	"github.com/rossedman/gh-fleet/internal/state"
)

// List prints the campaigns in local state.
func (a *App) List(c Common) error {
	sums, err := a.Store.List()
	if err != nil {
		return err
	}
	p := a.printer(c)
	if p.Machine() {
		if sums == nil {
			sums = []state.Summary{}
		}
		return p.PrintJSON(sums)
	}
	if len(sums) == 0 {
		fmt.Fprintln(a.Err, "No campaigns yet. Run gh fleet init to start one.")
		return nil
	}
	rows := make([][]string, len(sums))
	for i, s := range sums {
		updated := "-"
		if !s.UpdatedAt.IsZero() {
			updated = s.UpdatedAt.Local().Format(time.DateTime)
		}
		rows[i] = []string{s.Name, updated, p.Summary(s.Counts), s.File}
	}
	p.Table([]string{"CAMPAIGN", "UPDATED", "REPOS", "FILE"}, rows, func(_, col int, s string) string {
		if col == 0 {
			return p.Styles.Bold.Render(s)
		}
		if col == 1 || col == 3 {
			return p.Styles.Dim.Render(s)
		}
		return s
	})
	return nil
}
