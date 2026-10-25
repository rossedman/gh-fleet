package star

import (
	"fmt"
	"regexp"
	"strings"

	"go.starlark.net/syntax"
)

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
//
// Its docstring is a summary line, optional prose, then these sections,
// each a header line followed by lines indented four spaces:
//
//	Args:
//	    path: the file to edit, relative to the repo root.
//	Returns:
//	    The lines that were added.
//	Example:
//	    before .gitignore:
//	        node_modules
//	    call:
//	        ensure_lines(repo, ".gitignore", [".env"])
//	    after .gitignore:
//	        node_modules
//	        .env
//	    returns:
//	        [".env"]
//
// repo needs no Args entry; every other parameter does.
type LibFunc struct {
	Name string
	// Signature is the def line without "def" and the trailing colon.
	Signature string
	// Params are the parameter names, in order.
	Params []string
	// Line is the line of the def statement.
	Line int
	// Summary is the docstring's first line; Body the prose after it.
	Summary, Body string
	Args          []LibArg
	Returns       string
	Example       *LibExample
}

// LibArg documents one parameter.
type LibArg struct {
	Name, Doc string
}

// LibExample is a docstring example: files before, a call, and the files
// and value after. Tests run every example, so they cannot go stale.
type LibExample struct {
	// Campaign are files next to the campaign file.
	Campaign []LibFile
	Before   []LibFile
	// Call is one or more lines of Starlark; the last is an expression
	// whose value Returns describes.
	Call    string
	After   []LibFile
	Deleted []string
	// Returns is the repr of the call's value, or "" when not checked.
	Returns string
}

// LibFile is a file in an example. Content always ends in a newline
// unless it is empty.
type LibFile struct {
	Path, Content string
}

// ParseLibDoc reads the module docstring and the public functions, with
// their signatures and docstring sections, from a module's source.
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
		start, _ := def.Span()
		fn := LibFunc{Name: def.Name.Name, Line: int(start.Line), Signature: signature(lines, int(start.Line))}
		for _, p := range def.Params {
			fn.Params = append(fn.Params, paramName(p))
		}
		if len(def.Body) > 0 {
			if s, ok := docstring(def.Body[0]); ok {
				if err := fn.parseDoc(s); err != nil {
					return LibDoc{}, fmt.Errorf("%s: %s: %w", name, fn.Name, err)
				}
			}
		}
		doc.Funcs = append(doc.Funcs, fn)
	}
	return doc, nil
}

// signature joins a def statement's lines into one, without "def" and
// the trailing colon.
func signature(lines []string, line int) string {
	sig := ""
	for l := line - 1; l < len(lines); l++ {
		sig += strings.TrimSpace(lines[l])
		if strings.HasSuffix(sig, ":") {
			break
		}
		if strings.HasSuffix(sig, "(") || strings.HasSuffix(sig, ",") {
			sig += " "
		}
	}
	sig = strings.TrimSuffix(strings.TrimPrefix(sig, "def "), ":")
	return strings.ReplaceAll(strings.ReplaceAll(sig, "( ", "("), ", )", ")")
}

// paramName returns a parameter's name, without its default or star.
func paramName(p syntax.Expr) string {
	switch t := p.(type) {
	case *syntax.Ident:
		return t.Name
	case *syntax.BinaryExpr:
		return paramName(t.X)
	case *syntax.UnaryExpr:
		if t.X != nil {
			return paramName(t.X)
		}
	}
	return ""
}

// sectionHeader matches a docstring section header line.
var sectionHeader = regexp.MustCompile(`^(Args|Returns|Example):$`)

// parseDoc splits a function docstring into its summary, prose and
// sections.
func (fn *LibFunc) parseDoc(s string) error {
	lines := strings.Split(s, "\n")
	fn.Summary = strings.TrimSpace(lines[0])
	sections := map[string][]string{}
	current := ""
	var body []string
	for _, l := range lines[1:] {
		if m := sectionHeader.FindStringSubmatch(l); m != nil {
			if _, dup := sections[m[1]]; dup {
				return fmt.Errorf("%s: section repeated", m[1])
			}
			current = m[1]
			sections[current] = nil
			continue
		}
		if current == "" {
			body = append(body, l)
			continue
		}
		if strings.TrimSpace(l) != "" && !strings.HasPrefix(l, "    ") {
			return fmt.Errorf("%s: line %q is not indented four spaces", current, l)
		}
		sections[current] = append(sections[current], strings.TrimPrefix(l, "    "))
	}
	fn.Body = strings.TrimSpace(strings.Join(body, "\n"))
	for _, l := range sections["Args"] {
		switch {
		case strings.TrimSpace(l) == "":
		case strings.HasPrefix(l, " "):
			if len(fn.Args) == 0 {
				return fmt.Errorf("continuation line %q in Args comes before any argument", l)
			}
			fn.Args[len(fn.Args)-1].Doc += " " + strings.TrimSpace(l)
		default:
			name, text, ok := strings.Cut(l, ":")
			if !ok {
				return fmt.Errorf("want \"name: description\" in Args, got %q", l)
			}
			fn.Args = append(fn.Args, LibArg{Name: strings.TrimSpace(name), Doc: strings.TrimSpace(text)})
		}
	}
	fn.Returns = strings.TrimSpace(strings.Join(sections["Returns"], "\n"))
	if ex, ok := sections["Example"]; ok {
		e, err := parseExample(ex)
		if err != nil {
			return fmt.Errorf("in Example: %w", err)
		}
		fn.Example = e
	}
	return nil
}

// exampleDirective matches the lines that start a block of an example.
var exampleDirective = regexp.MustCompile(`^(campaign|before|after) (\S+):$|^(call|returns):$|^deleted (\S+)$`)

// parseExample reads an Example section: directives at the margin, each
// followed by its content indented four more spaces.
func parseExample(lines []string) (*LibExample, error) {
	e := &LibExample{}
	type block struct {
		kind, path string
		lines      []string
	}
	var blocks []*block
	for _, l := range lines {
		if m := exampleDirective.FindStringSubmatch(l); m != nil {
			switch {
			case m[1] != "":
				blocks = append(blocks, &block{kind: m[1], path: m[2]})
			case m[3] != "":
				blocks = append(blocks, &block{kind: m[3]})
			default:
				e.Deleted = append(e.Deleted, m[4])
			}
			continue
		}
		if strings.TrimSpace(l) == "" {
			if len(blocks) > 0 {
				b := blocks[len(blocks)-1]
				b.lines = append(b.lines, "")
			}
			continue
		}
		if len(blocks) == 0 || !strings.HasPrefix(l, "    ") {
			return nil, fmt.Errorf("line %q is not inside a block; start one with before <path>:, call:, after <path>:, returns: or deleted <path>", l)
		}
		b := blocks[len(blocks)-1]
		b.lines = append(b.lines, strings.TrimPrefix(l, "    "))
	}
	for _, b := range blocks {
		text := strings.TrimRight(strings.Join(b.lines, "\n"), "\n ")
		file := LibFile{Path: b.path, Content: text}
		if text != "" {
			file.Content += "\n"
		}
		switch b.kind {
		case "campaign":
			e.Campaign = append(e.Campaign, file)
		case "before":
			e.Before = append(e.Before, file)
		case "after":
			e.After = append(e.After, file)
		case "call":
			e.Call = text
		case "returns":
			e.Returns = text
		}
	}
	if e.Call == "" {
		return nil, fmt.Errorf("no call: block")
	}
	return e, nil
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
