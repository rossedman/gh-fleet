// Package app implements each gh fleet command on top of the campaign,
// target, engine, pr and state packages. Commands in cmd/ only parse
// flags and call into here, and everything here talks to GitHub through
// ghapi.Client, so it is tested with the in-memory fake.
package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rossedman/gh-fleet/internal/campaign"
	"github.com/rossedman/gh-fleet/internal/ghapi"
	"github.com/rossedman/gh-fleet/internal/gitops"
	"github.com/rossedman/gh-fleet/internal/render"
	"github.com/rossedman/gh-fleet/internal/star"
	"github.com/rossedman/gh-fleet/internal/state"
	"github.com/rossedman/gh-fleet/internal/target"
	"github.com/rossedman/gh-fleet/internal/timing"
	"github.com/rossedman/gh-fleet/internal/ui"
	"github.com/rossedman/gh-fleet/internal/workspace"
)

// App holds everything commands need from the outside world.
type App struct {
	Out io.Writer
	Err io.Writer
	In  io.Reader
	// Interactive is true when stdin and stdout are terminals.
	Interactive bool
	// Color enables styled output for people.
	Color bool
	// TTY is true when stdout is a terminal (or GH_FORCE_TTY is set).
	// Without it tables are tab-separated with no header row.
	TTY bool
	// ErrTTY is true when stderr is a terminal, where progress is a live
	// status line rather than periodic heartbeat lines.
	ErrTTY bool

	// trackMu guards tracker, the progress of the run in flight.
	trackMu sync.Mutex
	tracker *render.Tracker
	// progressEvery overrides how often progress redraws or prints, for
	// tests.
	progressEvery time.Duration
	// NewClient builds a GitHub client with the given write rate.
	NewClient  func(writeRate float64) (ghapi.Client, error)
	Store      *state.Store
	Cache      *target.Cache
	Workspaces *workspace.Manager
	Git        gitops.Git
	// Confirm asks a yes/no question; only called when Interactive.
	Confirm func(title string) (bool, error)
	// Version is the gh fleet version. A new version can change what a
	// change does, so it is part of the change's fingerprint.
	Version string
	// LibCache holds Starlark libraries loaded from other repositories.
	LibCache string
	// LibURL overrides the clone URL of those libraries (for tests).
	LibURL func(host, owner, repo string) string
	// Log gets debug logs; nil without --debug.
	Log *slog.Logger
	// Timings records how long things take; nil without --debug.
	Timings *timing.Recorder
}

// Default wires the real GitHub client, state dir, cache and git.
func Default(out, errOut io.Writer, in io.Reader, tty, errTTY, interactive, color bool) (*App, error) {
	stateDir, err := state.DefaultDir()
	if err != nil {
		return nil, err
	}
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir = os.TempDir()
	}
	a := &App{
		Out: out, Err: errOut, In: in, TTY: tty, ErrTTY: errTTY, Interactive: interactive, Color: color,
		Store:      &state.Store{Dir: stateDir},
		Cache:      &target.Cache{Dir: filepath.Join(cacheDir, "gh-fleet", "repos")},
		Workspaces: &workspace.Manager{},
		LibCache:   filepath.Join(cacheDir, "gh-fleet", "starlark"),
		Confirm: func(title string) (bool, error) {
			return ui.Confirm(in, errOut, title)
		},
	}
	a.NewClient = func(rate float64) (ghapi.Client, error) {
		gh, err := ghapi.New(ghapi.Options{WriteRate: rate, Timings: a.Timings})
		if err == nil {
			gh.Transport.OnWait = a.rateLimitWait
		}
		return gh, err
	}
	return a, nil
}

// rateLimitWait says why every request has stopped, so a long rate limit
// pause doesn't look like a hang.
func (a *App) rateLimitWait(why string, d time.Duration) {
	a.note(fmt.Sprintf("GitHub %s: waiting %s before the next request", why, d.Round(time.Second)))
}

// note prints a line about what's happening to stderr, above the live
// progress line when one is drawn.
func (a *App) note(msg string) {
	a.trackMu.Lock()
	t := a.tracker
	a.trackMu.Unlock()
	if t != nil {
		t.Note(msg)
		return
	}
	fmt.Fprintln(a.Err, msg)
}

// quiet reports whether progress notes would only clutter: machine output
// with nobody watching stderr.
func (a *App) quiet(c Common) bool {
	return (c.JSON || c.JQ != "" || c.Template != "") && !a.ErrTTY
}

