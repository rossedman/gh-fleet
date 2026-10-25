// Package gofiles walks Go packages for the repo's check tools, parsing
// each non-test file and telling hand-written files from generated ones.
package gofiles

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// File is one parsed Go source file.
type File struct {
	Path      string
	AST       *ast.File
	Fset      *token.FileSet
	Generated bool
}

// skipDir reports directories never walked.
func skipDir(name string) bool {
	return name == "vendor" || name == "testdata" || name == "node_modules" || (strings.HasPrefix(name, ".") && name != ".") || name == "site"
}

// Walk parses every non-test .go file under the given patterns. A pattern
// ending in /... is walked recursively; anything else is one directory.
func Walk(patterns []string) ([]File, error) {
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}
	var out []File
	fset := token.NewFileSet()
	for _, p := range patterns {
		root, recursive := strings.CutSuffix(p, "/...")
		if root == "" {
			root = "."
		}
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if path != root && (!recursive || skipDir(d.Name())) {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			src, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			f, err := parser.ParseFile(fset, path, src, parser.ParseComments)
			if err != nil {
				return err
			}
			out = append(out, File{Path: filepath.ToSlash(path), AST: f, Fset: fset, Generated: ast.IsGenerated(f)})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}
