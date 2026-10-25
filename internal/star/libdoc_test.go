package star

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// libDocs parses every built-in module.
func libDocs(t *testing.T) []LibDoc {
	t.Helper()
	var out []LibDoc
	for _, name := range LibModules() {
		src, err := LibSource(name)
		if err != nil {
			t.Fatal(err)
		}
		doc, err := ParseLibDoc(name, src)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, doc)
	}
	return out
}

// TestLibDocumented checks that every module and public function in the
// built-in library is fully documented, since the reference is built from
// the docstrings: a summary, every parameter but repo, what it returns,
// and an example.
func TestLibDocumented(t *testing.T) {
	t.Parallel()
	for _, doc := range libDocs(t) {
		if doc.Summary == "" {
			t.Errorf("%s has no module docstring", doc.Name)
		}
		for _, fn := range doc.Funcs {
			where := doc.Name + ": " + fn.Name
			if fn.Summary == "" {
				t.Errorf("%s has no docstring", where)
				continue
			}
			var want, got []string
			for _, p := range fn.Params {
				if p != "repo" {
					want = append(want, p)
				}
			}
			for _, a := range fn.Args {
				got = append(got, a.Name)
				if a.Doc == "" {
					t.Errorf("%s: argument %s has no description", where, a.Name)
				}
			}
			if !slices.Equal(want, got) {
				t.Errorf("%s: Args documents %v, want %v (every parameter but repo, in order)", where, got, want)
			}
			if fn.Returns == "" {
				t.Errorf("%s has no Returns section", where)
			}
			if fn.Example == nil {
				t.Errorf("%s has no Example section", where)
			}
		}
	}
}

// TestLibExamples runs every docstring example: it writes the "before"
// files, runs the call, and checks the "after" files and the returned
// value. It then runs the call again and checks that nothing is written,
// since every helper promises to be idempotent.
func TestLibExamples(t *testing.T) {
	t.Parallel()
	for _, doc := range libDocs(t) {
		var names []string
		for _, fn := range doc.Funcs {
			names = append(names, `"`+fn.Name+`"`)
		}
		for _, fn := range doc.Funcs {
			if fn.Example == nil {
				continue
			}
			t.Run(strings.TrimSuffix(doc.Name, ".star")+"."+fn.Name, func(t *testing.T) {
				t.Parallel()
				runLibExample(t, doc.Name, names, fn.Example)
			})
		}
	}
}

// runLibExample runs one docstring example.
func runLibExample(t *testing.T, module string, names []string, ex *LibExample) {
	t.Helper()
	call := strings.Split(ex.Call, "\n")
	last := len(call) - 1
	var body strings.Builder
	for _, l := range call[:last] {
		body.WriteString("    " + l + "\n")
	}
	body.WriteString("    result = " + call[last] + "\n")
	body.WriteString("    repo.set_var(\"result\", repr(result))\n")
	src := "load(\"" + LibPrefix + module + "\", " + strings.Join(names, ", ") + ")\n\ndef change(repo):\n" + body.String()

	camp := map[string]string{"change.star": src}
	for _, f := range ex.Campaign {
		camp[f.Path] = f.Content
	}
	s, err := load(t, camp, Options{})
	if err != nil {
		t.Fatalf("load: %v\n%s", err, src)
	}
	before := map[string]string{}
	for _, f := range ex.Before {
		before[f.Path] = f.Content
	}
	dir := files(t, before)
	info := RepoInfo{Owner: "acme", Name: "api", Vars: map[string]any{}}
	for i := range 2 {
		var log bytes.Buffer
		res, err := s.Run(context.Background(), dir, info, &log, time.Minute)
		if err != nil {
			t.Fatalf("run %d: %v", i+1, err)
		}
		if i == 0 && ex.Returns != "" {
			if got := res.Vars["result"]; got != ex.Returns {
				t.Errorf("returns %v, docstring says %s", got, ex.Returns)
			}
		}
		if i == 1 && strings.Contains(log.String(), "starlark: ") {
			t.Errorf("second run wrote files:\n%s", log.String())
		}
	}
	for _, f := range ex.After {
		got, err := os.ReadFile(filepath.Join(dir, f.Path))
		if err != nil {
			t.Errorf("after %s: %v", f.Path, err)
			continue
		}
		if string(got) != f.Content {
			t.Errorf("after %s =\n%s\ndocstring says\n%s", f.Path, got, f.Content)
		}
	}
	for _, p := range ex.Deleted {
		if _, err := os.Stat(filepath.Join(dir, p)); err == nil {
			t.Errorf("%s still exists; docstring says it is deleted", p)
		}
	}
	// Files the example doesn't mention in "after" must be unchanged.
	for _, f := range ex.Before {
		if slices.ContainsFunc(ex.After, func(a LibFile) bool { return a.Path == f.Path }) || slices.Contains(ex.Deleted, f.Path) {
			continue
		}
		if got, err := os.ReadFile(filepath.Join(dir, f.Path)); err == nil && string(got) != f.Content {
			t.Errorf("%s changed, but the docstring has no \"after %s:\" block", f.Path, f.Path)
		}
	}
}

func TestParseLibDocErrors(t *testing.T) {
	t.Parallel()
	for name, tt := range map[string]struct{ src, want string }{
		"repeated":   {"def f(repo):\n    \"\"\"S.\n\n    Returns:\n        x\n    Returns:\n        y\n    \"\"\"\n", "Returns: section repeated"},
		"indent":     {"def f(repo):\n    \"\"\"S.\n\n    Returns:\n    x\n    \"\"\"\n", "not indented four spaces"},
		"arg format": {"def f(repo, a):\n    \"\"\"S.\n\n    Args:\n        just text\n    \"\"\"\n", `want "name: description"`},
		"arg cont":   {"def f(repo, a):\n    \"\"\"S.\n\n    Args:\n          more\n    \"\"\"\n", "continuation line"},
		"stray":      {"def f(repo):\n    \"\"\"S.\n\n    Example:\n        x = 1\n    \"\"\"\n", "is not inside a block"},
		"no call":    {"def f(repo):\n    \"\"\"S.\n\n    Example:\n        returns:\n            1\n    \"\"\"\n", "no call: block"},
		"syntax":     {"def f(:\n", "got"},
	} {
		if _, err := ParseLibDoc("x.star", []byte(tt.src)); err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: err = %v, want %q", name, err, tt.want)
		}
	}
	doc, err := ParseLibDoc("x.star", []byte("\"\"\"Mod.\"\"\"\n\ndef f(repo, a, b = 1, *rest, **kw):\n    \"\"\"S.\n\n    Prose.\n\n    Args:\n        a: first,\n            continued.\n        b: second.\n    Returns:\n        Nothing.\n    Example:\n        campaign t.txt:\n            c\n        before x.txt:\n            1\n\n            2\n        call:\n            y = 1\n            f(repo, y)\n        after x.txt:\n        deleted z.txt\n        returns:\n            None\n    \"\"\"\n"))
	if err != nil {
		t.Fatal(err)
	}
	fn := doc.Funcs[0]
	if strings.Join(fn.Params, ",") != "repo,a,b,rest,kw" || fn.Body != "Prose." || fn.Args[0].Doc != "first, continued." || fn.Returns != "Nothing." {
		t.Fatalf("fn = %+v", fn)
	}
	e := fn.Example
	if e.Campaign[0].Content != "c\n" || e.Before[0].Content != "1\n\n2\n" || e.Call != "y = 1\nf(repo, y)" || e.After[0].Content != "" || e.Deleted[0] != "z.txt" || e.Returns != "None" {
		t.Fatalf("example = %+v", e)
	}
}
