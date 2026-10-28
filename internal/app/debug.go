package app

import (
	"strings"

	"github.com/rossedman/gh-fleet/internal/timing"
	"github.com/rossedman/gh-fleet/internal/ui"
)

// EnableDebug turns on --debug: debug logs on stderr, and timings of every
// phase, repo stage, git command and GitHub request, which Summary prints
// at the end.
func (a *App) EnableDebug() {
	a.Log = ui.NewLogger(noteWriter{a}, true)
	a.Timings = timing.New(a.Log)
	a.Git.Timings = a.Timings
}

// TimingSummary writes the timings table when --debug is on.
func (a *App) TimingSummary() error {
	return a.Timings.Summary(a.Err)
}

// phase times one step of a command, logging when it starts so a long one
// shows what it is waiting on. Call the returned func when it is done.
func (a *App) phase(name string) func() {
	if a.Log != nil {
		a.Log.Debug(name + ": starting")
	}
	return a.Timings.Start(name)
}

// noteWriter writes log lines through note, so they print above the live
// progress line instead of through it.
type noteWriter struct{ a *App }

// Write prints each line of p as a note.
func (w noteWriter) Write(p []byte) (int, error) {
	for _, l := range strings.Split(strings.TrimRight(string(p), "\n"), "\n") {
		w.a.note(l)
	}
	return len(p), nil
}
