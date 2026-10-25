// Package engine runs a campaign's per-repo pipeline across every target
// with a bounded worker pool.
//
// One repo's failure never stops the others. A fatal error (such as the
// primary rate limit headroom being reached) stops new repos from
// starting, and canceling the context does the same: in-flight repos
// finish and the rest are reported as pending.
package engine

import (
	"context"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/rossedman/gh-fleet/internal/state"
	"github.com/rossedman/gh-fleet/internal/target"
)

// Processor handles one repo. A non-nil error is fatal for the whole run.
type Processor interface {
	Process(ctx context.Context, t target.Target) (state.RepoState, error)
}

// ProcessorFunc adapts a function to Processor.
type ProcessorFunc func(ctx context.Context, t target.Target) (state.RepoState, error)

// Process calls f.
func (f ProcessorFunc) Process(ctx context.Context, t target.Target) (state.RepoState, error) {
	return f(ctx, t)
}

// Options configures Run.
type Options struct {
	// Concurrency is the number of repos processed at once (default 8).
	Concurrency int
	// OnResult is called once per repo as it finishes, from one goroutine
	// at a time.
	OnResult func(state.RepoState)
}

// Run processes every target and returns their states in target order,
// plus the first fatal error, if any.
func Run(ctx context.Context, targets []target.Target, p Processor, opts Options) ([]state.RepoState, error) {
	n := opts.Concurrency
	if n < 1 {
		n = 8
	}
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	results := make([]state.RepoState, len(targets))
	started := make([]bool, len(targets))
	var mu sync.Mutex
	var fatal error

	g := new(errgroup.Group)
	g.SetLimit(n)
	for i, t := range targets {
		if ctx.Err() != nil {
			break
		}
		g.Go(func() error {
			if ctx.Err() != nil {
				return nil
			}
			mu.Lock()
			started[i] = true
			mu.Unlock()
			rs, err := p.Process(ctx, t)
			if rs.Repo == "" {
				rs.Repo = t.Repo.FullName()
			}
			mu.Lock()
			defer mu.Unlock()
			results[i] = rs
			if err != nil && fatal == nil {
				fatal = err
				cancel(err)
			}
			if opts.OnResult != nil {
				opts.OnResult(rs)
			}
			return nil
		})
	}
	_ = g.Wait()

	for i, t := range targets {
		if !started[i] {
			results[i] = state.RepoState{Repo: t.Repo.FullName(), Status: state.Pending, Detail: "not started"}
		}
	}
	return results, fatal
}
