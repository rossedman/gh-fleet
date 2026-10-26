// Package examples holds the tested example campaigns. FS embeds them in
// the binary, so gh fleet init --from can start a campaign from one.
package examples

import (
	"embed"
	"io/fs"
	"sort"
)

// FS holds every example's campaign files and test cases.
//
//go:embed */fleet.yaml */*.star */*.md */*.csv */*.tmpl all:*/files all:*/tests
var FS embed.FS

// Names lists the examples, sorted.
func Names() []string {
	matches, _ := fs.Glob(FS, "*/fleet.yaml")
	names := make([]string, len(matches))
	for i, m := range matches {
		names[i] = m[:len(m)-len("/fleet.yaml")]
	}
	sort.Strings(names)
	return names
}
