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
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rossedman/gh-fleet/internal/campaign"
	"github.com/rossedman/gh-fleet/internal/ghapi"
	"github.com/rossedman/gh-fleet/internal/gitops"
	"github.com/rossedman/gh-fleet/internal/render"
	"github.com/rossedman/gh-fleet/internal/star"
	"github.com/rossedman/gh-fleet/internal/state"
	"github.com/rossedman/gh-fleet/internal/target"
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
	// NewClient builds a GitHub client with the given write rate.
	NewClient  func(writeRate float64) (ghapi.Client, error)
	Store      *state.Store
	Cache      *target.Cache
	Workspaces *workspace.Manager
	Git        gitops.Git
	// Confirm asks a yes/no question; only called when Interactive.
	Confirm func(title string) (bool, error)
	// LibCache holds Starlark libraries loaded from other repositories.
	LibCache string
	// LibURL overrides the clone URL of those libraries (for tests).
	LibURL func(host, owner, repo string) string
}

// Default wires the real GitHub client, state dir, cache and git.
func Default(out, errOut io.Writer, in io.Reader, interactive, color bool) (*App, error) {
	stateDir, err := state.DefaultDir()
	if err != nil {
		return nil, err
	}
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir = os.TempDir()
	}
	return &App{
		Out: out, Err: errOut, In: in, Interactive: interactive, Color: color,
		NewClient: func(rate float64) (ghapi.Client, error) {
			return ghapi.New(ghapi.Options{WriteRate: rate})
		},
		Store:      &state.Store{Dir: stateDir},
		Cache:      &target.Cache{Dir: filepath.Join(cacheDir, "gh-fleet", "repos")},
		Workspaces: &workspace.Manager{},
		LibCache:   filepath.Join(cacheDir, "gh-fleet", "starlark"),
		Confirm: func(title string) (bool, error) {
			return ui.Confirm(in, errOut, title)
		},
	}, nil
}

// Common holds the flags most commands share.
type Common struct {
	// Campaign is a campaign name from local state or a file path.
	Campaign string
	// File is the campaign file (--file); defaults to ./fleet.yaml.
	File string
	// Repos limits a run to these repos (--repo).
	Repos []string
	// Concurrency overrides limits.concurrency when positive.
	Concurrency int
	// WriteRate is API writes per second (--write-rate).
	WriteRate float64
	// Yes skips confirmation.
	Yes bool
	// KeepWorkdirs leaves clones on disk.
	KeepWorkdirs bool
	JSON         bool
	JQ           string
}

// ErrFailed is returned when one or more repos failed, so the process
// exits non-zero after reporting every repo.
var ErrFailed = errors.New("some repos failed")

// FailedError says how many repos failed.
type FailedError struct {
	N int
	// Hint is appended to the message, e.g. where to look next.
	Hint string
}

// Error describes the failure count.
func (e *FailedError) Error() string {
	msg := plural(e.N, "repo") + " failed"
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
	p := render.Printer{Out: a.Out, JSON: c.JSON, JQ: c.JQ}
	p.Styles = ui.For(a.Color && !p.Machine())
	return p
}

// errPrinter returns a printer for progress written to stderr.
func (a *App) errPrinter(c Common) render.Printer {
	return render.Printer{Out: a.Err, Styles: ui.For(a.Color && !c.JSON && c.JQ == "")}
}

// LoadCampaign finds and loads the campaign named by c: --file, then a
// positional path, then a campaign name in local state, then ./fleet.yaml.
func (a *App) LoadCampaign(c Common) (*campaign.Campaign, error) {
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
func (a *App) resolve(ctx context.Context, client ghapi.Client, camp *campaign.Campaign, repos []string, requirePush bool) (*target.Result, error) {
	rs := &target.Resolver{Client: client, Cache: a.Cache, AllowReadOnly: !requirePush, RequireProperties: camp.UsesProperties()}
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
		owner, _, _ := strings.Cut(r, "/")
		covered := false
		for _, o := range camp.Targets.Orgs {
			if strings.EqualFold(o, owner) {
				covered = true
			}
		}
		if !covered {
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