// track starts reporting progress over total repos, verb naming the run,
// until the returned stop is called. With lines false, finished repos
// are counted but not listed.
func (a *App) track(c Common, verb string, total, concurrency int, lines bool) (*render.Tracker, func()) {
	machine := c.JSON || c.JQ != "" || c.Template != ""
	t := &render.Tracker{Printer: a.errPrinter(c), Live: a.ErrTTY, Quiet: machine || !lines, Verb: verb, Total: total, Interval: a.progressEvery}
	a.trackMu.Lock()
	a.tracker = t
	a.trackMu.Unlock()
	t.Start(concurrency)
	return t, func() {
		t.Stop()
		a.trackMu.Lock()
		a.tracker = nil
		a.trackMu.Unlock()
	}
}

// Common holds the flags most commands share.
type Common struct {
	// Campaign is a campaign name from local state or a file path.
	Campaign string
	// File is the campaign file (--file); defaults to ./fleet.yaml.
	File string
	// Repos limits a run to these repos (--repo and --repo-file).
	Repos []string
	// Exclude drops these repos from the targets (--exclude and
	// --exclude-file), on top of the campaign's exclude.repos.
	Exclude []string
	// Concurrency overrides limits.concurrency when positive.
	Concurrency int
	// WriteRate is API writes per second (--write-rate).
	WriteRate float64
	// Yes skips confirmation.
	Yes bool
	// KeepWorkdirs leaves clones on disk.
	KeepWorkdirs bool
	// JSON switches to JSON output (--json); JSONFields, when set, keeps
	// only those fields of each record.
	JSON       bool
	JSONFields []string
	JQ         string
	// Template formats JSON output with a Go template (--template).
	Template string
}

// ErrPending is returned by status --exit-status while repos still need
// work, so scripts can wait on a campaign; the process exits 8, like
// gh pr checks with pending checks.
var ErrPending = errors.New("some repos are still pending")

// ErrFailed is returned when one or more repos failed, so the process
// exits non-zero after reporting every repo.
var ErrFailed = errors.New("some repos failed")

// FailedError says how many repos failed.
type FailedError struct {
	N int
	// Hint is appended to the message, e.g. where to look next.
	Hint string
	// Noun is what failed; defaults to repo.
	Noun string
}

// Error describes the failure count.
func (e *FailedError) Error() string {
	noun := e.Noun
	if noun == "" {
		noun = "repo"
	}
	msg := plural(e.N, noun) + " failed"
	if e.Hint != "" {
		msg += "; " + e.Hint
	}
	return msg
}

// statusHint points at the command that shows per-repo errors.
const statusHint = "run gh fleet status for details"

// Unwrap makes errors.Is(err, ErrFailed) work.
func (e *FailedError) Unwrap() error { return ErrFailed }

// failures returns a FailedError when any result failed.
func failures(results []state.RepoState, hint string) error {
	n := 0
	for _, r := range results {
		if r.Status == state.Failed {
			n++
		}
	}
	if n > 0 {
		return &FailedError{N: n, Hint: hint}
	}
	return nil
}

// printer returns the output printer for c.
func (a *App) printer(c Common) render.Printer {
	p := render.Printer{Out: a.Out, JSON: c.JSON, Fields: c.JSONFields, JQ: c.JQ, Template: c.Template, TTY: a.TTY}
	p.Styles = ui.For(a.Color && !p.Machine())
	return p
}

// errPrinter returns a printer for progress written to stderr.
func (a *App) errPrinter(c Common) render.Printer {
	machine := c.JSON || c.JQ != "" || c.Template != ""
	return render.Printer{Out: a.Err, TTY: true, Styles: ui.For(a.Color && !machine)}
}

// LoadCampaign finds and loads the campaign named by c: --file, then a
// positional path, then a campaign name in local state, then ./fleet.yaml.
func (a *App) LoadCampaign(c Common) (*campaign.Campaign, error) {
	camp, err := a.loadCampaign(c)
	if err != nil {
		return nil, err
	}
	camp.Targets.Exclude.Repos = append(camp.Targets.Exclude.Repos, c.Exclude...)
	return camp, nil
}

// noteTargets prints the campaign's target warnings and notes on stderr:
// parts of targets that can't do what they look like they do, and
// combinations that are easy to misread.
func (a *App) noteTargets(c Common, camp *campaign.Campaign) {
	if a.quiet(c) {
		return
	}
	e := a.errPrinter(c)
	for _, w := range camp.Warnings() {
		a.note(e.Styles.Warn.Render("warning:") + " " + w)
	}
	for _, n := range camp.Notes() {
		a.note(e.Styles.Dim.Render("note:") + " " + n)
	}
}

