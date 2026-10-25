// Command gendocs writes the generated reference pages of the docs site:
// one Markdown page per command from the cobra tree, the campaign file
// reference from the embedded JSON Schema, and the Starlark library
// reference from the library's docstrings, so the docs never drift from
// the binary.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra/doc"

	"github.com/rossedman/gh-fleet/cmd"
	"github.com/rossedman/gh-fleet/internal/campaign"
	"github.com/rossedman/gh-fleet/internal/star"
)

// main writes both references under docs/reference, or the directory given
// as the first argument.
func main() {
	out := "docs/reference"
	if len(os.Args) > 1 {
		out = os.Args[1]
	}
	if err := run(out); err != nil {
		fmt.Fprintln(os.Stderr, "gendocs:", err)
		os.Exit(1) //nolint:revive // main may exit
	}
}

// run generates every page into out.
func run(out string) error {
	cmdDir := filepath.Join(out, "commands")
	if err := os.RemoveAll(cmdDir); err != nil {
		return err
	}
	if err := os.MkdirAll(cmdDir, 0o755); err != nil {
		return err
	}
	root := cmd.Root()
	root.DisableAutoGenTag = true
	if err := doc.GenMarkdownTree(root, cmdDir); err != nil {
		return err
	}
	if err := promoteTitles(cmdDir); err != nil {
		return err
	}
	page, err := SchemaMarkdown(campaign.SchemaJSON)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(out, "campaign.md"), page, 0o644); err != nil { //nolint:gosec // docs are public
		return err
	}
	lib, err := LibMarkdown()
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(out, "starlark-library.md"), lib, 0o644) //nolint:gosec // docs are public
}

// LibMarkdown renders the reference for the built-in Starlark library:
// an index of modules, then each module's functions with their
// signatures and docstrings.
func LibMarkdown() ([]byte, error) {
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
	var b bytes.Buffer
	b.WriteString("---\ntitle: Starlark library reference\n---\n\n")
	b.WriteString("# Starlark library reference\n\n")
	b.WriteString("The helpers built into gh fleet, generated from their docstrings. ")
	b.WriteString("Load them by module, as in `load(\"" + star.LibPrefix + "files.star\", \"ensure_lines\")`; ")
	b.WriteString("they need no network and are versioned with the binary. ")
	b.WriteString("See [Starlark changes](../guides/starlark.md) for how scripts run.\n\n")
	b.WriteString("| Module | What it does |\n| --- | --- |\n")
	for _, d := range docs {
		fmt.Fprintf(&b, "| [`%s%s`](#%s) | %s |\n", star.LibPrefix, d.Name, anchor(d.Name), cell(d.Summary))
	}
	b.WriteString("\n")
	for _, d := range docs {
		fmt.Fprintf(&b, "## %s\n\n", d.Name)
		b.WriteString(d.Summary + "\n\n")
		if d.Body != "" {
			b.WriteString(d.Body + "\n\n")
		}
		names := make([]string, len(d.Funcs))
		for i, f := range d.Funcs {
			names[i] = `"` + f.Name + `"`
		}
		fmt.Fprintf(&b, "```python\nload(\"%s%s\", %s)\n```\n\n", star.LibPrefix, d.Name, strings.Join(names, ", "))
		for _, f := range d.Funcs {
			fmt.Fprintf(&b, "### %s.%s\n\n```python\n%s\n```\n\n%s\n\n", strings.TrimSuffix(d.Name, ".star"), f.Name, f.Signature, f.Doc)
		}
	}
	return append(bytes.TrimRight(b.Bytes(), "\n"), '\n'), nil
}

// anchor is the heading id mkdocs gives a module section.
func anchor(name string) string {
	return strings.ReplaceAll(name, ".", "")
}

// promoteTitles turns the "## gh fleet x" line cobra writes first into a
// page title, so each command page has an h1.
func promoteTitles(dir string) error {
	pages, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil {
		return err
	}
	for _, p := range pages {
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if rest, ok := bytes.CutPrefix(b, []byte("## ")); ok {
			if err := os.WriteFile(p, append([]byte("# "), rest...), 0o644); err != nil { //nolint:gosec // docs are public
				return err
			}
		}
	}
	return nil
}

// schemaNode is the subset of JSON Schema the reference renders.
type schemaNode struct {
	Description          string                 `json:"description"`
	Type                 schemaType             `json:"type"`
	Const                any                    `json:"const"`
	Default              any                    `json:"default"`
	Pattern              string                 `json:"pattern"`
	Minimum              *float64               `json:"minimum"`
	Maximum              *float64               `json:"maximum"`
	Required             []string               `json:"required"`
	Properties           map[string]*schemaNode `json:"properties"`
	Items                *schemaNode            `json:"items"`
	Ref                  string                 `json:"$ref"`
	AdditionalProperties any                    `json:"additionalProperties"`
	Defs                 map[string]*schemaNode `json:"$defs"`
}

// schemaType is a JSON Schema "type", which may be one name or a list of
// alternatives; a list is kept as "a or b".
type schemaType string

// UnmarshalJSON accepts a string or a list of strings.
func (t *schemaType) UnmarshalJSON(b []byte) error {
	var one string
	if json.Unmarshal(b, &one) == nil {
		*t = schemaType(one)
		return nil
	}
	var many []string
	if err := json.Unmarshal(b, &many); err != nil {
		return err
	}
	*t = schemaType(strings.Join(many, " or "))
	return nil
}

