package star

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
)

// LibPrefix starts the load path of a module in the built-in library, as
// in load("@fleet//files.star", "ensure_lines").
const LibPrefix = "@fleet//"

// libFS holds the built-in library modules.
//
//go:embed lib/*.star
var libFS embed.FS

// resolveLib checks that a @fleet// module exists and returns its path
// and root. Library modules are keyed by their load path.
func resolveLib(name string) (string, string, error) {
	rel := strings.TrimPrefix(name, LibPrefix)
	if _, err := fs.Stat(libFS, "lib/"+rel); err != nil || strings.Contains(rel, "/") {
		return "", "", fmt.Errorf("no built-in module %s; have %s", rel, strings.Join(LibModules(), ", "))
	}
	return name, LibPrefix, nil
}

// LibModules lists the built-in library's module file names.
func LibModules() []string {
	entries, _ := fs.ReadDir(libFS, "lib")
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.Name())
	}
	sort.Strings(out)
	return out
}

// LibSource returns the source of a built-in module by file name.
func LibSource(name string) ([]byte, error) {
	return fs.ReadFile(libFS, path.Join("lib", name))
}
