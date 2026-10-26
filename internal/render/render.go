// Package render writes command output: aligned tables and progress lines
// for people, and JSON (optionally filtered with a jq expression) for
// scripts. Machine output is never styled.
package render

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/cli/go-gh/v2/pkg/jq"

	"github.com/rossedman/gh-fleet/internal/state"
	"github.com/rossedman/gh-fleet/internal/ui"
)

// Printer writes one command's output.
type Printer struct {
	Out io.Writer
	// JSON switches to machine output.
	JSON bool
	// Fields keeps only these fields of each JSON record; empty keeps all.
	Fields []string
	// JQ filters JSON output; setting it implies JSON.
	JQ string
	// Template formats JSON output with a Go template; setting it implies
	// JSON.
	Template string
	// TTY lays tables out for people. Without it, as in a pipe, tables are
	// tab-separated with no header row, the way gh prints them.
	TTY bool
	// Styles colors human output; use ui.Plain() for pipes.
	Styles ui.Styles
}

// Machine reports whether output is JSON.
func (p Printer) Machine() bool { return p.JSON || p.JQ != "" || p.Template != "" }

// PrintJSON writes v as indented JSON, keeping only Fields when set, then
// filters it with the jq expression or formats it with the template.
func (p Printer) PrintJSON(v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if len(p.Fields) > 0 {
		if b, err = SelectFields(b, p.Fields); err != nil {
			return err
		}
	}
	switch {
	case p.JQ != "":
		return jq.Evaluate(bytes.NewReader(b), p.Out, p.JQ)
	case p.Template != "":
		return executeTemplate(p.Out, p.Template, b)
	}
	var out bytes.Buffer
	if err := json.Indent(&out, b, "", "  "); err != nil {
		return err
	}
	out.WriteByte('\n')
	_, err = p.Out.Write(out.Bytes())
	return err
}

// Table writes rows under headers with aligned columns. Styling is applied
// per cell by style, which may be nil.
func (p Printer) Table(headers []string, rows [][]string, style func(row, col int, s string) string) {
	if !p.TTY {
		clean := strings.NewReplacer("\t", " ", "\n", " ", "\r", " ")
		for _, r := range rows {
			cells := make([]string, min(len(r), len(headers)))
			for i := range cells {
				cells[i] = clean.Replace(r[i])
			}
			fmt.Fprintln(p.Out, strings.Join(cells, "\t"))
		}
		return
	}
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = lipgloss.Width(h)
	}
	for _, r := range rows {
		for i, c := range r {
			if i < len(widths) {
				widths[i] = max(widths[i], lipgloss.Width(c))
			}
		}
	}
	line := func(cells []string, styleCell func(col int, s string) string) {
		var b strings.Builder
		for i, c := range cells {
			if i >= len(widths) {
				break
			}
			cell := c
			if styleCell != nil {
				cell = styleCell(i, c)
			}
			b.WriteString(cell)
			if i < len(cells)-1 {
				b.WriteString(strings.Repeat(" ", widths[i]-lipgloss.Width(c)+2))
			}
		}
		fmt.Fprintln(p.Out, strings.TrimRight(b.String(), " "))
	}
	line(headers, func(_ int, s string) string { return p.Styles.Header.Render(s) })
	for ri, r := range rows {
		line(r, func(col int, s string) string {
			if style == nil {
				return s
			}
			return style(ri, col, s)
		})
	}
}

// StatusStyle colors a repo status word.
func (p Printer) StatusStyle(s state.Status) lipgloss.Style {
	switch s {
	case state.Merged, state.Done:
		return p.Styles.OK
	case state.PROpen, state.Changed:
		return p.Styles.Brand
	case state.Failed:
		return p.Styles.Bad
	case state.Skipped, state.Pending, state.Closed:
		return p.Styles.Warn
	default:
		return p.Styles.Dim
	}
}

// Label is the human word for a status.
func Label(s state.Status) string {
	if s == state.PROpen {
		return "pr open"
	}
	return string(s)
}

// statusOrder sorts statuses in a summary line.
var statusOrder = map[state.Status]int{
	state.Changed: 0, state.PROpen: 1, state.Merged: 2, state.Done: 3, state.Noop: 4,
	state.Closed: 5, state.Pending: 6, state.Failed: 7, state.Skipped: 8,
}

// Summary returns e.g. "389 changed · 23 noop · 4 failed · 2 skipped".
func (p Printer) Summary(counts map[state.Status]int) string {
	keys := make([]state.Status, 0, len(counts))
	for k, n := range counts {
		if n > 0 {
			keys = append(keys, k)
		}
	}
	sort.Slice(keys, func(i, j int) bool { return statusOrder[keys[i]] < statusOrder[keys[j]] })
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = p.StatusStyle(k).Render(fmt.Sprintf("%d %s", counts[k], Label(k)))
	}
	if len(parts) == 0 {
		return "no repos"
	}
	return strings.Join(parts, p.Styles.Dim.Render(" · "))
}

// Count tallies statuses.
func Count(results []state.RepoState) map[state.Status]int {
	m := map[state.Status]int{}
	for _, r := range results {
		m[r.Status]++
	}
	return m
}

// Progress writes one line for a finished repo.
func (p Printer) Progress(rs state.RepoState) {
	label := fmt.Sprintf("%-8s", Label(rs.Status))
	line := p.StatusStyle(rs.Status).Render(label) + " " + rs.Repo
	if rs.PRURL != "" {
		line += " " + p.Styles.Dim.Render(rs.PRURL)
	}
	switch {
	case rs.LastError != "" && rs.Status == state.Failed:
		line += " " + p.Styles.Bad.Render(firstLine(rs.LastError))
	case rs.Detail != "":
		line += " " + p.Styles.Dim.Render(rs.Detail)
	}
	fmt.Fprintln(p.Out, line)
}

// Diff writes a unified diff with added, removed and hunk lines colored.
func (p Printer) Diff(repo, diff string) {
	fmt.Fprintln(p.Out, p.Styles.Header.Render("── "+repo))
	for _, l := range strings.Split(strings.TrimRight(diff, "\n"), "\n") {
		switch {
		case strings.HasPrefix(l, "+++"), strings.HasPrefix(l, "---"), strings.HasPrefix(l, "diff "), strings.HasPrefix(l, "index "):
			l = p.Styles.Bold.Render(l)
		case strings.HasPrefix(l, "+"):
			l = p.Styles.Add.Render(l)
		case strings.HasPrefix(l, "-"):
			l = p.Styles.Del.Render(l)
		case strings.HasPrefix(l, "@@"):
			l = p.Styles.Hunk.Render(l)
		}
		fmt.Fprintln(p.Out, l)
	}
	fmt.Fprintln(p.Out)
}

// firstLine returns the first line of s.
func firstLine(s string) string {
	l, _, _ := strings.Cut(s, "\n")
	return l
}
