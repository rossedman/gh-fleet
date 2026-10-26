package star

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/pelletier/go-toml/v2"
	"go.yaml.in/yaml/v3"
)

// editScript sets vars.key to vars.value in file f.yaml with yamlpath.set.
const editScript = `load("@fleet//yamlpath.star", "set")
def change(repo):
    v = repo.vars["value"]
    if repo.vars["kind"] == "int":
        v = int(v)
    elif repo.vars["kind"] == "bool":
        v = v == "true"
    set(repo, "f.yaml", repo.vars["key"].split("."), v)
`

// keySegment is what fuzzed key segments are cut down to.
var keySegment = regexp.MustCompile(`[^a-z0-9_]+`)

// plainYAML reports whether s is a YAML mapping the line-based editor is
// meant for: one document, block style at the top, no anchors, tags or
// complex keys.
func plainYAML(s string) (map[string]any, bool) {
	if strings.ContainsAny(s, "&*!?%@`{}[]|>\t\r\"'\\") || strings.Contains(s, "---") || strings.Contains(s, "...") || strings.Contains(s, "#") {
		return nil, false
	}
	if strings.HasPrefix(strings.TrimLeft(s, "\n"), " ") {
		return nil, false // an indented root; TestYAMLFuzzFindings covers it
	}
	for _, line := range strings.Split(s, "\n") {
		// Every line is a key or a list item: no plain values wrapped
		// onto the next line.
		if l := strings.TrimSpace(line); l != "" && !strings.Contains(l, ":") && !strings.HasPrefix(l, "- ") && l != "-" {
			return nil, false
		}
	}
	var m map[string]any
	if err := yaml.Unmarshal([]byte(s), &m); err != nil || m == nil || !realKeys(m) {
		return nil, false
	}
	return m, true
}

// realKey is a key as config files write them, such as runs-on or
// app.kubernetes.io/name.
var realKey = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_./-]*$`)

// realKeys reports whether every key in a decoded document is a realKey.
func realKeys(v any) bool {
	switch t := v.(type) {
	case map[string]any:
		for k, c := range t {
			if !realKey.MatchString(k) || !realKeys(c) {
				return false
			}
		}
	case []any:
		for _, c := range t {
			if !realKeys(c) {
				return false
			}
		}
	}
	return true
}

// setPath sets a nested key in a decoded mapping, creating maps on the
// way, and reports false when the path runs through a non-map value.
func setPath(m map[string]any, keys []string, v any) bool {
	for _, k := range keys[:len(keys)-1] {
		next, ok := m[k]
		if !ok || next == nil {
			child := map[string]any{}
			m[k] = child
			m = child
			continue
		}
		child, ok := next.(map[string]any)
		if !ok {
			return false
		}
		m = child
	}
	m[keys[len(keys)-1]] = v
	return true
}

// FuzzYAMLSet checks yamlpath.set on plain YAML: the result still parses
// and equals the input with exactly that key set.
func FuzzYAMLSet(f *testing.F) {
	for _, seed := range []struct{ doc, key, value, kind string }{
		{"a:\n  b: 1\n", "a.c", "x", "string"},
		{"jobs:\n  test:\n    runs-on: ubuntu-latest\n", "jobs.test.runs-on", "ubuntu-24.04", "string"},
		{"a: 1\nb:\n  - x\n  - y\nc:\n  d: 2\n", "c.e", "3", "int"},
		{"x:\n  y:\n    z: true\n", "x.y.w", "false", "bool"},
		{"name: web\n", "spec.replicas", "2", "int"},
		{"a:\n  - b: 1\n    c: 2\nd: 3\n", "d", "yes please", "string"},
	} {
		f.Add(seed.doc, seed.key, seed.value, seed.kind)
	}
	s, err := Load(context.Background(), writeScript(f, editScript), "", Options{})
	if err != nil {
		f.Fatal(err)
	}
	f.Fuzz(func(t *testing.T, doc, key, value, kind string) {
		before, ok := plainYAML(doc)
		if !ok {
			return
		}
		var keys []string
		for _, k := range strings.Split(key, ".") {
			if k = keySegment.ReplaceAllString(strings.ToLower(k), ""); k != "" && len(k) <= 64 {
				keys = append(keys, k)
			}
		}
		if len(keys) == 0 || len(keys) > 3 || !utf8.ValidString(value) || strings.ContainsFunc(value, unicode.IsControl) || len(value) > 40 {
			return
		}
		var want any = value
		switch kind {
		case "int":
			if !regexp.MustCompile(`^[1-9][0-9]{0,6}$`).MatchString(value) {
				return
			}
			var n int
			_ = yaml.Unmarshal([]byte(value), &n)
			want = n
		case "bool":
			want = value == "true"
		default:
			kind = "string"
		}
		expected, _ := plainYAML(doc)
		if !setPath(expected, keys, want) {
			return // the path runs through a list or scalar; set refuses those
		}
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "f.yaml"), []byte(doc), 0o644); err != nil {
			t.Fatal(err)
		}
		vars := map[string]any{"key": strings.Join(keys, "."), "value": value, "kind": kind}
		if _, err := s.Run(context.Background(), dir, RepoInfo{Vars: vars}, io.Discard, time.Minute); err != nil {
			return // refusing is fine; writing something wrong is not
		}
		out, _ := os.ReadFile(filepath.Join(dir, "f.yaml"))
		var got map[string]any
		if err := yaml.Unmarshal(out, &got); err != nil {
			t.Fatalf("set %v=%q on\n%s\nwrote invalid YAML (%v):\n%s", keys, value, doc, err, out)
		}
		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("set %v=%q on\n%s\nwrote\n%s\ndecoded %v, want %v (was %v)", keys, value, doc, out, got, expected, before)
		}
	})
}

// writeScript writes a change script to a temp dir and returns its path.
func writeScript(f *testing.F, src string) string {
	dir := f.TempDir()
	p := filepath.Join(dir, "change.star")
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		f.Fatal(err)
	}
	return p
}

// tomlScript sets vars.key in [vars.table] of f.toml with toml.set.
const tomlScript = `load("@fleet//toml.star", "set")
def change(repo):
    v = repo.vars["value"]
    if repo.vars["kind"] == "int":
        v = int(v)
    elif repo.vars["kind"] == "bool":
        v = v == "true"
    set(repo, "f.toml", repo.vars["table"], repo.vars["key"], v)
