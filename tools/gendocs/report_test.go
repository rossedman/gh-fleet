package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteReportExamples(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := writeReportExamples(dir, reportFiles); err != nil {
		t.Fatal(err)
	}
	read := func(name string) string {
		b, err := os.ReadFile(filepath.Join(dir, "report", name))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	md := read("report.md.txt")
	for _, want := range []string{"# pin-actions\n", "**5 of 12 repos done (41%)**", "| acme/ledger | [#7](https://github.com/acme/ledger/pull/7) | merge conflict |", "_Generated 2026-10-26 14:00 UTC"} {
		if !strings.Contains(md, want) {
			t.Errorf("markdown missing %q:\n%s", want, md)
		}
	}
	comment := read("report-comment.html")
	for _, want := range []string{"<h1>pin-actions</h1>", "<table>", `<a href="https://github.com/acme/worker/pull/31">#31</a>`, "prefers-color-scheme: dark"} {
		if !strings.Contains(comment, want) {
			t.Errorf("rendered comment missing %q", want)
		}
	}
	if strings.Contains(comment, "{{body}}") {
		t.Error("placeholder left in the rendered comment")
	}
	if html := read("report.html"); !strings.Contains(html, "5 of 12 done (41%)") || !strings.Contains(html, `style="width: 41%"`) {
		t.Errorf("html report:\n%s", html)
	}

	// out is a file, so the report directory can't be made.
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeReportExamples(file, reportFiles); err == nil {
		t.Error("unwritable output accepted")
	}
	if err := writeReportExamples(t.TempDir(), map[string]string{"x.pdf": "pdf"}); err == nil || !strings.Contains(err.Error(), "x.pdf") {
		t.Errorf("unknown format: %v", err)
	}
	// A directory where a report file goes can't be written over.
	blocked := t.TempDir()
	if err := os.MkdirAll(filepath.Join(blocked, "report", "report.html"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeReportExamples(blocked, map[string]string{"report.html": "html"}); err == nil {
		t.Error("a directory in the way was overwritten")
	}
}
