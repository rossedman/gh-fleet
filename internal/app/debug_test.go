package app

import (
	"context"
	"strings"
	"testing"
)

// timingNames returns the names the app's recorder has seen.
func timingNames(a *App) map[string]int {
	got := map[string]int{}
	for _, s := range a.Timings.Stats() {
		got[s.Name] = s.Count
	}
	return got
}

func TestDebugTimesPlanAndApply(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.app.EnableDebug()
	if h.app.Git.Timings != h.app.Timings || h.app.Log == nil {
		t.Fatal("EnableDebug didn't wire git timings and the logger")
	}
	f := h.campaign(basic, files)
	if err := h.app.Plan(context.Background(), PlanOptions{Common: Common{File: f}}); err != nil {
		t.Fatal(err)
	}
	got := timingNames(h.app)
	for name, n := range map[string]int{
		"load campaign": 1, "compile starlark": 1, "resolve targets": 1, "pick wave": 1,
		"run change in every repo": 1, "save state": 1,
		"repo: total": 2, "repo: clone": 2, "repo: step #1": 2, "repo: diff": 2, "repo: cleanup": 2,
	} {
		if got[name] != n {
			t.Errorf("%s recorded %d times, want %d (all: %v)", name, got[name], n, got)
		}
	}
	if got["git clone"] != 2 {
		t.Errorf("git clone recorded %d times", got["git clone"])
	}
	logs := h.errOut.String()
	if !strings.Contains(logs, "resolve targets: starting") || !strings.Contains(logs, "repo=acme/api") {
		t.Fatalf("debug logs:\n%s", logs)
	}

	h.answer = true
	if err := h.app.Apply(context.Background(), ApplyOptions{Common: Common{File: f, Yes: true}}); err != nil {
		t.Fatal(err)
	}
	got = timingNames(h.app)
	for _, name := range []string{"index existing PRs", "skip unchanged repos", "repo: push", "repo: sync PR"} {
		if got[name] == 0 {
			t.Errorf("%s not recorded (all: %v)", name, got)
		}
	}

	h.reset()
	if err := h.app.TimingSummary(); err != nil {
		t.Fatal(err)
	}
	if s := h.errOut.String(); !strings.Contains(s, "Timings") || !strings.Contains(s, "repo: clone") {
		t.Fatalf("summary:\n%s", s)
	}
}

func TestNoDebugRecordsNothing(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	if err := h.app.Plan(context.Background(), PlanOptions{Common: Common{File: h.campaign(basic, files)}}); err != nil {
		t.Fatal(err)
	}
	if h.app.Timings.Stats() != nil || strings.Contains(h.errOut.String(), "starting") {
		t.Fatalf("recorded without --debug:\n%s", h.errOut.String())
	}
	if err := h.app.TimingSummary(); err != nil {
		t.Fatal(err)
	}
}

func TestNoteWriterSplitsLines(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	w := noteWriter{h.app}
	n, err := w.Write([]byte("one\ntwo\n"))
	if err != nil || n != 8 || h.errOut.String() != "one\ntwo\n" {
		t.Fatalf("wrote %d, %v: %q", n, err, h.errOut.String())
	}
}
