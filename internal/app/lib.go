package app

import (
	"fmt"
	"sort"
	"strings"

	"github.com/rossedman/gh-fleet/internal/star"
)

// LibOptions configures Lib.
type LibOptions struct {
	Common
	// Args names a module, a module and a function, or module.function.
	Args []string
	// Search lists the functions whose names or docs mention these words.
	Search string
	// Source prints a module's source instead of its docs.
	Source bool
}

// LibModule is one row of the module list.
type LibModule struct {
	Module  string `json:"module"`
	Summary string `json:"summary"`
}

// LibHit is one search result.
type LibHit struct {
	Module   string `json:"module"`
	Function string `json:"function"`
	Summary  string `json:"summary"`
}

// libDocs parses every built-in module's docs.
func libDocs() ([]star.LibDoc, error) {
	var docs []star.LibDoc
	for _, name := range star.LibModules() {
		src, err := star.LibSource(name)
		if err != nil {
			return nil, err
		}
		d, err := star.ParseLibDoc(name, src)
		if err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}
	return docs, nil
}

// Lib prints the built-in Starlark library's docs: the module list, one
// module, one function, a search, or a module's source.
func (a *App) Lib(o LibOptions) error {
	docs, err := libDocs()
	if err != nil {
		return err
	}
	p := a.printer(o.Common)
	if o.Search != "" {
		if len(o.Args) > 0 {
			return fmt.Errorf("--search takes no module")
		}
		hits := searchLib(docs, o.Search)
		if p.Machine() {
			return p.PrintJSON(hits)
		}
		if len(hits) == 0 {
			return fmt.Errorf("nothing in the library mentions %q", o.Search)
		}
		rows := make([][]string, len(hits))
		for i, h := range hits {
			rows[i] = []string{strings.TrimSuffix(h.Module, ".star") + "." + h.Function, h.Summary}
		}
		p.Table([]string{"FUNCTION", "WHAT IT DOES"}, rows, nil)
		return nil
	}
	if len(o.Args) == 0 {
		if o.Source {
			return fmt.Errorf("--source needs a module")
		}
		mods := make([]LibModule, len(docs))
		rows := make([][]string, len(docs))
		for i, d := range docs {
			mods[i] = LibModule{Module: d.Name, Summary: d.Summary}
			rows[i] = []string{strings.TrimSuffix(d.Name, ".star"), d.Summary}
		}
		if p.Machine() {
			return p.PrintJSON(mods)
		}
		p.Table([]string{"MODULE", "WHAT IT DOES"}, rows, nil)
		fmt.Fprintf(a.Err, "\n%d modules. gh fleet lib <module> lists its functions; gh fleet lib --search <words> finds one.\n", len(docs))
		return nil
	}
	mod, fn, err := libTarget(o.Args)
	if err != nil {
		return err
	}
	d, err := findModule(docs, mod)
	if err != nil {
		return err
	}
	if o.Source {
		if fn != "" {
			return fmt.Errorf("--source prints a whole module; drop .%s", fn)
		}
		src, err := star.LibSource(d.Name)
		if err != nil {
			return err
		}
		_, err = a.Out.Write(src)
		return err
	}
	if fn == "" {
		if p.Machine() {
			return p.PrintJSON(d)
		}
		a.printModule(p.Styles.Header.Render, p.Styles.Dim.Render, d)
		return nil
	}
	for _, f := range d.Funcs {
		if f.Name == fn {
			if p.Machine() {
				return p.PrintJSON(f)
			}
			a.printFunc(p.Styles.Header.Render, p.Styles.Dim.Render, d, f)
			return nil
		}
	}
	names := make([]string, len(d.Funcs))
	for i, f := range d.Funcs {
		names[i] = f.Name
	}
	return fmt.Errorf("no function %s in %s; it has %s", fn, d.Name, strings.Join(names, ", "))
}

// libTarget splits "docker", "docker.star", "docker.bump_image" or
// "docker bump_image" into a module and a function.
func libTarget(args []string) (string, string, error) {
	if len(args) > 2 {
		return "", "", fmt.Errorf("want a module and at most one function, got %d arguments", len(args))
	}
	mod := strings.TrimPrefix(args[0], star.LibPrefix)
	fn := ""
	if len(args) == 2 {
		fn = args[1]
	}
	if m, f, ok := strings.Cut(strings.Replace(mod, ".star", "", 1), "."); ok {
		if fn != "" {
			return "", "", fmt.Errorf("name the function once: %s or %s %s", args[0], args[0], args[1])
		}
		mod, fn = m, f
	}
	return strings.TrimSuffix(mod, ".star"), fn, nil
}

// findModule finds a module by name, suggesting near names when it
// doesn't exist.
func findModule(docs []star.LibDoc, mod string) (star.LibDoc, error) {
	var near []string
	for _, d := range docs {
		name := strings.TrimSuffix(d.Name, ".star")
		if strings.EqualFold(name, mod) {
			return d, nil
		}
		want, typos := strings.ToLower(mod), 1
		if len(want) >= 5 {
			typos = 2
		}
		if strings.HasPrefix(name, want) || editDistance(name, want) <= typos {
			near = append(near, name)
		}
	}
	if len(near) > 0 {
		return star.LibDoc{}, fmt.Errorf("no module %s in the library; close names: %s", mod, strings.Join(near, ", "))
	}
	return star.LibDoc{}, fmt.Errorf("no module %s in the library; gh fleet lib lists them", mod)
}

