package timing

import (
	"bytes"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"
)

// clock is a fake time source that moves only when told.
type clock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *clock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func TestNilRecorderDoesNothing(t *testing.T) {
	t.Parallel()
	var r *Recorder
	r.Start("x")()
	r.Add("x", time.Second)
	if r.Stats() != nil {
		t.Fatal("nil recorder has stats")
	}
	var buf bytes.Buffer
	if err := r.Summary(&buf); err != nil || buf.Len() != 0 {
		t.Fatalf("summary %q, %v", buf.String(), err)
	}
}

func TestStartAndAddAggregate(t *testing.T) {
	t.Parallel()
	c := &clock{now: time.Unix(0, 0)}
	r := New(nil)
	r.Now = c.Now
	stop := r.Start("clone")
	c.advance(3 * time.Second)
	stop()
	stop() // a second call records nothing
	r.Add("clone", time.Second)
	r.Add("diff", 500*time.Microsecond)

	got := r.Stats()
	if len(got) != 2 || got[0].Name != "clone" || got[1].Name != "diff" {
		t.Fatalf("stats = %+v", got)
	}
	cl := got[0]
	if cl.Count != 2 || cl.Total != 4*time.Second || cl.Max != 3*time.Second || cl.Avg() != 2*time.Second {
		t.Fatalf("clone = %+v avg %s", cl, cl.Avg())
	}
	if (Stat{}).Avg() != 0 {
		t.Fatal("empty stat has an average")
	}
}

func TestDefaultClock(t *testing.T) {
	t.Parallel()
	r := New(nil)
	stop := r.Start("x")
	stop()
	if s := r.Stats(); len(s) != 1 || s[0].Total < 0 {
		t.Fatalf("stats = %+v", s)
	}
}

func TestAddLogs(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	r := New(log)
	r.Add("git clone", 1500*time.Millisecond, "repo", "acme/api")
	out := buf.String()
	for _, want := range []string{"level=DEBUG", `msg="git clone"`, "repo=acme/api", "took=1.5s"} {
		if !strings.Contains(out, want) {
			t.Fatalf("log %q lacks %q", out, want)
		}
	}
}

func TestSummaryOrdersByTotal(t *testing.T) {
	t.Parallel()
	r := New(nil)
	r.Add("small", time.Millisecond)
	r.Add("big", 2*time.Second)
	r.Add("big", time.Second)
	var buf bytes.Buffer
	if err := r.Summary(&buf); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 4 || !strings.HasPrefix(lines[0], "Timings") || !strings.HasPrefix(lines[1], "NAME") {
		t.Fatalf("summary:\n%s", buf.String())
	}
	if f := strings.Fields(lines[2]); len(f) != 5 || f[0] != "big" || f[1] != "2" || f[2] != "3s" || f[3] != "1.5s" || f[4] != "2s" {
		t.Fatalf("big row = %q", lines[2])
	}
	if f := strings.Fields(lines[3]); f[0] != "small" || f[2] != "1ms" {
		t.Fatalf("small row = %q", lines[3])
	}
}

func TestRound(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		in, want time.Duration
	}{
		{1234567 * time.Nanosecond, time.Millisecond},
		{time.Millisecond, time.Millisecond},
		{999499 * time.Nanosecond, 999 * time.Microsecond},
		{1500 * time.Nanosecond, 2 * time.Microsecond},
	} {
		if got := round(tc.in); got != tc.want {
			t.Errorf("round(%s) = %s, want %s", tc.in, got, tc.want)
		}
	}
}

func TestConcurrentAdds(t *testing.T) {
	t.Parallel()
	r := New(nil)
	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() { r.Add("x", time.Millisecond) })
	}
	wg.Wait()
	if s := r.Stats(); s[0].Count != 50 || s[0].Total != 50*time.Millisecond {
		t.Fatalf("stats = %+v", s)
	}
}