`

// plainTOML reports whether s is TOML the line-based editor is meant for:
// tables and key = value lines with bare keys, no inline tables, arrays
// of tables or multi-line values.
func plainTOML(s string) (map[string]any, bool) {
	if strings.ContainsAny(s, "{}\\\t\r'#") || strings.Contains(s, "[[") || strings.Contains(s, `"""`) {
		return nil, false
	}
	for _, line := range strings.Split(s, "\n") {
		l := strings.TrimSpace(line)
		if l == "" {
			continue
		}
		if strings.HasPrefix(l, "[") {
			if !regexp.MustCompile(`^\[[a-z0-9_-]+(\.[a-z0-9_-]+)*\]$`).MatchString(l) {
				return nil, false
			}
			continue
		}
		if !regexp.MustCompile(`^[a-z0-9_-]+ *= *[^=\[\]]+$`).MatchString(l) {
			return nil, false
		}
	}
	m := map[string]any{}
	if err := toml.Unmarshal([]byte(s), &m); err != nil {
		return nil, false
	}
	return m, true
}

// FuzzTOMLSet checks toml.set on plain TOML: the result still parses and
// equals the input with exactly that key set.
func FuzzTOMLSet(f *testing.F) {
	for _, seed := range []struct{ doc, table, key, value, kind string }{
		{"[tool.ruff]\nline-length = 88\n", "tool.ruff", "line-length", "100", "int"},
		{"[project]\nname = \"api\"\n\n[tool.ruff]\nx = 1\n", "tool.ruff.lint", "select", "E", "string"},
		{"name = \"x\"\n[a]\nb = true\n", "", "version", "1.0", "string"},
		{"[a]\nb = 1\n[a.c]\nd = 2\n", "a", "e", "true", "bool"},
	} {
		f.Add(seed.doc, seed.table, seed.key, seed.value, seed.kind)
	}
	s, err := Load(context.Background(), writeScript(f, tomlScript), "", Options{})
	if err != nil {
		f.Fatal(err)
	}
	f.Fuzz(func(t *testing.T, doc, table, key, value, kind string) {
		if _, ok := plainTOML(doc); !ok {
			return
		}
		key = keySegment.ReplaceAllString(strings.ToLower(key), "")
		var tables []string
		for _, k := range strings.Split(table, ".") {
			if k = keySegment.ReplaceAllString(strings.ToLower(k), ""); k != "" && len(k) <= 32 {
				tables = append(tables, k)
			}
		}
		if key == "" || len(key) > 32 || len(tables) > 3 || !utf8.ValidString(value) || strings.ContainsFunc(value, unicode.IsControl) || len(value) > 40 {
			return
		}
		var want any = value
		switch kind {
		case "int":
			if !regexp.MustCompile(`^[1-9][0-9]{0,6}$`).MatchString(value) {
				return
			}
			var n int64
			_ = toml.Unmarshal([]byte("n = "+value), &struct{ N *int64 }{&n})
			want = n
		case "bool":
			want = value == "true"
		default:
			kind = "string"
		}
		expected, _ := plainTOML(doc)
		path := append(slices.Clone(tables), key)
		if !setPath(expected, path, want) || isTable(doc, path) {
			return // set must refuse these, which the unit tests check
		}
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "f.toml"), []byte(doc), 0o644); err != nil {
			t.Fatal(err)
		}
		vars := map[string]any{"table": strings.Join(tables, "."), "key": key, "value": value, "kind": kind}
		if _, err := s.Run(context.Background(), dir, RepoInfo{Vars: vars}, io.Discard, time.Minute); err != nil {
			return
		}
		out, _ := os.ReadFile(filepath.Join(dir, "f.toml"))
		var got map[string]any
		if err := toml.Unmarshal(out, &got); err != nil {
			t.Fatalf("set [%s] %s=%q on\n%s\nwrote invalid TOML (%v):\n%s", strings.Join(tables, "."), key, value, doc, err, out)
		}
		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("set [%s] %s=%q on\n%s\nwrote\n%s\ndecoded %v, want %v", strings.Join(tables, "."), key, value, doc, out, got, expected)
		}
	})
}

// isTable reports whether keys name a table in the TOML text, which set
// refuses to overwrite with a value.
func isTable(doc string, keys []string) bool {
	m := map[string]any{}
	_ = toml.Unmarshal([]byte(doc), &m)
	var v any = m
	for _, k := range keys {
		t, ok := v.(map[string]any)
		if !ok {
			return false
		}
		if v, ok = t[k]; !ok {
			return false
		}
	}
	_, ok := v.(map[string]any)
	return ok
}
