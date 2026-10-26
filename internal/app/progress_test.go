package app

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/rossedman/gh-fleet/internal/ghapi"
)

func TestRunProgress(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	f := h.campaign(basic, files)
	ctx := context.Background()
	if err := h.app.Plan(ctx, PlanOptions{Common: Common{File: f}}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Listing acme's repos\n", "Listed acme: 4 repos\n", "Planning 2 repos, 8 at a time\n", "changed  acme/api\n"} {
		if !strings.Contains(h.errOut.String(), want) {
			t.Errorf("plan progress missing %q:\n%s", want, h.errOut)
		}
	}

	// Machine output with nobody at the terminal stays quiet on stderr.
	h.reset()
	if err := h.app.Plan(ctx, PlanOptions{Common: Common{File: f, JSON: true}}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(h.errOut.String(), "Listed") || strings.Contains(h.errOut.String(), "acme/api") {
		t.Errorf("--json progress:\n%s", h.errOut)
	}

	// On a terminal, a status line counts repos as they finish.
	h.reset()
	h.app.ErrTTY = true
	if err := h.app.Apply(ctx, ApplyOptions{Common: Common{File: f, Yes: true}}); err != nil {
		t.Fatal(err)
	}
	if out := h.errOut.String(); !strings.Contains(out, "Looking up existing PRs in 2 repos\n") || !strings.Contains(out, "\r\x1b[KApplying to 2/2 repos (100%)") {
		t.Errorf("apply progress:\n%q", out)
	}
}

func TestRateLimitNotes(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.app.rateLimitWait("secondary rate limit", 61500*time.Millisecond)
	if got := h.errOut.String(); got != "GitHub secondary rate limit: waiting 1m2s before the next request\n" {
		t.Errorf("note without a run = %q", got)
	}
	// During a run the note goes above the status line.
	h.reset()
	h.app.ErrTTY = true
	tr, stop := h.app.track(Common{}, "Planning", 3, 1, true)
	h.app.note("waiting")
	stop()
	if got := h.errOut.String(); !strings.Contains(got, "\r\x1b[Kwaiting\n\r\x1b[KPlanning 0/3") || tr == nil {
		t.Errorf("note during a run = %q", got)
	}
}

func TestDefaultReportsRateLimitWaits(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Setenv("GH_TOKEN", "x")
	t.Setenv("GH_HOST", "")
	var out, errOut strings.Builder
	a, err := Default(&out, &errOut, strings.NewReader(""), false, false, false, false)
	if err != nil {
		t.Fatal(err)
	}
	c, err := a.NewClient(1)
	if err != nil {
		t.Fatal(err)
	}
	gh := c.(*ghapi.GitHub)
	gh.Transport.OnWait("search rate limit", time.Second)
	if !strings.Contains(errOut.String(), "GitHub search rate limit: waiting 1s") {
		t.Errorf("stderr = %q", errOut.String())
	}
}
