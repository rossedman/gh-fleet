// Command generated prints the import-path-qualified names of generated Go
// files, one per line, so the coverage gate can leave them out.
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/rossedman/gh-fleet/tools/internal/gofiles"
)

// module is this repo's module path.
const module = "github.com/rossedman/gh-fleet"

// main prints generated files under the given package patterns.
func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run prints generated files and returns the exit code.
func run(patterns []string, out, errOut io.Writer) int {
	files, err := gofiles.Walk(patterns)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	for _, f := range files {
		if f.Generated {
			fmt.Fprintln(out, module+"/"+strings.TrimPrefix(f.Path, "./"))
		}
	}
	return 0
}
