package star

import (
	"strings"
	"testing"

	starjson "go.starlark.net/lib/json"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

// editJSON decodes src, applies a Starlark edit to it as `d` and returns
// encodeLike's output.
func editJSON(t *testing.T, src, edit string) (string, error) {
	t.Helper()
	thread := &starlark.Thread{}
	d, err := starlark.Call(thread, starjson.Module.Members["decode"], starlark.Tuple{starlark.String(src)}, nil)
	if err != nil {
		t.Fatalf("decode %q: %v", src, err)
	}
	if edit != "" {
		globals := starlark.StringDict{"d": d}
		if _, err := starlark.ExecFileOptions(&syntax.FileOptions{}, thread, "edit.star", "def f():\n    "+strings.ReplaceAll(edit, "\n", "\n    ")+"\nf()\n", globals); err != nil {
			t.Fatalf("edit %q: %v", edit, err)
		}
	}
	return encodeLike(thread, d, src)
}

func TestEncodeLike(t *testing.T) {
	t.Parallel()
	for name, tt := range map[string]struct{ src, edit, want string }{
		"unchanged keeps everything": {"{\n  \"a\": [1,2],\n  \"b\": {\"x\": 1.50, \"é\": \"\\u00e9\"}\n}\n", "", "{\n  \"a\": [1,2],\n  \"b\": {\"x\": 1.50, \"é\": \"\\u00e9\"}\n}\n"},
		"one value":                  {"{\n  \"a\": [\"x\", \"y\"],\n  \"v\": \"1\"\n}", `d["v"] = "2"`, "{\n  \"a\": [\"x\", \"y\"],\n  \"v\": \"2\"\n}"},
		"append to inline list":      {"{\n  \"a\": [\"x\", \"y\"]\n}", `d["a"].append("z")`, "{\n  \"a\": [\"x\", \"y\", \"z\"]\n}"},
		"append to one-item list":    {"{\"a\": [\"x\"]}", `d["a"].append("z")`, "{\"a\": [\"x\", \"z\"]}"},
		"append to compact list":     {"{\"a\":[\"x\"]}", `d["a"].append("z")`, "{\"a\":[\"x\",\"z\"]}"},
		"append to multiline list":   {"{\n\t\"a\": [\n\t\t\"x\"\n\t]\n}", `d["a"].append("z")`, "{\n\t\"a\": [\n\t\t\"x\",\n\t\t\"z\"\n\t]\n}"},
		"new key indented":           {"{\n    \"a\": 1\n}", `d["b"] = {"c": [1, {"d": True}]}`, "{\n    \"a\": 1,\n    \"b\": {\n        \"c\": [\n            1,\n            {\n                \"d\": true\n            }\n        ]\n    }\n}"},
		"new plain list inline":      {"{\n  \"a\": [1]\n}", `d["b"] = ["x", "y"]`, "{\n  \"a\": [1],\n  \"b\": [\"x\", \"y\"]\n}"},
		"new plain list expanded":    {"{\n  \"a\": 1\n}", `d["b"] = ["x"]`, "{\n  \"a\": 1,\n  \"b\": [\n    \"x\"\n  ]\n}"},
		"delete middle and last":     {"{\n  \"a\": 1,\n  \"b\": 2,\n  \"c\": 3\n}", "d.pop(\"b\")\nd.pop(\"c\")", "{\n  \"a\": 1\n}"},
		"delete first":               {"{\"a\": 1, \"b\": 2}", `d.pop("a")`, "{\"b\": 2}"},
		"empty root gets indented":   {"{}", `d["a"] = {"b": 1}`, "{\n  \"a\": {\n    \"b\": 1\n  }\n}"},
		"empty nested object":        {"{\n  \"a\": {}\n}", `d["a"]["b"] = 1`, "{\n  \"a\": {\n    \"b\": 1\n  }\n}"},
		"inline object stays inline": {"{\n  \"p\": {\"semi\": true}\n}", `d["p"]["semi"] = False`, "{\n  \"p\": {\"semi\": false}\n}"},
		"inline object grows nested": {"{\n  \"p\": {\"a\": 1}\n}", `d["p"]["b"] = {"c": 2}`, "{\n  \"p\": {\n    \"a\": 1,\n    \"b\": {\n      \"c\": 2\n    }\n  }\n}"},
		"type change":                {"{\n  \"e\": \"x\"\n}", `d["e"] = ["x", "y"]`, "{\n  \"e\": [\n    \"x\",\n    \"y\"\n  ]\n}"},
		"compact file":               {"{\"a\":{\"b\":1}}", `d["a"]["c"] = [1, {"x": None}]`, "{\"a\":{\"b\":1,\"c\":[1,{\"x\":null}]}}"},
		"reorder is a change":        {"{\"a\": 1, \"b\": 2}", "v = {\"b\": 2, \"a\": 1}\nd.clear()\nd.update(v)", "{\"b\": 2, \"a\": 1}"},
		"whitespace around root":     {"\n  [1, 2]  \n", `d.append(3)`, "\n  [1, 2, 3]  \n"},
	} {
		got, err := editJSON(t, tt.src, tt.edit)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if got != tt.want {
			t.Errorf("%s:\n got %q\nwant %q", name, got, tt.want)
		}
	}
	thread := &starlark.Thread{}
	for _, bad := range []string{"", "{", "{\"a\" 1}", "[1,]x", "{\"a\": 1} x", "{1: 2}", "[1 2]", "\"open"} {
		if _, err := encodeLike(thread, starlark.None, bad); err == nil {
			t.Errorf("encodeLike(%q) succeeded", bad)
		}
	}
	if _, err := encodeLike(thread, starlark.NewDict(0), strings.Repeat("[", maxYAMLDepth+2)+strings.Repeat("]", maxYAMLDepth+2)); err == nil {
		t.Error("deep nesting accepted")
	}
}

// FuzzEncodeLike checks that writing back an unchanged value reproduces
// the text exactly, whatever its layout.
func FuzzEncodeLike(f *testing.F) {
	for _, s := range []string{"{}", "[1, 2.50]", "{\n  \"a\": [\"x\", {\"b\": null}],\n\t\"c\": \"\\u00e9\"\n}\n", "{\"a\":true,\"b\":[]}"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, src string) {
		thread := &starlark.Thread{}
		d, err := starlark.Call(thread, starjson.Module.Members["decode"], starlark.Tuple{starlark.String(src)}, nil)
		if err != nil {
			return
		}
		got, err := encodeLike(thread, d, src)
		if err != nil {
			if strings.Contains(err.Error(), "nested too deeply") {
				return
			}
			t.Fatalf("encodeLike(%q): %v", src, err)
		}
		if got != src {
			t.Fatalf("encodeLike(%q) = %q", src, got)
		}

		// An edited value must come back exactly when the output is decoded.
		added, _ := starlark.Call(thread, starjson.Module.Members["decode"], starlark.Tuple{starlark.String(`[1, {"k": ["v", null]}]`)}, nil)
		switch c := d.(type) {
		case *starlark.Dict:
			_ = c.SetKey(starlark.String("fuzz"), added)
			if c.Len() > 1 {
				first := c.Keys()[0]
				_, _, _ = c.Delete(first)
			}
		case *starlark.List:
			_ = c.Append(added)
		default:
			return
		}
		out, err := encodeLike(thread, d, src)
		if err != nil {
			return
		}
		back, err := starlark.Call(thread, starjson.Module.Members["decode"], starlark.Tuple{starlark.String(out)}, nil)
		if err != nil {
			t.Fatalf("edit of %q gave invalid JSON %q: %v", src, out, err)
		}
		if eq, _ := starlark.Equal(back, d); !eq {
			t.Fatalf("edit of %q gave %q", src, out)
		}
	})
}
