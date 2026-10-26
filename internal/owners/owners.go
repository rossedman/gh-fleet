// Package owners works out who owns the files a campaign changed, from a
// repo's CODEOWNERS file or its Backstage catalog, so their PRs can ask
// those people for review.
package owners

import (
	"bytes"
	"errors"
	"io"
	"path"
	"regexp"
	"strings"

	"go.yaml.in/yaml/v3"
)

// CodeownersPaths are where GitHub looks for CODEOWNERS, in its order.
var CodeownersPaths = []string{".github/CODEOWNERS", "CODEOWNERS", "docs/CODEOWNERS"}

// CatalogPaths are where a repo's Backstage catalog file usually lives.
var CatalogPaths = []string{"catalog-info.yaml", "catalog-info.yml"}

// rule is one CODEOWNERS line.
type rule struct {
	re     *regexp.Regexp
	owners []string
}

// Codeowners is a parsed CODEOWNERS file.
type Codeowners struct {
	rules []rule
}

// ParseCodeowners reads CODEOWNERS content. Lines with patterns it can't
// read are skipped, as GitHub skips them.
func ParseCodeowners(content string) Codeowners {
	var c Codeowners
	for _, line := range strings.Split(content, "\n") {
		if i := strings.Index(line, " #"); i >= 0 {
			line = line[:i]
		}
		fields := strings.Fields(line)
		if len(fields) == 0 || strings.HasPrefix(fields[0], "#") {
			continue
		}
		re, err := patternRegexp(fields[0])
		if err != nil {
			continue
		}
		c.rules = append(c.rules, rule{re: re, owners: fields[1:]})
	}
	return c
}

// patternRegexp turns a CODEOWNERS pattern, which follows .gitignore
// rules, into a regular expression over slash-separated paths.
func patternRegexp(p string) (*regexp.Regexp, error) {
	if strings.HasPrefix(p, "!") || strings.Contains(p, "[") {
		return nil, errors.New("unsupported pattern")
	}
	anchored := strings.HasPrefix(p, "/") || strings.Contains(strings.TrimSuffix(p, "/"), "/")
	dir := strings.HasSuffix(p, "/")
	p = strings.Trim(p, "/")
	var b strings.Builder
	if anchored {
		b.WriteString("^")
	} else {
		b.WriteString("(^|/)")
	}
	for i := 0; i < len(p); i++ {
		switch c := p[i]; {
		case c == '*' && i+1 < len(p) && p[i+1] == '*':
			i++
			if i+1 < len(p) && p[i+1] == '/' {
				i++
				b.WriteString("(.*/)?")
			} else {
				b.WriteString(".*")
			}
		case c == '*':
			b.WriteString("[^/]*")
		case c == '?':
			b.WriteString("[^/]")
		default:
			b.WriteString(regexp.QuoteMeta(string(c)))
		}
	}
	switch {
	case dir:
		b.WriteString("/")
	case strings.HasSuffix(p, "/*"):
		// docs/* owns the files directly in docs, not those deeper.
		b.WriteString("$")
	default:
		// A name is a file, or a directory and everything in it.
		b.WriteString("(/|$)")
	}
	return regexp.Compile(b.String())
}

// Owners returns the owners of paths: for each path the last rule that
// matches it, as GitHub decides, merged in order of first appearance.
// Teams and users come back as "org/team" and "user"; email addresses
// are dropped, since review can't be requested from them.
func (c Codeowners) Owners(paths []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, p := range paths {
		var match *rule
		for i := range c.rules {
			if c.rules[i].re.MatchString(p) {
				match = &c.rules[i]
			}
		}
		if match == nil {
			continue
		}
		for _, o := range match.owners {
			if !strings.HasPrefix(o, "@") {
				continue
			}
			o = strings.TrimPrefix(o, "@")
			if !seen[strings.ToLower(o)] {
				seen[strings.ToLower(o)] = true
				out = append(out, o)
			}
		}
	}
	return out
}

// CatalogOwners returns the owners of the entities in a Backstage catalog
// file as GitHub reviewers. A group ref, written as payments, group:payments
// or group:default/payments, becomes the team org/payments; a user ref
// becomes the user. Refs to other kinds are dropped.
func CatalogOwners(content []byte, org string) []string {
	dec := yaml.NewDecoder(bytes.NewReader(content))
	var out []string
	seen := map[string]bool{}
	for {
		var doc struct {
			Spec struct {
				Owner string `yaml:"owner"`
			} `yaml:"spec"`
		}
		err := dec.Decode(&doc)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			// A document that isn't an entity, such as a list, ends the
			// file for this purpose.
			break
		}
		r := reviewerFor(doc.Spec.Owner, org)
		if r != "" && !seen[r] {
			seen[r] = true
			out = append(out, r)
		}
	}
	return out
}

// reviewerFor converts a Backstage owner ref to a GitHub reviewer.
func reviewerFor(ref, org string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	kind, name, ok := strings.Cut(ref, ":")
	if !ok {
		kind, name = "group", ref
	}
	name = path.Base(name) // drops a namespace such as default/
	switch strings.ToLower(kind) {
	case "group":
		return org + "/" + name
	case "user":
		return name
	}
	return ""
}
