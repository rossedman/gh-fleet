package app

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/rossedman/gh-fleet/internal/state"
)

func TestPipedTargets(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.app.TTY = false
	f := h.campaign(basic, files)
	if err := h.app.Targets(context.Background(), Common{File: f}); err != nil {
		t.Fatal(err)
	}
	// Only the targets reach stdout, tab-separated with no header.
	if want := "acme/api\ttopic:service\nacme/web\ttopic:service\n"; h.out.String() != want {
		t.Fatalf("stdout = %q, want %q", h.out, want)
	}
	if !containsAll(h.errOut.String(), "acme/docs", "no push access", "2 repos (1 skipped)") {
		t.Fatalf("stderr = %q", h.errOut)
	}
}

// containsAll reports whether s has every one of want.
func containsAll(s string, want ...string) bool {
	for _, w := range want {
		if !strings.Contains(s, w) {
			return false
		}
	}
	return true
}

func TestExitStatus(t *testing.T) {
	t.Parallel()
	rows := func(ss ...state.Status) []state.RepoState {
		out := make([]state.RepoState, len(ss))
		for i, s := range ss {
			out[i] = state.RepoState{Status: s}
		}
		return out
	}
	if err := exitStatus(rows(state.Merged, state.Closed, state.Noop, state.Skipped, state.Done)); err != nil {
		t.Errorf("all done: %v", err)
	}
	if err := exitStatus(rows(state.Merged, state.PROpen)); !errors.Is(err, ErrPending) {
		t.Errorf("open PR: %v", err)
	}
	if err := exitStatus(rows(state.Pending, state.Failed, state.Failed)); !errors.Is(err, ErrFailed) || err.Error() != "2 repos failed" {
		t.Errorf("failed: %v", err)
	}

	h := newHarness(t)
	f := h.campaign(basic, files)
	if err := h.app.Status(context.Background(), StatusOptions{Common: Common{File: f}, ExitStatus: true}); !errors.Is(err, ErrPending) {
		t.Errorf("status --exit-status before apply: %v", err)
	}
}