// searchLib ranks functions by how well they match every word of query:
// in the function name first, then the summary, then the rest of the
// docs.
func searchLib(docs []star.LibDoc, query string) []LibHit {
	words := strings.Fields(strings.ToLower(query))
	type scored struct {
		hit   LibHit
		score int
	}
	var all []scored
	for _, d := range docs {
		mod := strings.ToLower(strings.TrimSuffix(d.Name, ".star"))
		for _, f := range d.Funcs {
			name := mod + "." + strings.ToLower(f.Name)
			summary := strings.ToLower(f.Summary)
			rest := strings.ToLower(d.Summary + " " + f.Body + " " + f.Returns)
			for _, arg := range f.Args {
				rest += " " + strings.ToLower(arg.Name+" "+arg.Doc)
			}
			score := 0
			for _, w := range words {
				switch {
				case strings.Contains(name, w):
					score += 3
				case strings.Contains(summary, w):
					score += 2
				case strings.Contains(rest, w):
					score++
				default:
					score = -1 << 30
				}
			}
			if score > 0 {
				all = append(all, scored{LibHit{Module: d.Name, Function: f.Name, Summary: f.Summary}, score})
			}
		}
	}
	sort.SliceStable(all, func(i, j int) bool {
		if all[i].score != all[j].score {
			return all[i].score > all[j].score
		}
		return all[i].hit.Module+all[i].hit.Function < all[j].hit.Module+all[j].hit.Function
	})
	hits := make([]LibHit, len(all))
	for i, s := range all {
		hits[i] = s.hit
	}
	return hits
}

// printModule prints a module's summary, prose, load line and functions.
func (a *App) printModule(header, dim func(...string) string, d star.LibDoc) {
	fmt.Fprintln(a.Out, header(star.LibPrefix+d.Name))
	fmt.Fprintln(a.Out)
	fmt.Fprintln(a.Out, d.Summary)
	if d.Body != "" {
		fmt.Fprintln(a.Out)
		fmt.Fprintln(a.Out, d.Body)
	}
	names := make([]string, len(d.Funcs))
	for i, f := range d.Funcs {
		names[i] = fmt.Sprintf("%q", f.Name)
	}
	fmt.Fprintf(a.Out, "\n    load(%q, %s)\n", star.LibPrefix+d.Name, strings.Join(names, ", "))
	for _, f := range d.Funcs {
		fmt.Fprintf(a.Out, "\n%s\n    %s\n", f.Signature, f.Summary)
	}
	fmt.Fprintf(a.Out, "\n%s\n", dim(fmt.Sprintf("gh fleet lib %s.<function> shows a function's arguments and example.", strings.TrimSuffix(d.Name, ".star"))))
}

// printFunc prints a function's signature, docs and example.
func (a *App) printFunc(header, dim func(...string) string, d star.LibDoc, f star.LibFunc) {
	fmt.Fprintln(a.Out, header(f.Signature))
	fmt.Fprintf(a.Out, "%s\n\n", dim(fmt.Sprintf("load(%q, %q)", star.LibPrefix+d.Name, f.Name)))
	fmt.Fprintln(a.Out, f.Summary)
	if f.Body != "" {
		fmt.Fprintf(a.Out, "\n%s\n", f.Body)
	}
	if len(f.Args) > 0 {
		fmt.Fprintf(a.Out, "\n%s\n", header("Arguments"))
		for _, arg := range f.Args {
			fmt.Fprintf(a.Out, "  %s: %s\n", arg.Name, indentRest(arg.Doc, "    "))
		}
	}
	if f.Returns != "" {
		fmt.Fprintf(a.Out, "\n%s\n  %s\n", header("Returns"), indentRest(f.Returns, "  "))
	}
	if e := f.Example; e != nil {
		fmt.Fprintf(a.Out, "\n%s\n", header("Example"))
		files := func(label string, fs []star.LibFile) {
			for _, file := range fs {
				fmt.Fprintf(a.Out, "\n  %s\n", dim(label+" "+file.Path+":"))
				fmt.Fprintln(a.Out, indent(strings.TrimSuffix(file.Content, "\n"), "    "))
			}
		}
		files("campaign file", e.Campaign)
		files("before", e.Before)
		fmt.Fprintf(a.Out, "\n  %s\n%s\n", dim("call:"), indent(e.Call, "    "))
		files("after", e.After)
		for _, path := range e.Deleted {
			fmt.Fprintf(a.Out, "\n  %s\n", dim("deleted "+path))
		}
		if e.Returns != "" {
			fmt.Fprintf(a.Out, "\n  %s\n    %s\n", dim("returns:"), e.Returns)
		}
	}
}

// indent prefixes every line of s.
func indent(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		if l != "" {
			lines[i] = prefix + l
		}
	}
	return strings.Join(lines, "\n")
}

// indentRest indents every line of s but the first.
func indentRest(s, prefix string) string {
	first, rest, ok := strings.Cut(s, "\n")
	if !ok {
		return s
	}
	return first + "\n" + indent(rest, prefix)
}

// editDistance is the Levenshtein distance between a and b, for
// suggesting a module name after a typo.
func editDistance(a, b string) int {
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur := make([]int, len(b)+1)
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(b)]
}
