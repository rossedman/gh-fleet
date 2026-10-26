package app

import (
	"fmt"
	htmltemplate "html/template"
	"io"
	"strings"
	"text/template"
	"time"

	"github.com/rossedman/gh-fleet/internal/campaign"
	"github.com/rossedman/gh-fleet/internal/ghapi"
	"github.com/rossedman/gh-fleet/internal/render"
	"github.com/rossedman/gh-fleet/internal/state"
)

// Report formats for gh fleet status --format.
const (
	FormatTable    = "table"
	FormatMarkdown = "markdown"
	FormatHTML     = "html"
)

// reportRow is one repo in a report section.
type reportRow struct {
	Repo, URL, Status, Checks, Review, Note string
	Number                                  int
}

// report is a campaign's progress, grouped for people to read.
type report struct {
	Name, Description, Generated string
	// Total counts every target that isn't skipped; Done the merged and
	// noop ones, which need nothing more.
	Total, Done, Percent int
	Counts               []reportCount
	Attention, Open      []reportRow
	Merged, Other        []reportRow
}

// reportCount is one status and how many repos have it.
type reportCount struct {
	Label string
	N     int
}

// newReport groups status rows: what needs a person, open PRs with what
// they wait on, merged PRs, and everything else.
func newReport(camp *campaign.Campaign, rows []state.RepoState, now time.Time) report {
	r := report{Name: camp.Name, Description: camp.Description, Generated: now.UTC().Format("2006-01-02 15:04 UTC")}
	counts := render.Count(rows)
	for _, s := range []state.Status{state.Merged, state.PROpen, state.Changed, state.Failed, state.Pending, state.Noop, state.Closed, state.Skipped} {
		if counts[s] > 0 {
			r.Counts = append(r.Counts, reportCount{render.Label(s), counts[s]})
		}
	}
	for _, rs := range rows {
		row := reportRow{Repo: rs.Repo, URL: rs.PRURL, Number: rs.PRNumber, Status: render.Label(rs.Status),
			Checks: strings.ToLower(rs.Checks), Review: strings.ToLower(strings.ReplaceAll(rs.Review, "_", " "))}
		if rs.Status != state.Skipped {
			r.Total++
		}
		switch rs.Status {
		case state.Merged:
			r.Done++
			r.Merged = append(r.Merged, row)
		case state.Failed:
			row.Note = firstLine(rs.LastError)
			r.Attention = append(r.Attention, row)
		case state.PROpen:
			// The same reasons gh fleet merge gives.
			action, reason := decideMerge(ghapi.PR{Checks: rs.Checks, Review: rs.Review, MergeState: rs.MergeState}, MergeOptions{})
			switch {
			case action == mergeNow:
				row.Note = "ready to merge"
			case reason == "checks failing" || reason == "changes requested" || strings.HasPrefix(reason, "merge conflict"):
				row.Note = strings.TrimSuffix(reason, "; gh fleet apply rebases the branch")
				r.Attention = append(r.Attention, row)
				continue
			default:
				row.Note = reason
			}
			r.Open = append(r.Open, row)
		default:
			if rs.Status == state.Noop {
				r.Done++
			}
			row.Note = firstLine(rs.Detail)
			r.Other = append(r.Other, row)
		}
	}
	if r.Total > 0 {
		r.Percent = r.Done * 100 / r.Total
	}
	return r
}

// cell makes text safe inside a Markdown table cell.
func cell(s string) string {
	return strings.NewReplacer("|", `\|`, "\n", " ").Replace(s)
}

// bar draws a progress bar of ten blocks.
func bar(percent int) string {
	full := percent / 10
	return strings.Repeat("█", full) + strings.Repeat("░", 10-full)
}

var reportFuncs = template.FuncMap{"cell": cell, "bar": bar, "dash": orDash}

var markdownReport = template.Must(template.New("md").Funcs(reportFuncs).Parse(`# {{.Name}}
{{if .Description}}
{{.Description}}
{{end}}
**{{.Done}} of {{.Total}} repos done ({{.Percent}}%)** ` + "`{{bar .Percent}}`" + `

{{range $i, $c := .Counts}}{{if $i}} · {{end}}{{$c.N}} {{$c.Label}}{{end}}
{{- with .Attention}}

## Needs attention

| Repo | PR | Problem |
| --- | --- | --- |
{{range .}}| {{.Repo}} | {{if .URL}}[#{{.Number}}]({{.URL}}){{else}}-{{end}} | {{cell .Note}} |
{{end}}{{end}}
{{- with .Open}}
## Open

| Repo | PR | Checks | Review | Waiting on |
| --- | --- | --- | --- | --- |
{{range .}}| {{.Repo}} | [#{{.Number}}]({{.URL}}) | {{dash .Checks}} | {{dash .Review}} | {{cell .Note}} |
{{end}}{{end}}
{{- with .Merged}}
## Merged

{{range .}}- {{.Repo}} [#{{.Number}}]({{.URL}})
{{end}}{{end}}
{{- with .Other}}
<details><summary>{{len .}} other repos</summary>

| Repo | Status | Note |
| --- | --- | --- |
{{range .}}| {{.Repo}} | {{.Status}} | {{cell .Note}} |
{{end}}
</details>
{{end}}
_Generated {{.Generated}} by [gh fleet](https://github.com/rossedman/gh-fleet)._
`))

