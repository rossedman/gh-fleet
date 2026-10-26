package main

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"go.starlark.net/syntax"

	"github.com/rossedman/gh-fleet/internal/star"
)

// Links the library pages point at.
const (
	repoURL   = "https://github.com/rossedman/gh-fleet"
	sourceURL = repoURL + "/blob/main/internal/star/lib/"
)

// Markers around the generated Starlark library entries in mkdocs.yml.
const (
	navBegin = "# gendocs:starlark-library:begin"
	navEnd   = "# gendocs:starlark-library:end"
)

// libDocs parses every built-in module.
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

// writeLib writes the library index to out/starlark-library.md, a page per
// module under out/starlark/, and the matching nav entries into nav. used
// maps "module.star/function" to the examples that call it.
func writeLib(out, nav string, used map[string][]string) error {
	docs, err := libDocs()
	if err != nil {
		return err
	}
	dir := filepath.Join(out, "starlark")
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(out, "starlark-library.md"), LibIndex(docs), 0o644); err != nil { //nolint:gosec // docs are public
		return err
	}
	for _, d := range docs {
		page := filepath.Join(dir, strings.TrimSuffix(d.Name, ".star")+".md")
		if err := os.WriteFile(page, LibPage(d, used), 0o644); err != nil { //nolint:gosec // docs are public
			return err
		}
	}
	if nav == "" {
		return nil
	}
	entries := []string{"Overview: reference/starlark-library.md"}
	for _, d := range docs {
		entries = append(entries, fmt.Sprintf("%s: reference/starlark/%s.md", d.Name, module(d)))
	}
	return updateNavBlock(nav, navBegin, navEnd, entries)
}

// module is a module's name without .star.
func module(d star.LibDoc) string {
	return strings.TrimSuffix(d.Name, ".star")
}

// LibIndex renders the library overview: how helpers behave, a table of
// modules, and an index of every function.
func LibIndex(docs []star.LibDoc) []byte {
	var b bytes.Buffer
	b.WriteString("---\ntitle: Starlark library\n---\n\n# Starlark library\n\n")
	b.WriteString("Helpers for the changes campaigns make most, built into gh fleet. ")
	b.WriteString("They need no network and are versioned with the binary. Load them by module:\n\n")
	fmt.Fprintf(&b, "```python\nload(\"%sfiles.star\", \"ensure_lines\")\n\ndef change(repo):\n    ensure_lines(repo, \".gitignore\", [\".env\"])\n```\n\n", star.LibPrefix)
	b.WriteString("Every helper:\n\n")
	b.WriteString("- takes `repo`, the value passed to `change(repo)`, as its first argument;\n")
	b.WriteString("- is **idempotent**: run it on a repo that already has the change and it writes nothing, so the repo comes out as a no-op;\n")
	b.WriteString("- edits files **line by line** where a parse and re-encode would lose comments or layout;\n")
	b.WriteString("- **returns what it changed**, usually a list of paths, so the script can tell the PR about it with `repo.set_var`.\n\n")
	b.WriteString("This reference is generated from the library's docstrings, and every example on these pages runs as a test, ")
	b.WriteString("so it always matches the code. See [Starlark changes](../guides/starlark.md) for how scripts run.\n\n")
	b.WriteString("## Modules\n\n| Module | What it does | Functions |\n| --- | --- | --- |\n")
	for _, d := range docs {
		fmt.Fprintf(&b, "| [`%s`](starlark/%s.md) | %s | %d |\n", d.Name, module(d), cell(d.Summary), len(d.Funcs))
	}
	b.WriteString("\n## All functions\n\n| Function | What it does |\n| --- | --- |\n")
	type entry struct{ mod, name, summary string }
	var all []entry
	for _, d := range docs {
		for _, f := range d.Funcs {
			all = append(all, entry{module(d), f.Name, f.Summary})
		}
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].name != all[j].name {
			return all[i].name < all[j].name
		}
		return all[i].mod < all[j].mod
	})
	for _, e := range all {
		fmt.Fprintf(&b, "| [`%s.%s`](starlark/%s.md#%s) | %s |\n", e.mod, e.name, e.mod, e.name, cell(e.summary))
	}
	return b.Bytes()
}

