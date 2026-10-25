package star

import (
	"bytes"
	gojson "encoding/json"
	"fmt"
	"math"
	"strconv"

	starjson "go.starlark.net/lib/json"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// jsonModule is Starlark's json module with encoders that keep a dict's
// key order, so rewriting a file like package.json changes only what the
// script changed.
var jsonModule = func() *starlarkstruct.Module {
	m := newModule("json", jsonEncoders)
	m.Members["decode"] = starjson.Module.Members["decode"]
	m.Members["indent"] = starjson.Module.Members["indent"]
	return m
}()

// jsonEncoders replace the standard module's sorting encoders.
var jsonEncoders = map[string]builtin{
	"encode": func(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var x starlark.Value
		if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &x); err != nil {
			return nil, err
		}
		var buf bytes.Buffer
		if err := encodeJSON(&buf, x, 0); err != nil {
			return nil, fmt.Errorf("%s: %w", b.Name(), err)
		}
		return starlark.String(buf.String()), nil
	},
	"encode_indent": func(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var x starlark.Value
		prefix, indent := "", "\t"
		if err := starlark.UnpackArgs(b.Name(), args, kwargs, "x", &x, "prefix?", &prefix, "indent?", &indent); err != nil {
			return nil, err
		}
		var buf bytes.Buffer
		if err := encodeJSON(&buf, x, 0); err != nil {
			return nil, fmt.Errorf("%s: %w", b.Name(), err)
		}
		var out bytes.Buffer
		if err := gojson.Indent(&out, buf.Bytes(), prefix, indent); err != nil {
			return nil, fmt.Errorf("%s: %w", b.Name(), err)
		}
		return starlark.String(out.String()), nil
	},
}

// encodeJSON writes v as compact JSON, keeping dict key order.
func encodeJSON(buf *bytes.Buffer, v starlark.Value, depth int) error {
	if depth > maxYAMLDepth {
		return fmt.Errorf("value nested too deeply")
	}
	switch t := v.(type) {
	case starlark.NoneType:
		buf.WriteString("null")
	case starlark.Bool:
		buf.WriteString(strconv.FormatBool(bool(t)))
	case starlark.Int:
		buf.WriteString(t.String())
	case starlark.Float:
		f := float64(t)
		if math.IsInf(f, 0) || math.IsNaN(f) {
			return fmt.Errorf("cannot encode non-finite float %v", f)
		}
		buf.WriteString(strconv.FormatFloat(f, 'g', -1, 64))
	case starlark.String:
		writeJSONString(buf, string(t))
	case *starlark.Dict:
		buf.WriteByte('{')
		for i, item := range t.Items() {
			k, ok := starlark.AsString(item[0])
			if !ok {
				return fmt.Errorf("dict key %s is not a string", item[0])
			}
			if i > 0 {
				buf.WriteByte(',')
			}
			writeJSONString(buf, k)
			buf.WriteByte(':')
			if err := encodeJSON(buf, item[1], depth+1); err != nil {
				return err
			}
		}
		buf.WriteByte('}')
	case *starlark.List, starlark.Tuple:
		buf.WriteByte('[')
		iter := starlark.Iterate(v)
		defer iter.Done()
		var x starlark.Value
		for i := 0; iter.Next(&x); i++ {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := encodeJSON(buf, x, depth+1); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
	default:
		return fmt.Errorf("cannot encode %s as JSON", v.Type())
	}
	return nil
}

// writeJSONString writes s as a JSON string without HTML escaping, so
// characters like < and & stay readable.
func writeJSONString(buf *bytes.Buffer, s string) {
	var b bytes.Buffer
	enc := gojson.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s)
	buf.Write(bytes.TrimSuffix(b.Bytes(), []byte("\n")))
}
