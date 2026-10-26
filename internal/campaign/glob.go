package campaign

import (
	"path"
	"strings"
)

// IsPattern reports whether s holds a glob character, so it names a set
// of repos rather than one.
func IsPattern(s string) bool {
	return strings.ContainsAny(s, "*?[")
}

// MatchRepo reports whether the repo named full (owner/name) matches
// pattern, ignoring case. A pattern with a slash matches the full name,
// one without matches the name alone. Patterns use path.Match syntax: *
// matches any run of characters except /, ? one character, and [a-z] a
// class.
func MatchRepo(pattern, full string) bool {
	pattern, full = strings.ToLower(pattern), strings.ToLower(full)
	if !strings.Contains(pattern, "/") {
		_, full, _ = strings.Cut(full, "/")
	}
	ok, _ := path.Match(pattern, full)
	return ok
}

// MatchAny returns the first of patterns that matches full, or "".
func MatchAny(patterns []string, full string) string {
	for _, p := range patterns {
		if MatchRepo(p, full) {
			return p
		}
	}
	return ""
}