// LibPage renders one module's page. used maps "module.star/function" to
// the example campaigns that call it.
func LibPage(d star.LibDoc, used map[string][]string) []byte {
	var b bytes.Buffer
	fmt.Fprintf(&b, "---\ntitle: %s\n---\n\n# %s\n\n%s\n\n", d.Name, d.Name, d.Summary)
	if d.Body != "" {
		b.WriteString(d.Body + "\n\n")
	}
	names := make([]string, len(d.Funcs))
	for i, f := range d.Funcs {
		names[i] = `"` + f.Name + `"`
	}
	fmt.Fprintf(&b, "```python\nload(\"%s%s\", %s)\n```\n\n", star.LibPrefix, d.Name, strings.Join(names, ", "))
	b.WriteString("| Function | What it does |\n| --- | --- |\n")
	for _, f := range d.Funcs {
		fmt.Fprintf(&b, "| [`%s`](#%s) | %s |\n", f.Name, f.Name, cell(f.Summary))
	}
	b.WriteString("\n")
	for _, f := range d.Funcs {
		writeFunc(&b, d, f, used[d.Name+"/"+f.Name])
	}
	return append(bytes.TrimRight(b.Bytes(), "\n"), '\n')
}

// writeFunc renders one function's section.
func writeFunc(b *bytes.Buffer, d star.LibDoc, f star.LibFunc, used []string) {
	fmt.Fprintf(b, "## %s\n\n```python\n%s\n```\n\n%s\n\n", f.Name, wrapSignature(f.Signature), f.Summary)
	if f.Body != "" {
		b.WriteString(f.Body + "\n\n")
	}
	if len(f.Params) > 0 {
		b.WriteString("**Arguments**\n\n| Name | Description |\n| --- | --- |\n")
		docs := map[string]string{"repo": "The repo passed to `change(repo)`."}
		for _, a := range f.Args {
			docs[a.Name] = upperFirst(sentence(a.Doc))
		}
		for _, p := range f.Params {
			fmt.Fprintf(b, "| `%s` | %s |\n", p, cell(docs[p]))
		}
		b.WriteString("\n")
	}
	if f.Returns != "" {
		fmt.Fprintf(b, "**Returns:** %s\n\n", sentence(f.Returns))
	}
	if e := f.Example; e != nil {
		writeExample(b, e)
	}
	if len(used) > 0 {
		links := make([]string, len(used))
		for i, u := range used {
			links[i] = fmt.Sprintf("[`%s`](../examples/%s.md)", u, u)
		}
		fmt.Fprintf(b, "**Used in:** %s\n\n", strings.Join(links, ", "))
	}
	fmt.Fprintf(b, "[Source](%s%s#L%d)\n\n", sourceURL, d.Name, f.Line)
}

// writeExample renders an example as the call, then tabs with the files
// before and after it.
func writeExample(b *bytes.Buffer, e *star.LibExample) {
	b.WriteString("**Example**\n\n")
	code := e.Call
	if e.Returns != "" {
		code += "\n# returns " + e.Returns
	}
	b.WriteString(fence("python", code, ""))
	b.WriteString("\n")
	if len(e.Campaign)+len(e.Before) > 0 {
		b.WriteString("=== \"Before\"\n\n")
		for _, f := range e.Campaign {
			fmt.Fprintf(b, "    `%s` next to the campaign file\n\n", f.Path)
			b.WriteString(fence(lang(f.Path), f.Content, "    "))
			b.WriteString("\n")
		}
		for _, f := range e.Before {
			fmt.Fprintf(b, "    `%s`\n\n", f.Path)
			b.WriteString(fence(lang(f.Path), f.Content, "    "))
			b.WriteString("\n")
		}
	}
	if len(e.After)+len(e.Deleted) > 0 {
		b.WriteString("=== \"After\"\n\n")
		for _, f := range e.After {
			fmt.Fprintf(b, "    `%s`\n\n", f.Path)
			b.WriteString(fence(lang(f.Path), f.Content, "    "))
			b.WriteString("\n")
		}
		for _, p := range e.Deleted {
			fmt.Fprintf(b, "    `%s` is deleted.\n\n", p)
		}
	}
}

// fence wraps content in a code fence long enough not to clash with it,
// indenting every line.
func fence(lang, content, indent string) string {
	ticks := "```"
	for strings.Contains(content, ticks) {
		ticks += "`"
	}
	var b strings.Builder
	b.WriteString(indent + ticks + lang + "\n")
	for _, l := range strings.Split(strings.TrimRight(content, "\n"), "\n") {
		if l == "" {
			b.WriteString("\n")
			continue
		}
		b.WriteString(indent + l + "\n")
	}
	b.WriteString(indent + ticks + "\n")
	return b.String()
}

