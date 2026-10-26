package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/russross/blackfriday/v2"

	"github.com/rossedman/gh-fleet/internal/app"
	"github.com/rossedman/gh-fleet/internal/campaign"
	"github.com/rossedman/gh-fleet/internal/state"
)

// reportCampaign and reportRows are the campaign the docs' example
// progress report describes: a repo in every state a report groups.
var (
	reportCampaign = &campaign.Campaign{Name: "pin-actions", Description: "Pin every third-party GitHub Action to a full commit SHA"}
	reportTime     = time.Date(2026, 10, 26, 14, 0, 0, 0, time.UTC)
)

// reportRows returns the example report's status rows.
func reportRows() []state.RepoState {
	pr := func(repo string, n int, checks, review, merge string) state.RepoState {
		return state.RepoState{
			Repo: "acme/" + repo, Status: state.PROpen, PRNumber: n,
			PRURL:  fmt.Sprintf("https://github.com/acme/%s/pull/%d", repo, n),
			Checks: checks, Review: review, MergeState: merge,
		}
	}
	merged := func(repo string, n int) state.RepoState {
		r := pr(repo, n, "SUCCESS", "APPROVED", "")
		r.Status = state.Merged
		return r
	}
	return []state.RepoState{
		merged("api", 41),
		merged("billing", 17),
		merged("web", 88),
		pr("gateway", 23, "SUCCESS", "APPROVED", "CLEAN"),
		pr("ledger", 7, "SUCCESS", "APPROVED", "DIRTY"),
		pr("notify", 12, "SUCCESS", "REVIEW_REQUIRED", "BLOCKED"),
		pr("payments", 54, "SUCCESS", "CHANGES_REQUESTED", "BLOCKED"),
		pr("search", 9, "PENDING", "REVIEW_REQUIRED", "BLOCKED"),
		pr("worker", 31, "FAILURE", "APPROVED", "BLOCKED"),
		{Repo: "acme/legacy-api", Status: state.Failed, LastError: "step resolve: gh api repos/acme/internal-action/commits/v2: HTTP 404"},
		{Repo: "acme/cli", Status: state.Noop, Detail: "no unpinned actions"},
		{Repo: "acme/docs-site", Status: state.Noop, Detail: "no unpinned actions"},
		{Repo: "acme/archive", Status: state.Skipped, Detail: "no push access"},
	}
}

// previewPage wraps rendered Markdown, in place of {{body}}, in a page
// styled like a GitHub comment, in light and dark.
const previewPage = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>pin-actions report, rendered</title>
<style>
:root { --bg: #fff; --fg: #1f2328; --dim: #59636e; --line: #d1d9e0; --alt: #f6f8fa; --link: #0969da; }
@media (prefers-color-scheme: dark) {
  :root { --bg: #0d1117; --fg: #f0f6fc; --dim: #9198a1; --line: #3d444d; --alt: #151b23; --link: #4493f8; }
}
body { background: var(--bg); color: var(--fg); font: 14px/1.5 -apple-system, "Segoe UI", "Noto Sans", Helvetica, Arial, sans-serif; margin: 0; padding: 16px 24px; }
h1 { font-size: 2em; border-bottom: 1px solid var(--line); padding-bottom: .3em; margin: 0 0 16px; }
h2 { font-size: 1.5em; border-bottom: 1px solid var(--line); padding-bottom: .3em; margin: 24px 0 16px; }
p, ul, table, details { margin: 0 0 16px; }
a { color: var(--link); text-decoration: none; }
code { font: 85% ui-monospace, SFMono-Regular, Menlo, monospace; background: var(--alt); padding: .2em .4em; border-radius: 6px; }
table { border-collapse: collapse; display: block; overflow: auto; }
th, td { border: 1px solid var(--line); padding: 6px 13px; }
th { font-weight: 600; }
tr:nth-child(2n) { background: var(--alt); }
em { color: var(--dim); }
summary { cursor: pointer; }
</style>
</head>
<body>
{{body}}</body>
</html>
`

// formatComment renders the Markdown report the way an issue comment shows
// it.
const formatComment = "comment"

// reportFiles names each example report file and its format.
var reportFiles = map[string]string{
	"report.md.txt":       app.FormatMarkdown,
	"report.html":         app.FormatHTML,
	"report-comment.html": formatComment,
}

// renderReport renders the example report in format: markdown or html as
// status --format prints it, or comment for the Markdown rendered.
func renderReport(format string) ([]byte, error) {
	f := format
	if format == formatComment {
		f = app.FormatMarkdown
	}
	var b bytes.Buffer
	if err := app.WriteReport(&b, f, reportCampaign, reportRows(), reportTime); err != nil {
		return nil, err
	}
	if format == formatComment {
		html := blackfriday.Run(b.Bytes(), blackfriday.WithExtensions(blackfriday.CommonExtensions))
		return []byte(strings.Replace(previewPage, "{{body}}", string(html), 1)), nil
	}
	return b.Bytes(), nil
}

// writeReportExamples writes the example progress report to out/report,
// one file per entry of files, which maps file names to formats.
func writeReportExamples(out string, files map[string]string) error {
	dir := filepath.Join(out, "report")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for name, format := range files {
		b, err := renderReport(format)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), b, 0o644); err != nil { //nolint:gosec // docs are public
			return err
		}
	}
	return nil
}
