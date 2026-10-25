package star

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"go.starlark.net/syntax"
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

// LibDoc is the documentation of one built-in module, taken from its
// docstrings.
type LibDoc struct {
	// Name is the module's file name, such as files.star.
	Name string
	// Summary is the first line of the module docstring; Body the rest.
	Summary, Body string
	Funcs         []LibFunc
}

// LibFunc documents one public function of a built-in module.
type LibFunc struct {
	Name string
	// Signature is the def line without "def" and the trailing colon.
	Signature string
	Doc       string
}

// ParseLibDoc reads the module docstring and the public functions, with
// their signatures and docstrings, from a module's source.
func ParseLibDoc(name string, src []byte) (LibDoc, error) {
	f, err := fileOptions.Parse(name, src, syntax.RetainComments)
	if err != nil {
		return LibDoc{}, err
	}
	doc := LibDoc{Name: name}
	lines := strings.Split(string(src), "\n")
	for i, stmt := range f.Stmts {
		if s, ok := docstring(stmt); ok && i == 0 {
			doc.Summary, doc.Body, _ = strings.Cut(s, "\n")
			doc.Body = strings.TrimSpace(doc.Body)
			continue
		}
		def, ok := stmt.(*syntax.DefStmt)
		if !ok || strings.HasPrefix(def.Name.Name, "_") {
			continue
		}
		fn := LibFunc{Name: def.Name.Name}
		start, _ := def.Span()
		sig := ""
		for l := int(start.Line) - 1; l < len(lines); l++ {
			sig += strings.TrimSpace(lines[l])
			if strings.HasSuffix(sig, ":") {
				break
			}
			if strings.HasSuffix(sig, "(") || strings.HasSuffix(sig, ",") {
				sig += " "
			}
		}
		sig = strings.TrimSuffix(strings.TrimPrefix(sig, "def "), ":")
		fn.Signature = strings.ReplaceAll(strings.ReplaceAll(sig, "( ", "("), ", )", ")")
		if len(def.Body) > 0 {
			if s, ok := docstring(def.Body[0]); ok {
				fn.Doc = s
			}
		}
		doc.Funcs = append(doc.Funcs, fn)
	}
	return doc, nil
}

// docstring returns the dedented text of a string-literal statement.
func docstring(stmt syntax.Stmt) (string, bool) {
	e, ok := stmt.(*syntax.ExprStmt)
	if !ok {
		return "", false
	}
	lit, ok := e.X.(*syntax.Literal)
	if !ok || lit.Token != syntax.STRING {
		return "", false
	}
	s, _ := lit.Value.(string)
	return dedent(s), true
}

// dedent strips the common indentation of every line after the first and
// trims the result.
func dedent(s string) string {
	lines := strings.Split(s, "\n")
	indent := -1
	for _, l := range lines[1:] {
		if strings.TrimSpace(l) == "" {
			continue
		}
		n := len(l) - len(strings.TrimLeft(l, " "))
		if indent < 0 || n < indent {
			indent = n
		}
	}
	for i := 1; i < len(lines); i++ {
		if len(lines[i]) >= indent && indent > 0 {
			lines[i] = lines[i][indent:]
		} else {
			lines[i] = strings.TrimSpace(lines[i])
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}
