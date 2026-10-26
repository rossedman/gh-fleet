package render

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"slices"
	"strings"
	"text/template"
	"time"
)

// SelectFields keeps only fields of the JSON records in b: of each object
// in a top-level array, or of a top-level object. Records without a field
// simply leave it out.
func SelectFields(b []byte, fields []string) ([]byte, error) {
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, err
	}
	pick := func(rec any) any {
		obj, ok := rec.(map[string]any)
		if !ok {
			return rec
		}
		out := make(map[string]any, len(fields))
		for _, f := range fields {
			if val, ok := obj[f]; ok {
				out[f] = val
			}
		}
		return out
	}
	switch t := v.(type) {
	case []any:
		for i := range t {
			t[i] = pick(t[i])
		}
	case map[string]any:
		v = pick(t)
	}
	return json.Marshal(v)
}

// FieldsOf lists the JSON fields of the records in samples: the elements
// of a slice, or a struct itself. They are sorted and without duplicates.
func FieldsOf(samples ...any) []string {
	var out []string
	for _, s := range samples {
		out = append(out, structFields(reflect.TypeOf(s))...)
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// structFields returns the JSON names of t's exported fields, following
// slices, pointers and embedded structs.
func structFields(t reflect.Type) []string {
	for t != nil && (t.Kind() == reflect.Slice || t.Kind() == reflect.Array || t.Kind() == reflect.Pointer) {
		t = t.Elem()
	}
	if t == nil || t.Kind() != reflect.Struct {
		return nil
	}
	var out []string
	for i := range t.NumField() {
		f := t.Field(i)
		name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		if f.Anonymous && name == "" {
			// encoding/json promotes an embedded struct's fields, even
			// when the struct type itself is unexported.
			out = append(out, structFields(f.Type)...)
			continue
		}
		if !f.IsExported() || name == "-" {
			continue
		}
		if name == "" {
			name = f.Name
		}
		out = append(out, name)
	}
	return out
}

// templateFuncs are the helpers gh offers in --template, minus the ones
// that need a terminal.
var templateFuncs = template.FuncMap{
	"join": func(sep string, v []any) string {
		parts := make([]string, len(v))
		for i, x := range v {
			parts[i] = fmt.Sprint(x)
		}
		return strings.Join(parts, sep)
	},
	"pluck": func(field string, v []any) []any {
		out := make([]any, 0, len(v))
		for _, x := range v {
			if m, ok := x.(map[string]any); ok {
				out = append(out, m[field])
			}
		}
		return out
	},
	"truncate": func(n int, s string) string {
		if r := []rune(s); len(r) > n {
			return string(r[:max(n-3, 0)]) + "..."
		}
		return s
	},
	"timefmt": func(layout, s string) (string, error) {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return "", err
		}
		return t.Format(layout), nil
	},
	"json": func(v any) (string, error) {
		b, err := json.Marshal(v)
		return string(b), err
	},
}

// executeTemplate formats JSON b with the Go template text.
func executeTemplate(w io.Writer, text string, b []byte) error {
	t, err := template.New("template").Funcs(templateFuncs).Parse(text)
	if err != nil {
		return fmt.Errorf("--template: %w", err)
	}
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	if err := t.Execute(w, v); err != nil {
		return fmt.Errorf("--template: %w", err)
	}
	return nil
}
