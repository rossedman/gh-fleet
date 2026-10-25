package campaign

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
	"go.yaml.in/yaml/v3"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// SchemaJSON is the JSON Schema campaign files are validated against.
//
//go:embed campaign.schema.json
var SchemaJSON []byte

// SchemaURL is where editors fetch the schema from, written into new
// campaign files as a yaml-language-server header.
const SchemaURL = "https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json"

// FieldError is one problem in a campaign file, located by line and column.
type FieldError struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Path    string `json:"path"`
	Message string `json:"message"`
}

// Error formats the problem as file:line:col: path: message.
func (e FieldError) Error() string {
	loc := e.File
	if e.Line > 0 {
		loc = fmt.Sprintf("%s:%d:%d", e.File, e.Line, e.Column)
	}
	if e.Path != "" {
		return fmt.Sprintf("%s: %s: %s", loc, e.Path, e.Message)
	}
	return fmt.Sprintf("%s: %s", loc, e.Message)
}

// ValidationError collects every problem found in a campaign file.
type ValidationError struct {
	Errors []FieldError
}

// Error lists every problem, one per line.
func (e *ValidationError) Error() string {
	lines := make([]string, len(e.Errors))
	for i, fe := range e.Errors {
		lines[i] = fe.Error()
	}
	return "invalid campaign:\n  " + strings.Join(lines, "\n  ")
}

var (
	schemaOnce sync.Once
	schema     *jsonschema.Schema
	errSchema  error
)

// compiledSchema compiles the embedded schema once.
func compiledSchema() (*jsonschema.Schema, error) {
	schemaOnce.Do(func() {
		doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(SchemaJSON))
		if err != nil {
			errSchema = err
			return
		}
		c := jsonschema.NewCompiler()
		if err := c.AddResource("campaign.schema.json", doc); err != nil {
			errSchema = err
			return
		}
		schema, errSchema = c.Compile("campaign.schema.json")
	})
	return schema, errSchema
}

// Load reads, validates and defaults the campaign file at path.
func Load(path string) (*Campaign, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	c, err := Parse(data, path)
	if err != nil {
		return nil, err
	}
	c.Path = abs
	if err := c.checkFiles(path); err != nil {
		return nil, err
	}
	return c, nil
}

// Parse validates and defaults campaign YAML. The filename is used only in
// error messages; relative file references are not checked.
func Parse(data []byte, filename string) (*Campaign, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, yamlError(filename, err)
	}
	if len(root.Content) == 0 {
		return nil, &ValidationError{Errors: []FieldError{{File: filename, Message: "file is empty"}}}
	}
	doc := root.Content[0]

	var generic any
	if err := doc.Decode(&generic); err != nil {
		return nil, yamlError(filename, err)
	}
	inst, err := toJSONValue(generic)
	if err != nil {
		return nil, &ValidationError{Errors: []FieldError{{File: filename, Line: doc.Line, Column: doc.Column, Message: err.Error()}}}
	}
	sch, err := compiledSchema()
	if err != nil {
		return nil, fmt.Errorf("compile campaign schema: %w", err)
	}
	if err := sch.Validate(inst); err != nil {
		var ve *jsonschema.ValidationError
		if !errors.As(err, &ve) {
			return nil, err
		}
		return nil, schemaErrors(filename, doc, ve)
	}

	var c Campaign
	if err := doc.Decode(&c); err != nil {
		return nil, yamlError(filename, err)
	}
	if fe := c.semanticErrors(filename, doc); len(fe) > 0 {
		return nil, &ValidationError{Errors: fe}
	}
	c.ApplyDefaults()
	return &c, nil
}