// loadCampaign finds and parses the campaign c names.
func (a *App) loadCampaign(c Common) (*campaign.Campaign, error) {
	switch {
	case c.File != "":
		return campaign.Load(c.File)
	case c.Campaign != "":
		if st, err := os.Stat(c.Campaign); err == nil && !st.IsDir() {
			return campaign.Load(c.Campaign)
		}
		if !a.Store.Exists(c.Campaign) {
			return nil, fmt.Errorf("no campaign %q in local state and no such file; run gh fleet list", c.Campaign)
		}
		s, err := a.Store.Load(c.Campaign)
		if err != nil {
			return nil, err
		}
		if s.File == "" {
			return nil, fmt.Errorf("campaign %q has no campaign file recorded", c.Campaign)
		}
		return campaign.Load(s.File)
	default:
		if _, err := os.Stat(campaign.DefaultFile); err != nil {
			return nil, fmt.Errorf("no %s in the current directory; pass --file or a campaign name, or run gh fleet init", campaign.DefaultFile)
		}
		return campaign.Load(campaign.DefaultFile)
	}
}

// scripts compiles and checks every starlark step of a campaign, fetching
// any libraries they load. It returns the scripts by step index.
func (a *App) scripts(ctx context.Context, camp *campaign.Campaign) (map[int]*star.Script, error) {
	out := map[int]*star.Script{}
	for i, st := range camp.Change.Steps {
		if st.Starlark == "" {
			continue
		}
		s, err := star.Load(ctx, camp.StarlarkFile(st), camp.Dir(), star.Options{
			CacheDir: a.LibCache, Git: a.Git, RemoteURL: a.LibURL, Env: camp.StepEnv(st),
		})
		if err != nil {
			where := fmt.Sprintf("change.steps[%d]", i)
			if st.Name != "" {
				where += " (" + st.Name + ")"
			}
			return nil, fmt.Errorf("%s: %w", where, err)
		}
		out[i] = s
	}
	return out, nil
}

// resolve resolves a campaign's targets, applies --repo and works out each
// target's variables.
func (a *App) resolve(ctx context.Context, c Common, client ghapi.Client, camp *campaign.Campaign, repos []string, requirePush bool) (*target.Result, error) {
	rs := &target.Resolver{Client: client, Cache: a.Cache, AllowReadOnly: !requirePush, RequireProperties: camp.UsesProperties()}
	if !a.quiet(c) {
		rs.Progress = a.note
	}
	// A repo both listed and excluded is left out, not an error; when that
	// leaves none, there is nothing to do rather than no limit.
	limited := len(repos) > 0
	repos = slices.DeleteFunc(slices.Clone(repos), func(r string) bool { return excludedName(camp, r) })
	if limited && len(repos) == 0 {
		return &target.Result{}, nil
	}
	rs.Only = repos
	res, err := rs.Resolve(ctx, camp.Targets)
	if err != nil {
		return nil, err
	}
	if err := res.Limit(repos); err != nil {
		return nil, err
	}
	res.SetVars(&camp.Vars)
	return res, nil
}

// concurrency picks the flag value, then the campaign limit.
func concurrency(c Common, camp *campaign.Campaign) int {
	if c.Concurrency > 0 {
		return c.Concurrency
	}
	return camp.Limits.Concurrency
}

// searchScope returns search qualifiers covering a campaign's repos.
func searchScope(camp *campaign.Campaign) []string {
	var q []string
	for _, o := range camp.Targets.Orgs {
		q = append(q, "user:"+o)
	}
	for _, r := range camp.Targets.Include.Repos {
		owner, _, hasOwner := strings.Cut(r, "/")
		covered := !hasOwner || campaign.IsPattern(owner) // only allowed alongside orgs
		for _, o := range camp.Targets.Orgs {
			if strings.EqualFold(o, owner) {
				covered = true
			}
		}
		switch {
		case covered:
		case campaign.IsPattern(r):
			// Search can't take a pattern; scope to the owner instead.
			if !slices.Contains(q, "user:"+owner) {
				q = append(q, "user:"+owner)
			}
		default:
			q = append(q, "repo:"+r)
		}
	}
	return q
}

// skippedStates turns skipped targets into repo states.
func skippedStates(res *target.Result) []state.RepoState {
	out := make([]state.RepoState, len(res.Skipped))
	for i, s := range res.Skipped {
		out[i] = state.RepoState{Repo: s.Repo, Status: state.Skipped, Detail: s.Reason}
	}
	return out
}

// sortStates orders repo states by repo name.
func sortStates(rs []state.RepoState) {
	sort.Slice(rs, func(i, j int) bool { return rs[i].Repo < rs[j].Repo })
}

// confirm asks before a destructive or outward-facing step. Without a
// terminal it fails with an error naming --yes instead of hanging.
func (a *App) confirm(c Common, title string) error {
	if c.Yes {
		return nil
	}
	if !a.Interactive {
		return errors.New("confirmation required; pass --yes to run without a terminal")
	}
	ok, err := a.Confirm(title)
	if err != nil {
		return err
	}
	if !ok {
		return ui.ErrNotConfirmed
	}
	return nil
}