// refTypes names the shared definitions that read better as one phrase
// than as their structure.
var refTypes = map[string]string{
	"#/$defs/propertyFilter": "map of property to value or list",
	"#/$defs/varMap":         "map of name to value or list",
}

// order lists top-level keys in the order a campaign file uses them.
var order = []string{"apiVersion", "name", "description", "targets", "change", "pr", "vars", "limits"}

// SchemaMarkdown renders the campaign schema as a reference page with one
// table per object.
func SchemaMarkdown(schemaJSON []byte) ([]byte, error) {
	var root schemaNode
	if err := json.Unmarshal(schemaJSON, &root); err != nil {
		return nil, err
	}
	var b bytes.Buffer
	b.WriteString("---\ntitle: Campaign file reference\n---\n\n")
	b.WriteString("# Campaign file reference\n\n")
	b.WriteString("Every field of a campaign file, generated from the JSON Schema shipped in the binary. ")
	b.WriteString("Editors that understand `yaml-language-server` validate and complete files written by `gh fleet init`.\n\n")
	b.WriteString(root.Description + "\n\n")
	b.WriteString("Schema URL: `" + campaign.SchemaURL + "`\n\n")
	writeObject(&b, "Top level", "", &root, root.Defs, order)
	return b.Bytes(), nil
}

// writeObject writes a table for one object, then sections for its
// object-typed children.
func writeObject(b *bytes.Buffer, title, prefix string, n *schemaNode, defs map[string]*schemaNode, keys []string) {
	if title != "Top level" {
		fmt.Fprintf(b, "## `%s`\n\n", prefix)
		if n.Description != "" {
			b.WriteString(n.Description + "\n\n")
		}
	} else {
		b.WriteString("## Top level\n\n")
	}
	if keys == nil {
		for k := range n.Properties {
			keys = append(keys, k)
		}
		sort.Strings(keys)
	}
	required := map[string]bool{}
	for _, r := range n.Required {
		required[r] = true
	}
	b.WriteString("| Field | Type | Default | Description |\n| --- | --- | --- | --- |\n")
	var children []string
	for _, k := range keys {
		p := n.Properties[k]
		if p == nil {
			continue
		}
		typ, named := refTypes[p.Ref]
		p = resolve(p, defs)
		if !named {
			typ = typeOf(p, defs)
		}
		name := "`" + k + "`"
		if required[k] {
			name += " (required)"
		}
		fmt.Fprintf(b, "| %s | %s | %s | %s |\n", name, typ, defaultOf(p), cell(describe(p)))
		switch {
		case p.Type == "object" && len(p.Properties) > 0:
			children = append(children, k)
		case p.Type == "array" && p.Items != nil && len(resolve(p.Items, defs).Properties) > 0:
			children = append(children, k+"[]")
		}
	}
	b.WriteString("\n")
	for _, k := range children {
		path := k
		if prefix != "" {
			path = prefix + "." + k
		}
		child := n.Properties[strings.TrimSuffix(k, "[]")]
		node := resolve(child, defs)
		if strings.HasSuffix(k, "[]") {
			items := *resolve(node.Items, defs)
			if items.Description == "" {
				items.Description = "Each entry of `" + strings.TrimSuffix(path, "[]") + "`."
			}
			node = &items
		}
		writeObject(b, k, path, node, defs, nil)
	}
}

// resolve follows a local $ref into $defs, keeping the referring node's
// description when it has one.
func resolve(n *schemaNode, defs map[string]*schemaNode) *schemaNode {
	if name, ok := strings.CutPrefix(n.Ref, "#/$defs/"); ok && defs[name] != nil {
		d := *defs[name]
		if n.Description != "" {
			d.Description = n.Description
		}
		return &d
	}
	return n
}

// typeOf names a node's type, with list item types.
func typeOf(n *schemaNode, defs map[string]*schemaNode) string {
	switch {
	case n.Const != nil:
		return "string"
	case n.Type == "array" && n.Items != nil:
		return "list of " + typeOf(resolve(n.Items, defs), defs)
	case n.Type == "object" && len(n.Properties) == 0 && n.AdditionalProperties != nil:
		return "map of string"
	case n.Type == "string" && n.Pattern == `^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`:
		return "`owner/name`"
	case n.Type == "":
		return "-"
	}
	return string(n.Type)
}

// defaultOf formats a node's default value.
func defaultOf(n *schemaNode) string {
	switch {
	case n.Const != nil:
		return fmt.Sprintf("`%v`", n.Const)
	case n.Default == nil:
		return ""
	case n.Default == "":
		return `""`
	}
	return fmt.Sprintf("`%v`", n.Default)
}

// describe returns a node's description plus its value constraints.
func describe(n *schemaNode) string {
	d := n.Description
	if n.Const != nil {
		d += fmt.Sprintf(" Must be `%v`.", n.Const)
	}
	if n.Minimum != nil && n.Maximum != nil {
		d += fmt.Sprintf(" Between %v and %v.", *n.Minimum, *n.Maximum)
	} else if n.Minimum != nil {
		d += fmt.Sprintf(" At least %v.", *n.Minimum)
	}
	return strings.TrimSpace(d)
}

// cell escapes text for a Markdown table cell.
func cell(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "|", `\|`), "\n", " ")
}
