package render

import (
	"bytes"
	"strings"
	"testing"

	"github.com/rossedman/gh-fleet/internal/state"
	"github.com/rossedman/gh-fleet/internal/ui"
)

func TestTableAligns(t *testing.T) {
	t.Parallel()
	var b bytes.Buffer
	p := Printer{Out: &b, Styles: ui.Plain()}
	p.Table([]string{"REPO", "STATUS"}, [][]string{{"acme/api", "noop"}, {"a/b", "failed", "extra"}}, nil)
	want := "REPO      STATUS\nacme/api  noop\na/b       failed\n"
	if b.String() != want {
		t.Fatalf("table:\n%q\nwant:\n%q", b.String(), want)
	}
	b.Reset()
	p.Table([]string{"A"}, [][]string{{"x"}}, func(_, _ int, s string) string { return "[" + s + "]" })
	if !strings.Contains(b.String(), "[x]") {
		t.Fatal(b.String())
	}
}

func TestJSONAndJQ(t *testing.T) {
	t.Parallel()
	var b bytes.Buffer
	p := Printer{Out: &b, JSON: true}
	if !p.Machine() {
		t.Fatal("JSON not machine")
	}
	if err := p.PrintJSON(map[string]int{"a": 1}); err != nil || b.String() != "{\n  \"a\": 1\n}\n" {
		t.Fatalf("json = %q, %v", b.String(), err)
	}
	b.Reset()
	p = Printer{Out: &b, JQ: ".a"}
	if err := p.PrintJSON(map[string]int{"a": 1}); err != nil || b.String() != "1\n" {
		t.Fatalf("jq = %q, %v", b.String(), err)
	}
	if err := p.PrintJSON(func() {}); err == nil {
		t.Fatal("expected marshal error")
	}
	p.JQ = ".["
	if err := p.PrintJSON(1); err == nil {
		t.Fatal("expected jq error")
	}
}

func TestSummaryProgressAndDiff(t *testing.T) {
	t.Parallel()
	var b bytes.Buffer
	p := Printer{Out: &b, Styles: ui.Plain()}
	results := []state.RepoState{{Status: state.Failed}, {Status: state.Changed}, {Status: state.Changed}, {Status: state.Noop}, {Status: state.Skipped}, {Status: state.PROpen}}
	if got := p.Summary(Count(results)); got != "2 changed · 1 pr open · 1 noop · 1 failed · 1 skipped" {
		t.Fatal(got)
	}
	if p.Summary(nil) != "no repos" {
		t.Fatal("empty summary")
	}
	p.Progress(state.RepoState{Repo: "a/b", Status: state.Failed, LastError: "boom\nmore"})
	p.Progress(state.RepoState{Repo: "a/c", Status: state.PROpen, PRURL: "https://x/1", Detail: "created"})
	out := b.String()
	if !strings.Contains(out, "failed   a/b boom\n") || !strings.Contains(out, "pr open  a/c https://x/1 created") {
		t.Fatalf("progress:\n%s", out)
	}
	b.Reset()
	p.Diff("a/b", "diff --git a/x b/x\nindex 1..2\n--- a/x\n+++ b/x\n@@ -1 +1 @@\n-old\n+new\n ctx\n")
	if !strings.HasPrefix(b.String(), "── a/b\n") || !strings.Contains(b.String(), "+new") {
		t.Fatalf("diff:\n%s", b.String())
	}
	// Colored styles render every status.
	c := Printer{Out: &b, Styles: ui.Color()}
	for _, s := range []state.Status{state.Merged, state.PROpen, state.Failed, state.Skipped, state.Noop} {
		_ = c.StatusStyle(s).Render(string(s))
	}
}
