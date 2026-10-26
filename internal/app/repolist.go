package app

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/rossedman/gh-fleet/internal/campaign"
	"github.com/rossedman/gh-fleet/internal/ghapi"
)

// ParseRepoList reads repo names, one per line. Blank lines and # comments
// are skipped, and only a line's first field counts, so the output of
// gh repo list works as is. A name may be owner/name or a GitHub URL. name
// labels errors.
func ParseRepoList(r io.Reader, name string) ([]string, error) {
	var out []string
	sc := bufio.NewScanner(r)
	for n := 1; sc.Scan(); n++ {
		line, _, _ := strings.Cut(sc.Text(), "#")
		fields := strings.FieldsFunc(line, func(c rune) bool { return c == ',' || c == ' ' || c == '\t' || c == '\r' })
		if len(fields) == 0 {
			continue
		}
		repo := repoName(fields[0])
		if _, _, err := ghapi.SplitRepo(repo); err != nil {
			return nil, fmt.Errorf("%s:%d: %q is not owner/name", name, n, fields[0])
		}
		out = append(out, repo)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return out, nil
}

// repoName trims a GitHub URL or clone address down to owner/name.
func repoName(s string) string {
	for _, p := range []string{"https://", "http://", "git@github.com:", "github.com/", "www.github.com/"} {
		s = strings.TrimPrefix(s, p)
	}
	return strings.TrimSuffix(strings.TrimSuffix(s, "/"), ".git")
}

// ReadRepoFiles reads every file in paths with ParseRepoList and returns
// the names in order, without duplicates. A path of - reads stdin.
func ReadRepoFiles(paths []string, stdin io.Reader) ([]string, error) {
	var out []string
	seen := map[string]bool{}
	for _, p := range paths {
		var names []string
		var err error
		if p == "-" {
			names, err = ParseRepoList(stdin, "stdin")
		} else {
			var f *os.File
			if f, err = os.Open(p); err != nil {
				return nil, err
			}
			names, err = ParseRepoList(f, p)
			_ = f.Close()
		}
		if err != nil {
			return nil, err
		}
		for _, n := range names {
			if !seen[strings.ToLower(n)] {
				seen[strings.ToLower(n)] = true
				out = append(out, n)
			}
		}
	}
	return out, nil
}

// excludedName reports whether the campaign excludes a repo by name or
// pattern.
func excludedName(camp *campaign.Campaign, repo string) bool {
	return campaign.MatchAny(camp.Targets.Exclude.Repos, repo) != ""
}
