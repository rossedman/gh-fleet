package campaign

import (
	"fmt"
	"strings"
)

// Warnings returns parts of the targets that can't do what they look like
// they do: filters with no orgs to filter, and include.names entries for
// owners outside orgs. A campaign with warnings still loads.
func (c *Campaign) Warnings() []string {
	t := c.Targets
	var out []string
	if keys := filterKeys(t.Include); len(t.Orgs) == 0 && len(keys) > 0 {
		out = append(out, fmt.Sprintf("%s %s the repos of orgs, and targets has no orgs, so %s nothing; the targets are only what include.repos names",
			joinKeys(keys), verb(keys, "filters", "filter"), verb(keys, "it selects", "they select")))
	}
	if len(t.Orgs) > 0 {
		for _, n := range t.Include.Names {
			owner, _, ok := strings.Cut(n, "/")
			if ok && !IsPattern(owner) && !hasOwner(t.Orgs, owner) {
				out = append(out, fmt.Sprintf("include.names %q names an owner that isn't in orgs, so it can't match; to add a repo from another owner, list it in include.repos", n))
			}
		}
	}
	return out
}

// Notes returns how the targets combine where that is easy to misread:
// include.repos beside other include filters adds to what they select in
// orgs, rather than being the whole selection.
func (c *Campaign) Notes() []string {
	t := c.Targets
	keys := filterKeys(t.Include)
	if len(t.Orgs) == 0 || len(t.Include.Repos) == 0 || len(keys) == 0 {
		return nil
	}
	return []string{fmt.Sprintf("include.repos adds its repos to what %s %s in %s, so every repo there is still checked; with include.repos alone, the targets are only the repos it names",
		joinKeys(keys), verb(keys, "selects", "select"), strings.Join(t.Orgs, ", "))}
}

// filterKeys names the include filters that are set, in the order the
// campaign reference lists them.
func filterKeys(inc Include) []string {
	var keys []string
	for _, k := range []struct {
		name string
		set  bool
	}{
		{"names", len(inc.Names) > 0},
		{"topics", len(inc.Topics) > 0},
		{"languages", len(inc.Languages) > 0},
		{"files", len(inc.Files) > 0},
		{"search", inc.Search != ""},
		{"properties", len(inc.Properties) > 0},
	} {
		if k.set {
			keys = append(keys, "include."+k.name)
		}
	}
	return keys
}

// joinKeys lists keys as prose: "a", "a and b", "a, b and c".
func joinKeys(keys []string) string {
	if len(keys) == 1 {
		return keys[0]
	}
	return strings.Join(keys[:len(keys)-1], ", ") + " and " + keys[len(keys)-1]
}

// verb picks the singular or plural form for keys.
func verb(keys []string, one, many string) string {
	if len(keys) == 1 {
		return one
	}
	return many
}

// hasOwner reports whether owner is one of orgs, ignoring case.
func hasOwner(orgs []string, owner string) bool {
	for _, o := range orgs {
		if strings.EqualFold(o, owner) {
			return true
		}
	}
	return false
}
