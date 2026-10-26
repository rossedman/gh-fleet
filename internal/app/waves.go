package app

import (
	"fmt"
	"slices"

	"github.com/rossedman/gh-fleet/internal/campaign"
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
	take := func(r string) bool {
		if i := slices.Index(remaining, r); i >= 0 {
			remaining = slices.Delete(remaining, i, i+1)
			return true
		}
		return false
	}
	var out [][]string
	for _, w := range waves {
		var members []string
		for _, r := range w.Repos {
			if take(r) {
				members = append(members, r)
			}
		}
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
