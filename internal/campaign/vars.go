package campaign

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Vars are a campaign's per-repo variables. A repo's values are built in
// order, later sources winning: Default, then every matching Match rule in
// file order, then the repo's row in File.
type Vars struct {
	Default map[string]any `yaml:"default,omitempty" json:"default,omitempty"`
	Match   []VarRule      `yaml:"match,omitempty" json:"match,omitempty"`
	// File is a CSV, JSON or YAML file of per-repo values, relative to the
	// campaign file.
	File string `yaml:"file,omitempty" json:"file,omitempty"`

	// rows are File's values keyed by lower-case repo full name.
	rows map[string]map[string]any
}

// VarRule sets variables on the repos its When matches.
type VarRule struct {
	When When           `yaml:"when" json:"when"`
	Set  map[string]any `yaml:"set" json:"set"`
}

// When selects repos for a VarRule. Every key that is set must match; values
// within a key are alternatives. An empty When matches every repo.
type When struct {
	Repos      []string              `yaml:"repos,omitempty" json:"repos,omitempty"`
	Topics     []string              `yaml:"topics,omitempty" json:"topics,omitempty"`
	Languages  []string              `yaml:"languages,omitempty" json:"languages,omitempty"`
	Properties map[string]StringList `yaml:"properties,omitempty" json:"properties,omitempty"`
}

// Row returns the data file's values for a repo, or nil.
func (v *Vars) Row(repo string) map[string]any {
	return v.rows[strings.ToLower(repo)]
}

// RowRepos returns the repos named in the data file, sorted.
func (v *Vars) RowRepos() []string {
	out := make([]string, 0, len(v.rows))
	for r := range v.rows {
		out = append(out, r)
	}
	sort.Strings(out)
	return out
}

// UsesProperties reports whether the campaign depends on custom property
// values: in its target filters, its vars rules, its PR templates, or its
// steps (FLEET_PROP_ in a shell script, .properties in Starlark). A
// Starlark script that loads code from another repo can't be checked, so
// it counts as using them. Fetching properties costs a call per 100 repos
// of each org, so campaigns that don't use them skip it.
func (c *Campaign) UsesProperties() bool {
	if len(c.Targets.Include.Properties) > 0 || len(c.Targets.Exclude.Properties) > 0 {
		return true
	}
	for _, r := range c.Vars.Match {
		if len(r.When.Properties) > 0 {
			return true
		}
	}
	texts := append([]string{c.PR.Title, c.PR.Body, c.PR.Commit.Message}, c.PR.Labels...)
	texts = append(texts, c.PR.Reviewers...)
	if p := c.fileRef(c.PR.Body); p != "" {
		if b, err := os.ReadFile(p); err == nil {
			texts = append(texts, string(b))
		}
	}
	for _, t := range texts {
		if strings.Contains(t, ".Properties") {
			return true
		}
	}
	return c.stepsUseProperties()
}

// stepsUseProperties reports whether a step reads custom properties.
func (c *Campaign) stepsUseProperties() bool {
	read := func(p string) string {
		b, _ := os.ReadFile(p) // a missing file fails the run elsewhere
		return string(b)
	}
	for _, s := range c.Change.Steps {
		if s.Run != "" {
			script := s.Run
			if p := c.fileRef(s.Run); p != "" {
				script = read(p)
			}
			if strings.Contains(script, "FLEET_PROP_") {
				return true
			}
		}
		if s.Starlark != "" && starlarkUsesProperties(filepath.Join(c.Dir(), s.Starlark), read, map[string]bool{}) {
			return true
		}
	}
	return false
}

// Loads in a Starlark script: a local one ("./x.star", relative to the
// script) and one from another repo ("host/owner/repo@sha:x.star").
var (
	localLoad  = regexp.MustCompile(`load\(\s*["'](\.{1,2}/[^"']+)["']`)
	remoteLoad = regexp.MustCompile(`load\(\s*["'][^"'@]+/[^"']*@`)
)

// starlarkUsesProperties reports whether the script at p, or a local
// script it loads, reads repo.properties or loads code from another repo.
func starlarkUsesProperties(p string, read func(string) string, seen map[string]bool) bool {
	if seen[p] {
		return false
	}
	seen[p] = true
	src := read(p)
	if strings.Contains(src, ".properties") || remoteLoad.MatchString(src) {
		return true
	}
	for _, m := range localLoad.FindAllStringSubmatch(src, -1) {
		if starlarkUsesProperties(filepath.Join(filepath.Dir(p), m[1]), read, seen) {
			return true
		}
	}
	return false
}

// StringList is a list of strings that also accepts numbers and booleans in
// YAML, so `tier: [1, 2]` works as well as `tier: ["1", "2"]`.
type StringList []string

// UnmarshalYAML decodes a scalar or a sequence of scalars.
func (l *StringList) UnmarshalYAML(n *yaml.Node) error {
	switch n.Kind {
	case yaml.ScalarNode:
		*l = StringList{n.Value}
		return nil
	case yaml.SequenceNode:
		out := make(StringList, 0, len(n.Content))
		for _, c := range n.Content {
			if c.Kind != yaml.ScalarNode {
				return fmt.Errorf("line %d: want a string, number or boolean", c.Line)
			}
			out = append(out, c.Value)
		}
		*l = out
		return nil
	}
	return fmt.Errorf("line %d: want a value or a list of values", n.Line)
}