var htmlReport = htmltemplate.Must(htmltemplate.New("html").Funcs(htmltemplate.FuncMap{"dash": orDash}).Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Name}}</title>
<style>
:root { --bg: #fff; --fg: #1f2328; --dim: #656d76; --line: #d1d9e0; --ok: #1a7f37; --bad: #cf222e; --warn: #9a6700; --bar: #eaeef2; }
@media (prefers-color-scheme: dark) {
  :root { --bg: #0d1117; --fg: #e6edf3; --dim: #8d96a0; --line: #30363d; --ok: #3fb950; --bad: #f85149; --warn: #d29922; --bar: #21262d; }
}
body { background: var(--bg); color: var(--fg); font: 15px/1.5 -apple-system, "Segoe UI", Helvetica, Arial, sans-serif; max-width: 960px; margin: 0 auto; padding: 24px 16px; }
h1 { margin: 0 0 4px; font-size: 26px; }
h2 { margin: 32px 0 8px; font-size: 18px; }
p.desc, .dim, footer { color: var(--dim); }
.progress { display: flex; align-items: center; gap: 12px; margin: 16px 0 8px; }
.track { flex: 1; height: 10px; background: var(--bar); border-radius: 5px; overflow: hidden; }
.fill { height: 100%; background: var(--ok); }
table { border-collapse: collapse; width: 100%; font-size: 14px; }
th, td { text-align: left; padding: 6px 8px; border-bottom: 1px solid var(--line); vertical-align: top; }
th { color: var(--dim); font-weight: 600; }
.wrap { overflow-x: auto; }
a { color: inherit; }
.bad { color: var(--bad); }
.ok { color: var(--ok); }
.warn { color: var(--warn); }
footer { margin-top: 32px; font-size: 13px; }
</style>
</head>
<body>
<h1>{{.Name}}</h1>
{{if .Description}}<p class="desc">{{.Description}}</p>{{end}}
<div class="progress"><strong>{{.Done}} of {{.Total}} done ({{.Percent}}%)</strong><div class="track"><div class="fill" style="width: {{.Percent}}%"></div></div></div>
<p class="dim">{{range $i, $c := .Counts}}{{if $i}} · {{end}}{{$c.N}} {{$c.Label}}{{end}}</p>
{{with .Attention}}<h2 class="bad">Needs attention</h2>
<div class="wrap"><table><tr><th>Repo</th><th>PR</th><th>Problem</th></tr>
{{range .}}<tr><td>{{.Repo}}</td><td>{{if .URL}}<a href="{{.URL}}">#{{.Number}}</a>{{else}}-{{end}}</td><td class="bad">{{.Note}}</td></tr>
{{end}}</table></div>{{end}}
{{with .Open}}<h2>Open</h2>
<div class="wrap"><table><tr><th>Repo</th><th>PR</th><th>Checks</th><th>Review</th><th>Waiting on</th></tr>
{{range .}}<tr><td>{{.Repo}}</td><td><a href="{{.URL}}">#{{.Number}}</a></td><td class="{{if eq .Checks "success"}}ok{{else if eq .Checks "pending" "expected"}}warn{{end}}">{{dash .Checks}}</td><td>{{dash .Review}}</td><td>{{.Note}}</td></tr>
{{end}}</table></div>{{end}}
{{with .Merged}}<h2 class="ok">Merged</h2>
<div class="wrap"><table><tr><th>Repo</th><th>PR</th></tr>
{{range .}}<tr><td>{{.Repo}}</td><td><a href="{{.URL}}">#{{.Number}}</a></td></tr>
{{end}}</table></div>{{end}}
{{with .Other}}<details><summary><h2 style="display: inline">{{len .}} other repos</h2></summary>
<div class="wrap"><table><tr><th>Repo</th><th>Status</th><th>Note</th></tr>
{{range .}}<tr><td>{{.Repo}}</td><td>{{.Status}}</td><td class="dim">{{.Note}}</td></tr>
{{end}}</table></div></details>{{end}}
<footer>Generated {{.Generated}} by <a href="https://github.com/rossedman/gh-fleet">gh fleet</a>.</footer>
</body>
</html>
`))

// writeReport renders the report in format.
func writeReport(w io.Writer, format string, r report) error {
	switch format {
	case FormatMarkdown:
		return markdownReport.Execute(w, r)
	case FormatHTML:
		return htmlReport.Execute(w, r)
	}
	return fmt.Errorf("unknown report format %q", format)
}
