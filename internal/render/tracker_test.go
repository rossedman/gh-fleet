package render

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rossedman/gh-fleet/internal/state"
	"github.com/rossedman/gh-fleet/internal/ui"
)

// lockedBuffer is a bytes.Buffer safe for the tracker's ticker goroutine.
type lockedBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

// Write appends p.
func (l *lockedBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

// String returns what was written.
func (l *lockedBuffer) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.String()
}

// clock is a settable time source.
type clock struct{ t time.Time }

// now returns the clock's time.
func (c *clock) now() time.Time { return c.t }

func TestTrackerHeartbeat(t *testing.T) {
	t.Parallel()
	var out lockedBuffer
	c := &clock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	tr := &Tracker{Printer: Printer{Out: &out, Styles: ui.Plain()}, Verb: "Planning", Total: 10, Interval: time.Hour, Now: c.now}
	tr.Start(4)
	tr.Result(state.RepoState{Repo: "acme/a", Status: state.Changed})
	tr.Result(state.RepoState{Repo: "acme/b", Status: state.Failed, LastError: "boom"})
	c.t = c.t.Add(30 * time.Second)
	tr.tick() // fewer than 3 done: no estimate yet
	tr.Result(state.RepoState{Repo: "acme/c", Status: state.Noop})
	c.t = c.t.Add(30 * time.Second)
	tr.tick()
	tr.Note("GitHub secondary rate limit: waiting 1m0s")
	tr.Stop()
	tr.Stop() // a second stop is harmless
	got := out.String()
	for _, want := range []string{
		"Planning 10 repos, 4 at a time\n",
		"changed  acme/a\n",
		"failed   acme/b boom\n",
		"Planning 2/10 repos (20%) · 1 failed · 30s elapsed\n",
		"Planning 3/10 repos (30%) · 1 failed · 1m0s elapsed · about 2m20s left\n",
		"GitHub secondary rate limit: waiting 1m0s\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "\r") {
		t.Errorf("heartbeat mode drew a live line:\n%q", got)
	}

	// Nothing more prints once every repo is done.
	out = lockedBuffer{}
	done := &Tracker{Printer: Printer{Out: &out, Styles: ui.Plain()}, Verb: "Planning", Total: 1, Interval: time.Hour}
	done.Start(1)
	done.Result(state.RepoState{Repo: "acme/a", Status: state.Noop})
	done.tick()
	done.Stop()
	if strings.Contains(out.String(), "1/1") {
		t.Errorf("heartbeat after the last repo:\n%s", out.String())
	}
}

func TestTrackerLive(t *testing.T) {
	t.Parallel()
	var out lockedBuffer
	tr := &Tracker{Printer: Printer{Out: &out, Styles: ui.Plain()}, Live: true, Verb: "Applying to", Total: 2, Interval: time.Hour}
	tr.Start(8)
	tr.Result(state.RepoState{Repo: "acme/a", Status: state.PROpen, PRURL: "https://x/1"})
	tr.Pause(func() { out.Write([]byte("diff\n")) })
	tr.tick()
	tr.Stop()
	got := out.String()
	for _, want := range []string{
		"Applying to 2 repos, 8 at a time\n\r\x1b[KApplying to 0/2 repos (0%) · 0s elapsed",
		"\r\x1b[Kpr open  acme/a https://x/1\n\r\x1b[KApplying to 1/2 repos (50%)",
		"\r\x1b[Kdiff\n\r\x1b[KApplying to 1/2",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%q", want, got)
		}
	}
	if !strings.HasSuffix(got, "\r\x1b[K") {
		t.Errorf("status line left behind:\n%q", got)
	}

	// Quiet counts repos without listing them, and prints no header.
	out = lockedBuffer{}
	q := &Tracker{Printer: Printer{Out: &out, Styles: ui.Plain()}, Live: true, Quiet: true, Verb: "Running in", Total: 1}
	q.Start(1)
	q.Result(state.RepoState{Repo: "acme/a", Status: state.Noop})
	q.Stop()
	if got := out.String(); strings.Contains(got, "acme/a") || strings.Contains(got, "at a time") || !strings.Contains(got, "Running in 1/1 repos (100%)") {
		t.Errorf("quiet tracker:\n%q", got)
	}
	(&Tracker{}).Stop() // never started
	if (&Tracker{}).now().IsZero() {
		t.Error("default clock")
	}
}
