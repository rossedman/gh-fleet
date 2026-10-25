// Command doccheck fails when a package, function or method has no doc
// comment. Test files and generated files are skipped.
package main

import (
	"fmt"
	"go/ast"
	"io"
	"os"
	"path"
	"sort"

	"github.com/rossedman/gh-fleet/tools/internal/gofiles"
)

// main reports undocumented declarations under the given patterns.
func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run checks the patterns and returns the exit code.
func run(patterns []string, out, errOut io.Writer) int {
	files, err := gofiles.Walk(patterns)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	problems := Check(files)
	for _, p := range problems {
		fmt.Fprintln(out, p)
	}
	if len(problems) > 0 {
		fmt.Fprintf(errOut, "doccheck: %d missing doc comments\n", len(problems))
		return 1
	}
	return 0
}

// Check returns one line per missing package or function doc comment.
func Check(files []gofiles.File) []string {
	var out []string
	pkgDoc := map[string]bool{}
	pkgSeen := map[string]string{}
	for _, f := range files {
		if f.Generated {
			continue
		}
		dir := path.Dir(f.Path)
		pkgSeen[dir] = f.AST.Name.Name
		if f.AST.Doc != nil {
			pkgDoc[dir] = true
		}
		for _, d := range f.AST.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Doc != nil {
				continue
			}
			name := fn.Name.Name
			if fn.Recv != nil && len(fn.Recv.List) > 0 {
				name = recvName(fn.Recv.List[0].Type) + "." + name
			}
			pos := f.Fset.Position(fn.Pos())
			out = append(out, fmt.Sprintf("%s:%d: %s has no doc comment", f.Path, pos.Line, name))
		}
	}
	for dir, name := range pkgSeen {
		if !pkgDoc[dir] {
			out = append(out, fmt.Sprintf("%s: package %s has no package comment", dir, name))
		}
	}
	sort.Strings(out)
	return out
}

// recvName returns the type name of a method receiver.
func recvName(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.StarExpr:
		return recvName(t.X)
	case *ast.IndexExpr:
		return recvName(t.X)
	case *ast.IndexListExpr:
		return recvName(t.X)
	case *ast.Ident:
		return t.Name
	}
	return "?"
}