// semanticErrors reports rules the schema cannot express.
func (c *Campaign) semanticErrors(filename string, doc *yaml.Node) []FieldError {
	var out []FieldError
	if len(c.Targets.Orgs) == 0 && len(c.Targets.Include.Repos) == 0 {
		n := lookup(doc, []string{"targets"})
		out = append(out, FieldError{File: filename, Line: n.Line, Column: n.Column, Path: "targets", Message: "needs orgs, include.repos, or both"})
	}
	if c.PR.Branch != "" && !validBranch(c.PR.Branch) {
		n := lookup(doc, []string{"pr", "branch"})
		out = append(out, FieldError{File: filename, Line: n.Line, Column: n.Column, Path: "pr.branch", Message: "is not a valid git branch name"})
	}
	return out
}

// checkFiles reports change.run and pr.body values that look like paths
// but do not exist next to the campaign file, and loads vars.file.
func (c *Campaign) checkFiles(filename string) error {
	var out []FieldError
	for _, f := range []struct {
		path []string
		v    string
	}{{[]string{"change", "run"}, c.Change.Run}, {[]string{"pr", "body"}, c.PR.Body}} {
		if looksLikePath(f.v) && c.fileRef(f.v) == "" {
			out = append(out, FieldError{File: filename, Path: strings.Join(f.path, "."), Message: fmt.Sprintf("file %q not found next to the campaign file", f.v)})
		}
	}
	if c.Change.Starlark != "" && c.StarlarkFile() == "" {
		out = append(out, FieldError{File: filename, Path: "change.starlark", Message: fmt.Sprintf("file %q not found next to the campaign file", c.Change.Starlark)})
	}
	if err := c.loadVarsFile(); err != nil {
		out = append(out, FieldError{File: filename, Path: "vars.file", Message: fmt.Sprintf("%s: %v", c.Vars.File, err)})
	}
	if len(out) > 0 {
		data, err := os.ReadFile(filename)
		if err == nil {
			var root yaml.Node
			if yaml.Unmarshal(data, &root) == nil && len(root.Content) > 0 {
				for i := range out {
					n := lookup(root.Content[0], strings.Split(out[i].Path, "."))
					out[i].Line, out[i].Column = n.Line, n.Column
				}
			}
		}
		return &ValidationError{Errors: out}
	}
	return nil
}

// validBranch applies the subset of git check-ref-format rules that matter
// for names people write by hand.
func validBranch(b string) bool {
	if b == "" || strings.HasPrefix(b, "/") || strings.HasSuffix(b, "/") || strings.HasSuffix(b, ".lock") ||
		strings.HasPrefix(b, "-") || strings.HasSuffix(b, ".") || strings.Contains(b, "..") || strings.Contains(b, "//") || strings.Contains(b, "@{") {
		return false
	}
	for _, r := range b {
		if r < 0x20 || r == 0x7f || strings.ContainsRune(" ~^:?*[\\", r) {
			return false
		}
	}
	return true
}

// yamlError turns a YAML syntax or type error into a ValidationError,
// keeping the line numbers the YAML library reports.
func yamlError(filename string, err error) error {
	var te *yaml.TypeError
	if errors.As(err, &te) {
		fe := make([]FieldError, 0, len(te.Errors))
		for _, msg := range te.Errors {
			fe = append(fe, parseYAMLLine(filename, msg))
		}
		return &ValidationError{Errors: fe}
	}
	return &ValidationError{Errors: []FieldError{parseYAMLLine(filename, err.Error())}}
}

// parseYAMLLine extracts "line N:" from a YAML library message.
func parseYAMLLine(filename, msg string) FieldError {
	msg = strings.TrimPrefix(msg, "yaml: ")
	fe := FieldError{File: filename, Message: msg}
	if rest, ok := strings.CutPrefix(msg, "line "); ok {
		if i := strings.Index(rest, ":"); i > 0 {
			if n, err := strconv.Atoi(rest[:i]); err == nil {
				fe.Line, fe.Column = n, 1
				fe.Message = strings.TrimSpace(rest[i+1:])
			}
		}
	}
	return fe
}

// toJSONValue converts a YAML-decoded value into the JSON data model the
// schema validator expects, by round-tripping it through encoding/json.
func toJSONValue(v any) (any, error) {
	b, err := json.Marshal(normalize(v))
	if err != nil {
		return nil, err
	}
	return jsonschema.UnmarshalJSON(bytes.NewReader(b))
}

