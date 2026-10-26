// Package target resolves a campaign's targets into a list of repos.
//
// Org listings come from one paginated GraphQL query per org (cached for
// ten minutes), file filters from batched object(expression:) lookups and
// search filters from the code search API. Every matched repo carries the
// reasons it matched.
package target

import (
	"slices"
	"sort"
	"strings"

	"github.com/rossedman/gh-fleet/internal/campaign"
	"github.com/rossedman/gh-fleet/internal/ghapi"
)

// Facts are what is known about a repo beyond its metadata.
type Facts struct {
	// Files lists the include.files paths that exist in the repo.
	Files []string
	// SearchHit reports whether the include.search query hit the repo.
	SearchHit bool
}

// Decision is the outcome of evaluating the filters for one repo.
type Decision struct {
	Match bool
	// Reasons say why the repo matched, e.g. "topic:service".
	Reasons []string
	// Excluded names the exclude rule that dropped the repo, if any.
	Excluded string
}

// Evaluate applies include and exclude filters to one repo. Include keys
// combine with AND and values within a key with OR; include.repos bypasses
// the include filters; exclude always wins. Repo lists and include.names
// take glob patterns. With no include filters, every repo of orgs matches,
// unless include.repos is set: then only the repos it names do.
func Evaluate(t campaign.Targets, r ghapi.Repo, f Facts) Decision {
	return evaluate(t, r, f, false)
}

// OnlyRepos reports whether include.repos is the whole selection: it is
// set and no include filter is, so orgs only scope its bare names and
// patterns rather than adding all of their repos.
func OnlyRepos(inc campaign.Include) bool {
	return len(inc.Repos) > 0 && len(inc.Names) == 0 && len(inc.Topics) == 0 && len(inc.Languages) == 0 &&
		len(inc.Files) == 0 && inc.Search == "" && len(inc.Properties) == 0
}

// evaluate is Evaluate; with metadataOnly, the file and search filters,
// whose facts aren't known yet, pass.
func evaluate(t campaign.Targets, r ghapi.Repo, f Facts, metadataOnly bool) Decision {
	if why := excluded(t.Exclude, r); why != "" {
		return Decision{Excluded: why}
	}
	if campaign.MatchAny(t.Include.Repos, r.FullName()) != "" {
		return Decision{Match: true, Reasons: []string{"include.repos"}}
	}
	if OnlyRepos(t.Include) {
		return Decision{}
	}
	inc := t.Include
	if metadataOnly {
		inc.Files, inc.Search = nil, ""
	}
	var reasons []string
	if len(inc.Names) > 0 {
		hit := campaign.MatchAny(inc.Names, r.FullName())
		if hit == "" {
			return Decision{}
		}
		reasons = append(reasons, "name:"+hit)
	}
	if len(inc.Topics) > 0 {
		hit := firstShared(inc.Topics, r.Topics)
		if hit == "" {
			return Decision{}
		}
		reasons = append(reasons, "topic:"+hit)
	}
	if len(inc.Languages) > 0 {
		if r.Language == "" || !containsFold(inc.Languages, r.Language) {
			return Decision{}
		}
		reasons = append(reasons, "language:"+r.Language)
	}
	if len(inc.Files) > 0 {
		hit := ""
		for _, p := range inc.Files {
			if slices.Contains(f.Files, p) {
				hit = p
				break
			}
		}
		if hit == "" {
			return Decision{}
		}
		reasons = append(reasons, "file:"+hit)
	}
	if len(inc.Properties) > 0 {
		hits, ok := matchProperties(inc.Properties, r.Properties)
		if !ok {
			return Decision{}
		}
		reasons = append(reasons, hits...)
	}
	if inc.Search != "" {
		if !f.SearchHit {
			return Decision{}
		}
		reasons = append(reasons, "search")
	}
	if len(reasons) == 0 {
		reasons = []string{"org:" + r.Owner}
	}
	return Decision{Match: true, Reasons: reasons}
}

// MetadataOnly reports whether a repo can still match once file and
// search facts are known, so those lookups can skip hopeless repos.
func MetadataOnly(t campaign.Targets, r ghapi.Repo) bool {
	return evaluate(t, r, Facts{}, true).Match
}

// excluded returns the exclude rule that drops r, or "".
func excluded(e campaign.Exclude, r ghapi.Repo) string {
	switch {
	case campaign.MatchAny(e.Repos, r.FullName()) != "":
		return "exclude.repos"
	case r.Archived && (e.Archived == nil || *e.Archived):
		return "archived"
	case r.Fork && (e.Forks == nil || *e.Forks):
		return "fork"
	}
	if t := firstShared(e.Topics, r.Topics); t != "" {
		return "topic:" + t
	}
	for _, name := range sortedKeys(e.Properties) {
		if hits, ok := matchProperties(map[string]campaign.StringList{name: e.Properties[name]}, r.Properties); ok {
			return hits[0]
		}
	}
	return ""
}

// matchProperties reports whether every filtered property has one of its
// allowed values, returning a reason per property. Multi-select values
// match when any selected value is allowed.
func matchProperties(want map[string]campaign.StringList, have map[string]string) ([]string, bool) {
	var reasons []string
	for _, name := range sortedKeys(want) {
		v, ok := lookupFold(have, name)
		if !ok {
			return nil, false
		}
		hit := ""
		for _, part := range strings.Split(v, ",") {
			if containsFold(want[name], part) {
				hit = part
				break
			}
		}
		if hit == "" {
			return nil, false
		}
		reasons = append(reasons, "property:"+name+"="+hit)
	}
	return reasons, true
}

// lookupFold finds a map key ignoring case.
func lookupFold(m map[string]string, key string) (string, bool) {
	if v, ok := m[key]; ok {
		return v, true
	}
	for k, v := range m {
		if strings.EqualFold(k, key) {
			return v, true
		}
	}
	return "", false
}

// sortedKeys returns a map's keys in order, for deterministic reasons.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// firstShared returns the first value of want that has contains, ignoring
// case, or "".
func firstShared(want, have []string) string {
	for _, w := range want {
		if containsFold(have, w) {
			return w
		}
	}
	return ""
}

// containsFold reports whether list contains s, ignoring case.
func containsFold(list []string, s string) bool {
	for _, v := range list {
		if strings.EqualFold(v, s) {
			return true
		}
	}
	return false
}
