package target

import (
	"maps"
	"slices"

	"github.com/rossedman/gh-fleet/internal/campaign"
	"github.com/rossedman/gh-fleet/internal/ghapi"
)

// Vars builds one repo's variables: vars.default, then every vars.match
// rule whose when matches the repo, in order, then the repo's row in
// vars.file. Later values replace earlier ones.
func Vars(v *campaign.Vars, r ghapi.Repo) map[string]any {
	out := map[string]any{}
	set := func(m map[string]any) {
		for k, val := range m {
			out[k] = cloneValue(val)
		}
	}
	set(v.Default)
	for _, rule := range v.Match {
		if When(rule.When, r) {
			set(rule.Set)
		}
	}
	set(v.Row(r.FullName()))
	return out
}

// When reports whether a vars.match rule applies to a repo: every key that
// is set must match, values within a key are alternatives.
func When(w campaign.When, r ghapi.Repo) bool {
	if len(w.Repos) > 0 && campaign.MatchAny(w.Repos, r.FullName()) == "" {
		return false
	}
	if len(w.Topics) > 0 && firstShared(w.Topics, r.Topics) == "" {
		return false
	}
	if len(w.Languages) > 0 && (r.Language == "" || !containsFold(w.Languages, r.Language)) {
		return false
	}
	if len(w.Properties) > 0 {
		if _, ok := matchProperties(w.Properties, r.Properties); !ok {
			return false
		}
	}
	return true
}

// cloneValue copies list values so repos never share a slice.
func cloneValue(v any) any {
	if list, ok := v.([]any); ok {
		return slices.Clone(list)
	}
	if m, ok := v.(map[string]any); ok {
		return maps.Clone(m)
	}
	return v
}