// normalize converts map[any]any values (YAML allows non-string keys) into
// map[string]any so they can be encoded as JSON, and drops null mapping
// values so a key left with only comments under it counts as absent.
func normalize(v any) any {
	switch t := v.(type) {
	case map[string]any:
		for k, e := range t {
			if e == nil {
				delete(t, k)
				continue
			}
			t[k] = normalize(e)
		}
		return t
	case map[any]any:
		m := make(map[string]any, len(t))
		for k, e := range t {
			if e != nil {
				m[fmt.Sprint(k)] = normalize(e)
			}
		}
		return m
	case []any:
		for i, e := range t {
			t[i] = normalize(e)
		}
		return t
	default:
		return v
	}
}

// schemaErrors flattens a schema validation error into one FieldError per
// failing leaf, each located at the YAML node it refers to.
func schemaErrors(filename string, doc *yaml.Node, ve *jsonschema.ValidationError) error {
	seen := map[string]bool{}
	var out []FieldError
	var walk func(e *jsonschema.ValidationError)
	add := func(fe FieldError) {
		if key := fe.Error(); !seen[key] {
			seen[key] = true
			out = append(out, fe)
		}
	}
	walk = func(e *jsonschema.ValidationError) {
		parent := lookup(doc, e.InstanceLocation)
		path := strings.Join(e.InstanceLocation, ".")
		// A bad key name: point at the key itself.
		// The validator reuses its location slice, so the reported location
		// of a property name error is unreliable: find the key instead.
		if pn, ok := e.ErrorKind.(*kind.PropertyNames); ok {
			n, keyPath := findKey(doc, pn.Property, nil)
			if n == nil {
				n = doc
			}
			add(FieldError{
				File: filename, Line: n.Line, Column: n.Column, Path: strings.Join(keyPath, "."),
				Message: "invalid name: use letters, digits and underscores, not starting with a digit",
			})
			return
		}
		// "Needs one of these keys" fails once per alternative too.
		if _, ok := e.ErrorKind.(*kind.AnyOf); ok {
			if missing := requiredAlternatives(e); len(missing) > 0 {
				add(FieldError{
					File: filename, Line: parent.Line, Column: parent.Column, Path: path,
					Message: "needs " + strings.Join(missing, ", ") + ", or both",
				})
				return
			}
		}
		if _, ok := e.ErrorKind.(*kind.Pattern); ok {
			if hint, ok := patternHints[path]; ok {
				add(FieldError{File: filename, Line: parent.Line, Column: parent.Column, Path: path, Message: hint})
				return
			}
		}
		// "One of these types" fails once per alternative; report it once.
		if _, ok := e.ErrorKind.(*kind.OneOf); ok {
			if got, want, ok := typeAlternatives(e); ok {
				add(FieldError{
					File: filename, Line: parent.Line, Column: parent.Column, Path: path,
					Message: fmt.Sprintf("got %s, want %s", got, strings.Join(want, ", ")),
				})
				return
			}
		}
		if len(e.Causes) > 0 {
			for _, c := range e.Causes {
				walk(c)
			}
			return
		}
		if ap, ok := e.ErrorKind.(*kind.AdditionalProperties); ok {
			for _, prop := range ap.Properties {
				n := keyNode(parent, prop)
				add(FieldError{
					File: filename, Line: n.Line, Column: n.Column,
					Path:    strings.Join(append(append([]string{}, e.InstanceLocation...), prop), "."),
					Message: "unknown field",
				})
			}
			return
		}
		add(FieldError{
			File:    filename,
			Line:    parent.Line,
			Column:  parent.Column,
			Path:    strings.Join(e.InstanceLocation, "."),
			Message: leafMessage(e),
		})
	}
	walk(ve)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Line != out[j].Line {
			return out[i].Line < out[j].Line
		}
		return out[i].Column < out[j].Column
	})
	return &ValidationError{Errors: out}
}

