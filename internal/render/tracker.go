package render

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rossedman/gh-fleet/internal/state"
)

// Tracker reports a run's progress on stderr while repos are processed.
// On a terminal it keeps one status line at the bottom, redrawn as repos
// finish and every second; elsewhere, such as a CI log, it prints a
// heartbeat line every Interval so a long run is never silent.
type Tracker struct {
	// Printer writes the lines; its Out is stderr.
	Printer Printer
	// Live redraws a status line in place; set it when Out is a terminal.
	Live bool
	// Quiet prints only the status line, not a line per repo, for
	// machine output.
	Quiet bool
	// Verb names the run, such as "Planning".
	Verb string
	// Total is the number of repos.
	Total int
	// Interval is how often a heartbeat prints without Live (default 30s)
	// and the status line redraws with it (default 1s).
	Interval time.Duration
	// Now returns the current time. Defaults to time.Now.
	Now func() time.Time

	mu      sync.Mutex
	start   time.Time
	done    int
	failed  int
	drawn   bool
	stop    chan struct{}
	stopped sync.WaitGroup
}

// Start prints what is about to happen and starts the ticker.
func (t *Tracker) Start(concurrency int) {
	t.mu.Lock()
	t.start = t.now()
	t.stop = make(chan struct{})
	t.mu.Unlock()
	if !t.Quiet {
		fmt.Fprintf(t.Printer.Out, "%s %s, %d at a time\n", t.Verb, plural(t.Total, "repo"), concurrency)
	}
	interval := t.Interval
	if interval <= 0 {
		interval = 30 * time.Second
		if t.Live {
			interval = time.Second
		}
	}
	t.stopped.Add(1)
	go func() {
		defer t.stopped.Done()
		tick := time.NewTicker(interval)
		defer tick.Stop()
		for {
			select {
			case <-t.stop:
				return
			case <-tick.C:
				t.tick()
			}
		}
	}()
	t.mu.Lock()
	t.redraw()
	t.mu.Unlock()
}

// tick redraws the status line, or prints a heartbeat.
func (t *Tracker) tick() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.Live {
		t.redraw()
		return
	}
	if t.done < t.Total {
		fmt.Fprintln(t.Printer.Out, t.Printer.Styles.Dim.Render(t.status()))
	}
}

// Result records a finished repo and prints its line above the status.
func (t *Tracker) Result(rs state.RepoState) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.done++
	if rs.Status == state.Failed {
		t.failed++
	}
	if !t.Quiet {
		t.clear()
		t.Printer.Progress(rs)
	}
	t.redraw()
}

// Note prints a line, such as a rate limit wait, above the status line.
func (t *Tracker) Note(msg string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.clear()
	fmt.Fprintln(t.Printer.Out, msg)
	t.redraw()
}

// Pause clears the status line while f writes, such as a diff to stdout
// on the same terminal, then draws it again.
func (t *Tracker) Pause(f func()) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.clear()
	f()
	t.redraw()
}

// Stop stops the ticker and clears the status line.
func (t *Tracker) Stop() {
	if t.stop == nil {
		return
	}
	close(t.stop)
	t.stopped.Wait()
	t.mu.Lock()
	defer t.mu.Unlock()
	t.clear()
	t.stop = nil
}

// status describes progress: done of total, failures, elapsed time and,
// once a few repos are done, an estimate of what's left.
func (t *Tracker) status() string {
	elapsed := t.now().Sub(t.start).Round(time.Second)
	pct := 0
	if t.Total > 0 {
		pct = t.done * 100 / t.Total
	}
	parts := []string{fmt.Sprintf("%d/%d repos (%d%%)", t.done, t.Total, pct)}
	if t.failed > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", t.failed))
	}
	parts = append(parts, elapsed.String()+" elapsed")
	if t.done >= 3 && t.done < t.Total {
		left := time.Duration(float64(t.now().Sub(t.start)) / float64(t.done) * float64(t.Total-t.done)).Round(time.Second)
		parts = append(parts, "about "+left.String()+" left")
	}
	return t.Verb + " " + strings.Join(parts, " · ")
}

// redraw draws the status line on a terminal. The caller holds mu.
func (t *Tracker) redraw() {
	if !t.Live || t.stop == nil {
		return
	}
	fmt.Fprint(t.Printer.Out, "\r\x1b[K"+t.Printer.Styles.Dim.Render(t.status()))
	t.drawn = true
}

// clear erases the status line. The caller holds mu.
func (t *Tracker) clear() {
	if t.drawn {
		fmt.Fprint(t.Printer.Out, "\r\x1b[K")
		t.drawn = false
	}
}

// now returns the current time from the injected clock.
func (t *Tracker) now() time.Time {
	if t.Now != nil {
		return t.Now()
	}
	return time.Now()
}

// plural formats n with a noun, adding s unless n is 1.
func plural(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return fmt.Sprintf("%d %ss", n, noun)
}
