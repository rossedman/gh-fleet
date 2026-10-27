package app

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/rossedman/gh-fleet/internal/campaign"
	"github.com/rossedman/gh-fleet/internal/ghapi"
	"github.com/rossedman/gh-fleet/internal/pr"
	"github.com/rossedman/gh-fleet/internal/state"
	"github.com/rossedman/gh-fleet/internal/target"
)

// waveGate is which targets a run may change under limits.waves.
type waveGate struct {
	// Allowed are the repos of the waves that are open, plus any repo
	// that already has an open PR.
	Allowed map[string]bool
	// Held maps each held-back repo to why it waits.
	Held map[string]string
	// Current is the 1-based wave being rolled out; Waves how many there
	// are, the final catch-all included.
	Current, Waves int
}

// splitWaves assigns sorted target names to waves: each wave takes its
// named repos, then the next Size targets; what's left is a final wave.
func splitWaves(waves []campaign.Wave, names []string) [][]string {
	remaining := slices.Clone(names)
	slices.Sort(remaining)
	var out [][]string
	for _, w := range waves {
		var members []string
		// A wave takes every remaining target its names or patterns match.
		remaining = slices.DeleteFunc(remaining, func(r string) bool {
			if campaign.MatchAny(w.Repos, r) != "" {
				members = append(members, r)
				return true
			}
			return false
		})
		if w.Size.N > 0 {
			n := min(w.Size.Of(len(names)), len(remaining))
			members = append(members, remaining[:n]...)
			remaining = remaining[n:]
		}
		if len(members) > 0 {
			slices.Sort(members)
			out = append(out, members)
		}
	}
	if len(remaining) > 0 {
		out = append(out, remaining)
	}
	return out
}

// gateWaves opens the first wave with a repo that isn't done, and every
// wave before it; done reports a merged or unchanged repo, open one with
// an open PR, which always stays allowed so it keeps being updated.
func gateWaves(waves []campaign.Wave, names []string, done, open func(string) bool) waveGate {
	split := splitWaves(waves, names)
	g := waveGate{Allowed: map[string]bool{}, Held: map[string]string{}, Waves: len(split)}
	g.Current = len(split)
	for i, wave := range split {
		finished := 0
		for _, r := range wave {
			g.Allowed[r] = true
			if done(r) {
				finished++
			}
		}
		if finished < len(wave) {
			g.Current = i + 1
			for j := i + 1; j < len(split); j++ {
				for _, r := range split[j] {
					if open(r) {
						g.Allowed[r] = true
						continue
					}
					g.Held[r] = fmt.Sprintf("wave %d of %d: starts when wave %d is done (%d of %d merged or unchanged)", j+1, len(split), i+1, finished, len(wave))
				}
			}
			break
		}
	}
	return g
}

// waveDone reports a repo whose part of a rollout is finished: its PR
// merged, or the last run found nothing to change.
func waveDone(st *state.Campaign, prs *pr.Manager) func(string) bool {
	return func(r string) bool {
		if p := prs.Existing(r); p != nil && p.State == ghapi.StateMerged {
			return true
		}
		rs := st.Get(r)
		return rs != nil && rs.Status == state.Noop
	}
}

// waveOpen reports a repo with an open campaign PR.
func waveOpen(prs *pr.Manager) func(string) bool {
	return func(r string) bool {
		p := prs.Existing(r)
		return p != nil && p.State == ghapi.StateOpen
	}
}

// planWave narrows a plan to one wave of limits.waves: wave, 1-based, or
// when it is 0 the wave apply would roll out now. Targets with no waves
// are returned as they are.
func (a *App) planWave(ctx context.Context, c Common, client ghapi.Client, camp *campaign.Campaign, res *target.Result, wave int) ([]target.Target, error) {
	if len(camp.Limits.Waves) == 0 {
		if wave > 0 {
			return nil, errors.New("--wave needs limits.waves in the campaign")
		}
		return res.Targets, nil
	}
	names := res.Names()
	split := splitWaves(camp.Limits.Waves, names)
	if wave > len(split) {
		return nil, fmt.Errorf("--wave %d: the campaign's targets make %s", wave, plural(len(split), "wave"))
	}
	if wave == 0 {
		st, err := a.Store.Load(camp.Name)
		if err != nil {
			return nil, err
		}
		prs := pr.NewManager(client, 0)
		if !a.quiet(c) {
			a.note("Looking up existing PRs in " + plural(len(names), "repo") + " to find the current wave")
		}
		if err := prs.Index(ctx, names, camp.PR.Branch, camp.Marker(), searchScope(camp)); err != nil {
			return nil, err
		}
		wave = gateWaves(camp.Limits.Waves, names, waveDone(st, prs), waveOpen(prs)).Current
	}
	in := map[string]bool{}
	if wave > 0 {
		for _, r := range split[wave-1] {
			in[r] = true
		}
	}
	var kept []target.Target
	for _, t := range res.Targets {
		if in[t.Repo.FullName()] {
			kept = append(kept, t)
		}
	}
	if !a.quiet(c) {
		a.note(fmt.Sprintf("Wave %d of %d: planning %s; %s in other waves (--wave N or --all-waves plans them)",
			wave, len(split), plural(len(kept), "repo"), plural(len(names)-len(kept), "repo")))
	}
	return kept, nil
}