// varName is what a variable may be called: usable as .Vars.name in a
// template and as FLEET_VAR_NAME in a script.
var varName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// ValidVarName reports whether name can be used as a variable.
func ValidVarName(name string) bool { return varName.MatchString(name) }

// EnvName turns a variable or property name into the suffix of its
// environment variable: upper case, with anything but letters, digits and
// underscores replaced by underscores.
func EnvName(name string) string {
	b := []byte(strings.ToUpper(name))
	for i, c := range b {
		if (c < 'A' || c > 'Z') && (c < '0' || c > '9') && c != '_' {
			b[i] = '_'
		}
	}
	return string(b)
}

// FormatValue renders a variable value as text: lists are comma-joined,
// whole numbers have no decimal point.
func FormatValue(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case bool:
		return strconv.FormatBool(t)
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case []any:
		parts := make([]string, len(t))
		for i, e := range t {
			parts[i] = FormatValue(e)
		}
		return strings.Join(parts, ",")
	case []string:
		return strings.Join(t, ",")
	}
	return fmt.Sprint(v)
}

// loadVarsFile reads vars.file relative to the campaign directory.
func (c *Campaign) loadVarsFile() error {
	if c.Vars.File == "" {
		return nil
	}
	p := c.Vars.File
	if !filepath.IsAbs(p) {
		p = filepath.Join(c.Dir(), p)
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return err
	}
	var rows map[string]map[string]any
	switch strings.ToLower(filepath.Ext(p)) {
	case ".csv":
		rows, err = parseVarsCSV(data)
	case ".json":
		rows, err = parseVarsMap(data, json.Unmarshal)
	case ".yaml", ".yml":
		rows, err = parseVarsMap(data, yaml.Unmarshal)
	default:
		return fmt.Errorf("unsupported file type %q: use .csv, .json, .yaml or .yml", filepath.Ext(p))
	}
	if err != nil {
		return err
	}
	c.Vars.rows = rows
	return nil
}

// parseVarsCSV reads a CSV whose header has a `repo` column; every other
// column is a variable. Empty cells leave the variable unset.
func parseVarsCSV(data []byte) (map[string]map[string]any, error) {
	r := csv.NewReader(bytes.NewReader(data))
	r.TrimLeadingSpace = true
	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	repoCol := -1
	for i, h := range header {
		header[i] = strings.TrimSpace(h)
		if header[i] == "repo" {
			repoCol = i
		} else if !ValidVarName(header[i]) {
			return nil, fmt.Errorf("line 1: column %q is not a valid variable name", header[i])
		}
	}
	if repoCol < 0 {
		return nil, errors.New("line 1: no `repo` column")
	}
	rows := map[string]map[string]any{}
	for {
		rec, err := r.Read()
		if errors.Is(err, io.EOF) {
			return rows, nil
		}
		if err != nil {
			return nil, err
		}
		line, _ := r.FieldPos(0)
		repo := strings.TrimSpace(rec[repoCol])
		if err := checkRepoKey(repo); err != nil {
			return nil, fmt.Errorf("line %d: %w", line, err)
		}
		key := strings.ToLower(repo)
		if _, dup := rows[key]; dup {
			return nil, fmt.Errorf("line %d: %s appears twice", line, repo)
		}
		vals := map[string]any{}
		for i, cell := range rec {
			if i != repoCol && strings.TrimSpace(cell) != "" {
				vals[header[i]] = strings.TrimSpace(cell)
			}
		}
		rows[key] = vals
	}
}

// parseVarsMap reads a JSON or YAML object mapping owner/name to an object
// of variables.
func parseVarsMap(data []byte, unmarshal func([]byte, any) error) (map[string]map[string]any, error) {
	var raw map[string]map[string]any
	if err := unmarshal(data, &raw); err != nil {
		return nil, err
	}
	rows := map[string]map[string]any{}
	for repo, vals := range raw {
		if err := checkRepoKey(repo); err != nil {
			return nil, err
		}
		for k, v := range vals {
			if !ValidVarName(k) {
				return nil, fmt.Errorf("%s: %q is not a valid variable name", repo, k)
			}
			if !validValue(v) {
				return nil, fmt.Errorf("%s: %s must be a string, number, boolean or list of those", repo, k)
			}
		}
		rows[strings.ToLower(repo)] = vals
	}
	return rows, nil
}

// checkRepoKey validates an owner/name key in a data file.
func checkRepoKey(repo string) error {
	owner, name, ok := strings.Cut(repo, "/")
	if !ok || owner == "" || name == "" || strings.Contains(name, "/") {
		return fmt.Errorf("%q is not an owner/name repo", repo)
	}
	return nil
}

// validValue reports whether v is a scalar or a list of scalars.
func validValue(v any) bool {
	switch t := v.(type) {
	case string, bool, int, int64, float64:
		return true
	case []any:
		for _, e := range t {
			switch e.(type) {
			case string, bool, int, int64, float64:
			default:
				return false
			}
		}
		return true
	}
	return false
}
