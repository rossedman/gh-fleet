package app

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/rossedman/gh-fleet/internal/campaign"
	"github.com/rossedman/gh-fleet/internal/state"
)

// reportRows is one repo in every state a report groups.
var reportRows = []state.RepoState{
	{Repo: "acme/a", Status: state.Merged, PRNumber: 1, PRURL: "https://github.com/acme/a/pull/1"},
	{Repo: "acme/b", Status: state.PROpen, PRNumber: 2, PRURL: "https://github.com/acme/b/pull/2", Checks: "SUCCESS", Review: "APPROVED", MergeState: "CLEAN"},
	{Repo: "acme/c", Status: state.PROpen, PRNumber: 3, PRURL: "https://github.com/acme/c/pull/3", Checks: "PENDING", Review: "REVIEW_REQUIRED", MergeState: "BLOCKED"},
	{Repo: "acme/d", Status: state.PROpen, PRNumber: 4, PRURL: "https://github.com/acme/d/pull/4", Checks: "FAILURE", Review: "APPROVED", MergeState: "BLOCKED"},
	{Repo: "acme/e", Status: state.PROpen, PRNumber: 5, PRURL: "https://github.com/acme/e/pull/5", Checks: "SUCCESS", Review: "APPROVED", MergeState: "DIRTY"},
	{Repo: "acme/f", Status: state.Failed, LastError: "step lock: exit 1 | bundler\ntrace"},
	{Repo: "acme/g", Status: state.Noop, Detail: "already up to date"},
	{Repo: "acme/h", Status: state.Skipped, Detail: "archived"},
}

func TestReport(t *testing.T) {
	t.Parallel()
	camp := &campaign.Campaign{Name: "rails-7-2", Description: "Move every Rails app to 7.2"}
	r := newReport(camp, reportRows, time.Date(2026, 10, 26, 2, 40, 0, 0, time.UTC))
	if r.Total != 7 || r.Done != 2 || r.Percent != 28 || len(r.Attention) != 3 || len(r.Open) != 2 || len(r.Merged) != 1 || len(r.Other) != 2 {
		t.Fatalf("report = %+v", r)
	}
	var md bytes.Buffer
	if err := writeReport(&md, FormatMarkdown, r); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"# rails-7-2\n\nMove every Rails app to 7.2\n",
		"**2 of 7 repos done (28%)** `██░░░░░░░░`",
		"1 merged · 4 pr open · 1 failed · 1 noop · 1 skipped",
		"## Needs attention\n\n| Repo | PR | Problem |\n| --- | --- | --- |\n| acme/d | [#4](https://github.com/acme/d/pull/4) | checks failing |\n| acme/e | [#5](https://github.com/acme/e/pull/5) | merge conflict |\n| acme/f | - | step lock: exit 1 \\| bundler |\n",
		"| acme/b | [#2](https://github.com/acme/b/pull/2) | success | approved | ready to merge |",
		"| acme/c | [#3](https://github.com/acme/c/pull/3) | pending | review required | checks running |",
		"## Merged\n\n- acme/a [#1](https://github.com/acme/a/pull/1)\n",
		"<details><summary>2 other repos</summary>",
		"| acme/g | noop | already up to date |",
		"_Generated 2026-10-26 02:40 UTC by [gh fleet]",
	} {
		if !strings.Contains(md.String(), want) {
			t.Errorf("markdown missing %q:\n%s", want, md.String())
		}
	}

	var html bytes.Buffer
	camp.Description = "<script>x</script>"
	if err := writeReport(&html, FormatHTML, newReport(camp, reportRows, time.Now())); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"<title>rails-7-2</title>", "2 of 7 done (28%)", `style="width: 28%"`, `<a href="https://github.com/acme/d/pull/4">#4</a>`, `<td class="bad">merge conflict</td>`, "&lt;script&gt;", "prefers-color-scheme: dark"} {
		if !strings.Contains(html.String(), want) {
			t.Errorf("html missing %q", want)
		}
	}
	if strings.Contains(html.String(), "<script>x") {
		t.Error("description not escaped")
	}
	if err := writeReport(&html, "pdf", r); err == nil {
		t.Error("unknown format accepted")
	}

	// An empty campaign still renders.
	md.Reset()
	if err := writeReport(&md, FormatMarkdown, newReport(&campaign.Campaign{Name: "x"}, nil, time.Now())); err != nil || !strings.Contains(md.String(), "**0 of 0 repos done (0%)**") {
		t.Errorf("empty: %v\n%s", err, md.String())
	}
}

func TestStatusFormats(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	f := h.campaign(basic, files)
	ctx := context.Background()
	h.openPRs(f)
	h.gh.SetMergeState("acme/web", 1, "DIRTY")
	h.reset()
	if err := h.app.Status(ctx, StatusOptions{Common: Common{File: f}, Format: FormatMarkdown}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.out.String(), "# add-line") || !strings.Contains(h.out.String(), "| acme/api | [#1](") || !strings.Contains(h.out.String(), "ready to merge") || !strings.Contains(h.out.String(), "merge conflict") {
		t.Fatalf("markdown:\n%s", h.out)
	}
	st, _ := h.app.Store.Load("add-line")
	if st.Get("acme/web").MergeState != "DIRTY" {
		t.Errorf("merge state not recorded: %+v", st.Get("acme/web"))
	}
	h.reset()
	if err := h.app.Status(ctx, StatusOptions{Common: Common{File: f}, Format: FormatHTML}); err != nil || !strings.HasPrefix(h.out.String(), "<!doctype html>") {
		t.Fatalf("html: %v %.100s", err, h.out)
	}
	for _, o := range []StatusOptions{{Common: Common{File: f}, Format: "pdf"}, {Common: Common{File: f, JSON: true}, Format: FormatMarkdown}} {
		if err := h.app.Status(ctx, o); err == nil {
			t.Errorf("%+v accepted", o)
		}
	}
}
