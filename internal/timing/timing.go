// Package timing measures how long gh fleet spends where, for --debug: each
// phase of a command, each stage of each repo, each git command and each
// GitHub request. A nil *Recorder records nothing, so callers time
// unconditionally and pay nothing without --debug.
package timing

import (
	"fmt"
	"io"
	"log/slog"
	"sort"
	"sync"
	"text/tabwriter"
	"time"
)

// Recorder collects durations by name. It is safe for concurrent use.
type Recorder struct {
	// Log, when set, gets a debug line for every duration recorded.
	Log *slog.Logger
	// Now returns the current time; defaults to time.Now.
	Now func() time.Time

	mu    sync.Mutex
	stats map[string]*Stat
	order []string
}

// Stat is every duration recorded under one name.
type Stat struct {
	Name  string
	Count int
	Total time.Duration
	Max   time.Duration
}

// Avg is the mean duration.
func (s Stat) Avg() time.Duration {
	if s.Count == 0 {
		return 0
	}
	return s.Total / time.Duration(s.Count)
}

// New returns a Recorder logging to log, which may be nil.
func New(log *slog.Logger) *Recorder {
	return &Recorder{Log: log}
}

// now returns the recorder's clock.
func (r *Recorder) now() time.Time {
	if r.Now != nil {
		return r.Now()
	}
	return time.Now()
}

// Start starts timing name. The returned func stops the timer and records
// the elapsed time with attrs, which are slog key-value pairs for the log
// line. Calling it more than once records once.
func (r *Recorder) Start(name string, attrs ...any) func() {
	if r == nil {
		return func() {}
	}
	start := r.now()
	var once sync.Once
	return func() { once.Do(func() { r.Add(name, r.now().Sub(start), attrs...) }) }
}

// Add records d under name.
func (r *Recorder) Add(name string, d time.Duration, attrs ...any) {
	if r == nil {
		return
	}
	r.mu.Lock()
	if r.stats == nil {
		r.stats = map[string]*Stat{}
	}
	s, ok := r.stats[name]
	if !ok {
		s = &Stat{Name: name}
		r.stats[name] = s
		r.order = append(r.order, name)
	}
	s.Count++
	s.Total += d
	s.Max = max(s.Max, d)
	r.mu.Unlock()
	if r.Log != nil {
		r.Log.Debug(name, append(attrs, "took", round(d))...)
	}
}

// Stats returns every name's durations, in the order each name was first
// recorded.
func (r *Recorder) Stats() []Stat {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Stat, len(r.order))
	for i, n := range r.order {
		out[i] = *r.stats[n]
	}
	return out
}

// Summary writes a table of every name's count, total, mean and longest
// duration, largest total first. Durations recorded by concurrent workers
// add up, so a total can exceed the wall time.
func (r *Recorder) Summary(w io.Writer) error {
	stats := r.Stats()
	if len(stats) == 0 {
		return nil
	}
	sort.SliceStable(stats, func(i, j int) bool { return stats[i].Total > stats[j].Total })
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "\nTimings (totals add up across parallel workers):")
	fmt.Fprintln(tw, "NAME\tCOUNT\tTOTAL\tAVG\tMAX")
	for _, s := range stats {
		fmt.Fprintf(tw, "%s\t%d\t%s\t%s\t%s\n", s.Name, s.Count, round(s.Total), round(s.Avg()), round(s.Max))
	}
	return tw.Flush()
}

// round shortens d for reading: whole milliseconds, or microseconds below
// a millisecond.
func round(d time.Duration) time.Duration {
	if d < time.Millisecond {
		return d.Round(time.Microsecond)
	}
	return d.Round(time.Millisecond)
}