// patternHints replace a regular expression in an error with what the
// field should look like.
var patternHints = map[string]string{
	"name":            "use lowercase letters, digits and dashes, starting with a letter or digit",
	"change.timeout":  "use a Go duration such as 30s, 5m or 1h",
	"change.starlark": "must be a .star file",
}

// requiredAlternatives returns the quoted keys of an anyOf whose every
// branch failed only because a key was missing, or nil.
func requiredAlternatives(e *jsonschema.ValidationError) []string {
	var out []string
	for _, c := range e.Causes {
		r, ok := c.ErrorKind.(*kind.Required)
		if !ok || len(c.Causes) > 0 {
			return nil
		}
		for _, m := range r.Missing {
			out = append(out, "'"+m+"'")
		}
	}
	return out
}

// findKey returns the first mapping key node named key, depth first, and
// its path.
func findKey(n *yaml.Node, key string, path []string) (*yaml.Node, []string) {
	switch n.Kind {
	case yaml.MappingNode:
		for i := 0; i+1 < len(n.Content); i += 2 {
			k := n.Content[i]
			p := append(append([]string{}, path...), k.Value)
			if k.Value == key {
				return k, p
			}
			if found, fp := findKey(n.Content[i+1], key, p); found != nil {
				return found, fp
			}
		}
	case yaml.SequenceNode:
		for i, c := range n.Content {
			if found, fp := findKey(c, key, append(append([]string{}, path...), strconv.Itoa(i))); found != nil {
				return found, fp
			}
		}
	}
	return nil, nil
}

// typeAlternatives reports whether every leaf under e is a type mismatch,
// returning the type found and every type that would have been accepted.
func typeAlternatives(e *jsonschema.ValidationError) (string, []string, bool) {
	var got string
	var want []string
	seen := map[string]bool{}
	var walk func(e *jsonschema.ValidationError) bool
	walk = func(e *jsonschema.ValidationError) bool {
		if len(e.Causes) > 0 {
			for _, c := range e.Causes {
				if !walk(c) {
					return false
				}
			}
			return true
		}
		t, ok := e.ErrorKind.(*kind.Type)
		if !ok {
			return false
		}
		got = t.Got
		for _, w := range t.Want {
			if w == "array" {
				w = "a list"
			}
			if !seen[w] {
				seen[w] = true
				want = append(want, w)
			}
		}
		return true
	}
	if !walk(e) || got == "" {
		return "", nil, false
	}
	return got, want, true
}

// leafMessage renders a validator message without its location prefix.
func leafMessage(e *jsonschema.ValidationError) string {
	return strings.TrimSpace(e.ErrorKind.LocalizedString(printer))
}

// printer renders validator messages in English.
var printer = message.NewPrinter(language.English)

// keyNode returns the key node for prop in a mapping, or the mapping itself.
func keyNode(n *yaml.Node, prop string) *yaml.Node {
	if n.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(n.Content); i += 2 {
			if n.Content[i].Value == prop {
				return n.Content[i]
			}
		}
	}
	return n
}

// lookup finds the YAML node at a JSON-pointer-like path. When a key is
// missing it returns the deepest node found, so errors about a missing
// property point at its parent mapping.
func lookup(n *yaml.Node, path []string) *yaml.Node {
	cur := n
	for _, seg := range path {
		next := child(cur, seg)
		if next == nil {
			return cur
		}
		cur = next
	}
	return cur
}

// child returns a mapping's value (or a sequence's element) for one path
// segment, or nil when there is none.
func child(n *yaml.Node, seg string) *yaml.Node {
	switch n.Kind {
	case yaml.MappingNode:
		for i := 0; i+1 < len(n.Content); i += 2 {
			if n.Content[i].Value == seg {
				return n.Content[i+1]
			}
		}
	case yaml.SequenceNode:
		if i, err := strconv.Atoi(seg); err == nil && i >= 0 && i < len(n.Content) {
			return n.Content[i]
		}
	case yaml.AliasNode:
		if n.Alias != nil {
			return child(n.Alias, seg)
		}
	}
	return nil
}