// lang picks a code fence language from a file name.
func lang(p string) string {
	base := path.Base(p)
	switch {
	case base == "Dockerfile" || strings.HasPrefix(base, "Dockerfile.") || strings.HasSuffix(strings.ToLower(base), ".dockerfile"):
		return "dockerfile"
	case base == "CODEOWNERS" || base == "go.mod" || base == ".gitignore" || strings.HasPrefix(base, ".") && path.Ext(base) == "":
		return "text"
	}
	switch path.Ext(base) {
	case ".yml", ".yaml":
		return "yaml"
	case ".json":
		return "json"
	case ".toml":
		return "toml"
	case ".tf":
		return "hcl"
	case ".md":
		return "markdown"
	case ".star":
		return "python"
	case ".sh":
		return "bash"
	case ".csv":
		return "text"
	case ".ts":
		return "typescript"
	}
	return "text"
}

// sentence ends text with a full stop.
func sentence(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || strings.HasSuffix(s, ".") {
		return s
	}
	return s + "."
}

// wrapSignature puts each parameter of a signature longer than 72
// characters on its own line, so it fits the page without scrolling.
func wrapSignature(sig string) string {
	open := strings.Index(sig, "(")
	if len(sig) <= 72 || open < 0 || !strings.HasSuffix(sig, ")") {
		return sig
	}
	var params []string
	depth, start := 0, open+1
	inner := sig[:len(sig)-1]
	for i := start; i < len(inner); i++ {
		switch inner[i] {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
		case ',':
			if depth == 0 {
				params = append(params, strings.TrimSpace(inner[start:i]))
				start = i + 1
			}
		}
	}
	params = append(params, strings.TrimSpace(inner[start:]))
	return sig[:open+1] + "\n    " + strings.Join(params, ",\n    ") + ",\n)"
}

// upperFirst capitalizes the first letter.
func upperFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// usedIn maps "module.star/function" to the example campaigns whose
// Starlark loads it from the built-in library.
func usedIn(dir string) (map[string][]string, error) {
	out := map[string][]string{}
	scripts, err := filepath.Glob(filepath.Join(dir, "*", "*.star"))
	if err != nil {
		return nil, err
	}
	opts := &syntax.FileOptions{Set: true, While: true, TopLevelControl: true}
	for _, p := range scripts {
		f, err := opts.Parse(p, nil, 0)
		if err != nil {
			return nil, err
		}
		example := filepath.Base(filepath.Dir(p))
		for _, stmt := range f.Stmts {
			load, ok := stmt.(*syntax.LoadStmt)
			if !ok || !strings.HasPrefix(load.ModuleName(), star.LibPrefix) {
				continue
			}
			mod := strings.TrimPrefix(load.ModuleName(), star.LibPrefix)
			for _, id := range load.From {
				key := mod + "/" + id.Name
				if !contains(out[key], example) {
					out[key] = append(out[key], example)
				}
			}
		}
	}
	for k := range out {
		sort.Strings(out[k])
	}
	return out, nil
}

// contains reports whether list holds s.
func contains(list []string, s string) bool {
	for _, e := range list {
		if e == s {
			return true
		}
	}
	return false
}

// updateNavBlock rewrites the lines between the begin and end markers in
// mkdocs.yml with one "- entry" line each, indented like the begin marker.
func updateNavBlock(nav, navBegin, navEnd string, entries []string) error {
	b, err := os.ReadFile(nav)
	if err != nil {
		return err
	}
	lines := strings.Split(string(b), "\n")
	begin, end := -1, -1
	for i, l := range lines {
		switch strings.TrimSpace(l) {
		case navBegin:
			begin = i
		case navEnd:
			end = i
		}
	}
	if begin < 0 || end < begin {
		return fmt.Errorf("%s: no %q ... %q block", nav, navBegin, navEnd)
	}
	indent := lines[begin][:len(lines[begin])-len(strings.TrimLeft(lines[begin], " "))]
	block := make([]string, len(entries))
	for i, e := range entries {
		block[i] = indent + "- " + e
	}
	out := append(append(append([]string{}, lines[:begin+1]...), block...), lines[end:]...)
	updated := strings.Join(out, "\n")
	if updated == string(b) {
		return nil
	}
	return os.WriteFile(nav, []byte(updated), 0o644) //nolint:gosec // docs config is public
}
