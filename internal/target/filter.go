// Package target resolves a campaign's targets into a list of repos.
//
// Org listings come from one paginated GraphQL query per org (cached for
// ten minutes), file filters from batched object(expression:) lookups and
// search filters from the code search API. Every matched repo carries the
// reasons it matched.
package target

import (
	"slices"
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
// the include filters; exclude always wins.
func Evaluate(t campaign.Targets, r ghapi.Repo, f Facts) Decision {
	if why := excluded(t.Exclude, r); why != "" {
		return Decision{Excluded: why}
	}
	if containsFold(t.Include.Repos, r.FullName()) {
		return Decision{Match: true, Reasons: []string{"include.repos"}}
	}
	inc := t.Include
	var reasons []string
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
	relaxed := t
	relaxed.Include.Files = nil
	relaxed.Include.Search = ""
	return Evaluate(relaxed, r, Facts{}).Match
}

// excluded returns the exclude rule that drops r, or "".
func excluded(e campaign.Exclude, r ghapi.Repo) string {
	switch {
	case containsFold(e.Repos, r.FullName()):
		return "exclude.repos"
	case r.Archived && (e.Archived == nil || *e.Archived):
		return "archived"
	case r.Fork && (e.Forks == nil || *e.Forks):
		return "fork"
	}
	if t := firstShared(e.Topics, r.Topics); t != "" {
		return "topic:" + t
	}
	return ""
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
