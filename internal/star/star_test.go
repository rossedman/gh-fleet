package star

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rossedman/gh-fleet/internal/testutil"
)

// files writes name→content into a new temp dir and returns it.
func files(t *testing.T, m map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range m {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// load compiles change.star from a campaign dir built from m.
func load(t *testing.T, m map[string]string, opts Options) (*Script, error) {
	t.Helper()
	dir := files(t, m)
	return Load(context.Background(), filepath.Join(dir, "change.star"), dir, opts)
}

// run compiles src and runs it against a repo dir built from repoFiles.
func run(t *testing.T, src string, repoFiles map[string]string, info RepoInfo) (string, Result, string, error) {
	t.Helper()
	s, err := load(t, map[string]string{"change.star": src}, Options{Env: map[string]string{"MODE": "sha"}})
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	dir := files(t, repoFiles)
	var log bytes.Buffer
	res, err := s.Run(context.Background(), dir, info, &log, time.Minute)
	return dir, res, log.String(), err
}

func TestLoadErrors(t *testing.T) {
	t.Parallel()
	for name, tt := range map[string]struct {
		files map[string]string
		want  string
	}{
		"syntax":         {map[string]string{"change.star": "def change(repo):\n  x = \n"}, "change.star:2:7: got newline"},
		"undefined":      {map[string]string{"change.star": "def change(repo):\n  nope()\n"}, "change.star:2:3: undefined: nope"},
		"no change":      {map[string]string{"change.star": "x = 1\n"}, "no change(repo) function defined"},
		"not a function": {map[string]string{"change.star": "change = 1\n"}, "no change(repo) function defined"},
		"params":         {map[string]string{"change.star": "def change(a, b):\n  pass\n"}, "change must take exactly one parameter"},
		"top-level fail": {map[string]string{"change.star": "fail('boom')\ndef change(repo):\n  pass\n"}, "boom"},
		"outside":        {map[string]string{"change.star": "load('../x.star', 'a')\n"}, `load "../x.star": ../x.star is outside`},
		"bare name":      {map[string]string{"change.star": "load('lib.star', 'a')\n"}, `relative to this one`},
		"not star":       {map[string]string{"change.star": "load('./lib.py', 'a')\n"}, "only .star files"},
		"branch ref":     {map[string]string{"change.star": "load('github.com/a/b@main:x.star', 'a')\n"}, "full commit SHA"},
		"missing":        {map[string]string{"change.star": "load('./lib.star', 'a')\n"}, "no such file"},
		"cycle":          {map[string]string{"change.star": "load('./a.star', 'x')\n", "a.star": "load('./b.star', 'y')\nx = 1\n", "b.star": "load('./a.star', 'x')\ny = 1\n"}, "load cycle"},
		"bad symbol":     {map[string]string{"change.star": "load('./a.star', 'nope')\ndef change(repo):\n  pass\n", "a.star": "x = 1\n"}, "nope"},
		"lib syntax":     {map[string]string{"change.star": "load('./lib/a.star', 'x')\n", "lib/a.star": "x = (\n"}, "lib/a.star:2:1"},
	} {
		_, err := load(t, tt.files, Options{})
		if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: err = %v, want %q", name, err, tt.want)
		}
	}
	if _, err := Load(context.Background(), "/nonexistent/change.star", "/nonexistent", Options{}); err == nil {
		t.Fatal("missing script accepted")
	}
}

func TestLocalLoads(t *testing.T) {
	t.Parallel()
	s, err := load(t, map[string]string{
		"change.star":   "load('./lib/util.star', 'ensure')\ndef change(repo):\n  ensure(repo, 'a.txt', 'hi\\n')\n",
		"lib/util.star": "load('./more.star', 'join')\ndef ensure(repo, path, content):\n  if not repo.exists(path):\n    repo.write(path, join(content))\n",
		"lib/more.star": "def join(s):\n  return s\n",
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if _, err := s.Run(context.Background(), dir, RepoInfo{}, nil, 0); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "a.txt")); string(b) != "hi\n" {
		t.Fatalf("a.txt = %q", b)
	}
}

func TestRemoteLoad(t *testing.T) {
	t.Parallel()
	env := testutil.GitEnv(t)
	lib := testutil.NewRemote(t, env, map[string]string{
		"security.star": "load('./helpers.star', 'line')\ndef harden(repo):\n  repo.write('SECURITY.md', line())\n",
		"helpers.star":  "def line():\n  return 'report privately\\n'\n",
		"escape.star":   "load('../../x.star', 'y')\n",
	})
	sha := lib.Branch("main")
	cache := t.TempDir()
	opts := Options{CacheDir: cache, Git: testutil.Git(env), RemoteURL: func(host, owner, repo string) string {
		if host != "github.com" || owner != "acme" || repo != "fleet-lib" {
			t.Errorf("unexpected library %s/%s/%s", host, owner, repo)
		}
		return lib.URL()
	}}
	src := "load('github.com/acme/fleet-lib@" + sha + ":security.star', 'harden')\ndef change(repo):\n  harden(repo)\n"
	s, err := load(t, map[string]string{"change.star": src}, opts)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if _, err := s.Run(context.Background(), dir, RepoInfo{}, nil, 0); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "SECURITY.md")); string(b) != "report privately\n" {
		t.Fatalf("SECURITY.md = %q", b)
	}
	// The checkout is cached: a second load doesn't fetch.
	opts.RemoteURL = func(string, string, string) string { return "file:///nonexistent" }
	if _, err := load(t, map[string]string{"change.star": src}, opts); err != nil {
		t.Fatalf("cached library refetched: %v", err)
	}

	bad := strings.Repeat("0", 40)
	if _, err := load(t, map[string]string{"change.star": "load('github.com/acme/fleet-lib@" + bad + ":x.star', 'y')\n"}, Options{CacheDir: t.TempDir(), Git: testutil.Git(env), RemoteURL: func(string, string, string) string { return lib.URL() }}); err == nil || !strings.Contains(err.Error(), "fetch github.com/acme/fleet-lib@000000000000") {
		t.Fatalf("unknown commit: %v", err)
	}
	if _, err := load(t, map[string]string{"change.star": "load('github.com/acme/fleet-lib@" + sha + ":escape.star', 'y')\n"}, opts); err == nil || !strings.Contains(err.Error(), "outside") {
		t.Fatalf("library escaping its checkout: %v", err)
	}
	if _, err := load(t, map[string]string{"change.star": "load('github.com/acme/fleet-lib@" + sha + ":x.star', 'y')\n"}, Options{}); err == nil || !strings.Contains(err.Error(), "no cache directory") {
		t.Fatalf("no cache: %v", err)
	}
}

func TestRepoMethods(t *testing.T) {
	t.Parallel()
	src := `
def change(repo):
    lines = repo.read(".gitignore", default = "").splitlines()
    if ".env" not in lines:
        repo.append(".gitignore", ".env\n")
    repo.write("docs/new.md", "# %s for %s\n" % (repo.name, repo.vars["team"]))
    repo.write("same.txt", "same\n")
    if repo.exists("old.cfg", "nope"):
        repo.delete("old.cfg", "missing.cfg", "legacy")
    found = repo.glob("**/*.go", exclude = ["vendor/**"])
    repo.write("found.txt", ",".join(found) + "\n")
    repo.set_var("count", len(found))
    repo.set_var("names", [repo.owner, repo.full_name, repo.default_branch, repo.language] + repo.topics)
    repo.set_var("tier", repo.properties["tier"])
    repo.set_var("mode", env["MODE"])
    print("done", repo)
`
	dir, res, log, err := run(t, src, map[string]string{
		".gitignore": "node_modules\n", "same.txt": "same\n", "old.cfg": "x", "legacy/a": "y",
		"main.go": "", "pkg/a.go": "", "vendor/x/b.go": "", ".git/c.go": "", "run.sh": "#!/bin/sh\n",
	}, RepoInfo{Owner: "acme", Name: "api", DefaultBranch: "main", Language: "Go", Topics: []string{"service"},
		Properties: map[string]string{"tier": "1"}, Vars: map[string]any{"team": "payments"}})
	if err != nil {
		t.Fatal(err)
	}
	read := func(p string) string {
		b, _ := os.ReadFile(filepath.Join(dir, p))
		return string(b)
	}
	if read(".gitignore") != "node_modules\n.env\n" || read("docs/new.md") != "# api for payments\n" || read("found.txt") != "main.go,pkg/a.go\n" {
		t.Fatalf("files: %q %q %q", read(".gitignore"), read("docs/new.md"), read("found.txt"))
	}
	for _, gone := range []string{"old.cfg", "legacy"} {
		if _, err := os.Stat(filepath.Join(dir, gone)); err == nil {
			t.Errorf("%s not deleted", gone)
		}
	}
	if res.Vars["count"] != int64(2) || res.Vars["tier"] != "1" || res.Vars["mode"] != "sha" {
		t.Fatalf("vars = %v", res.Vars)
	}
	if names := res.Vars["names"].([]any); len(names) != 5 || names[1] != "acme/api" {
		t.Fatalf("names = %v", names)
	}
	for _, want := range []string{"starlark: append .gitignore (change.star:5:20)", "starlark: write docs/new.md (change.star:6:15)", "starlark: delete old.cfg", "done <repo acme/api>"} {
		if !strings.Contains(log, want) {
			t.Errorf("log missing %q:\n%s", want, log)
		}
	}
	if strings.Contains(log, "same.txt") {
		t.Error("unchanged write logged")
	}
}

func TestWriteKeepsMode(t *testing.T) {
	t.Parallel()
	dir, _, _, err := run(t, "def change(repo):\n  repo.write('run.sh', '#!/bin/sh\\necho hi\\n')\n", map[string]string{"run.sh": "#!/bin/sh\n"}, RepoInfo{})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(dir, "run.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
	s, _ := load(t, map[string]string{"change.star": "def change(repo):\n  repo.write('run.sh', 'changed\\n')\n"}, Options{})
	if _, err := s.Run(context.Background(), dir, RepoInfo{}, nil, 0); err != nil {
		t.Fatal(err)
	}
	if st, _ := os.Stat(filepath.Join(dir, "run.sh")); st.Mode().Perm() != 0o755 {
		t.Fatalf("mode = %v", st.Mode())
	}
}

func TestSandbox(t *testing.T) {
	t.Parallel()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret"), []byte("s"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct{ src, want string }{
		{"repo.read('../x')", `"../x" is not a path inside the repo`},
		{"repo.write('/etc/x', 'y')", `"/etc/x" is not a path inside the repo`},
		{"repo.read('')", "is not a path inside the repo"},
		{"repo.write('.git/config', 'x')", "is inside .git"},
		{"repo.delete('.git')", "is inside .git"},
		{"repo.write('./.git/hooks/pre-commit', 'x')", "is inside .git"},
		{"repo.write('a/../.git/config', 'x')", "is inside .git"},
		{"repo.read('link/secret')", "repo.read: link/secret:"},
		{"repo.write('link/new', 'x')", "repo.write: link/new:"},
		{"repo.read('missing')", "repo.read: missing: no such file"},
		{"repo.glob('[')", `invalid pattern "["`},
		{"repo.glob()", "needs at least one path"},
		{"repo.exists(1)", "argument 1 is int, want string"},
		{"repo.exists('a', x = 1)", "unexpected keyword argument"},
		{"repo.glob('*', exclude = 'a')", "exclude must be a list"},
		{"repo.glob('*', exclude = [1])", "exclude must be a list of strings"},
		{"repo.set_var('bad-name', 1)", "is not a valid variable name"},
		{"repo.set_var('x', {'a': 1})", "a variable is a string, number, boolean or list"},
		{"repo.set_var('x', repo)", "cannot use a repo here"},
		{"repo.nope", "has no .nope field or method"},
		{"{repo: 1}", "unhashable type: repo"},
		{"repo.properties['x'] = 1", "frozen"},
		{"fail('custom failure')", "custom failure"},
	} {
		s, err := load(t, map[string]string{"change.star": "def change(repo):\n  " + tt.src + "\n"}, Options{})
		if err != nil {
			t.Fatalf("%s: load: %v", tt.src, err)
		}
		dir := files(t, map[string]string{".git/config": "[core]\n"})
		if err := os.Symlink(outside, filepath.Join(dir, "link")); err != nil {
			t.Fatal(err)
		}
		_, err = s.Run(context.Background(), dir, RepoInfo{}, nil, 0)
		if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: err = %v, want %q", tt.src, err, tt.want)
		}
		if err != nil && strings.Contains(err.Error(), dir) {
			t.Errorf("%s: error leaks the clone path: %v", tt.src, err)
		}
	}
	if _, err := os.Stat(filepath.Join(outside, "new")); err == nil {
		t.Fatal("wrote through a symlink out of the repo")
	}
	if _, _, _, err := run(t, "def change(repo):\n  repo.glob('*', 1)\n", nil, RepoInfo{}); err == nil {
		t.Fatal("non-string glob pattern accepted")
	}
	s, _ := load(t, map[string]string{"change.star": "def change(repo):\n  pass\n"}, Options{})
	if _, err := s.Run(context.Background(), "/nonexistent/dir", RepoInfo{}, nil, 0); err == nil {
		t.Fatal("missing repo dir accepted")
	}
}

func TestLimits(t *testing.T) {
	t.Parallel()
	s, err := load(t, map[string]string{"change.star": "def change(repo):\n  while True:\n    pass\n"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Run(context.Background(), t.TempDir(), RepoInfo{}, nil, 50*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "starlark change timed out after 50ms") {
		t.Fatalf("timeout: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.Run(ctx, t.TempDir(), RepoInfo{}, nil, 0); err == nil || !strings.Contains(err.Error(), "canceled") {
		t.Fatalf("cancel: %v", err)
	}
	// Recursion is not allowed in this dialect.
	s, err = load(t, map[string]string{"change.star": "def f(n):\n  return f(n)\ndef change(repo):\n  f(1)\n"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Run(context.Background(), t.TempDir(), RepoInfo{}, nil, 0); err == nil || !strings.Contains(err.Error(), "called recursively") {
		t.Fatalf("recursion: %v", err)
	}
}

func TestModules(t *testing.T) {
	t.Parallel()
	src := `
def change(repo):
    pkg = json.decode(repo.read("package.json"))
    pkg["engines"] = {"node": ">=22"}
    repo.write("package.json", json.encode_indent(pkg, indent = "  ") + "\n")
    repo.set_var("compact", json.encode({"b": [1, 2.5, None, True], "a": "<&>"}))
    repo.set_var("indented", json.indent('{"z":1}', indent = " "))

    doc = yaml.decode(repo.read("ci.yml"))
    doc["permissions"] = {"contents": "read"}
    repo.write("ci.yml", yaml.encode(doc))
    repo.set_var("empty", yaml.decode("") == None)
    repo.set_var("alias", yaml.decode("a: &x [1, 2]\nb: *x\n")["b"])

    repo.set_var("cmp", [semver.compare("1.2.3", "v1.10.0"), semver.compare("2.0.0", "2.0.0")])
    repo.set_var("less", semver.less("v1.9.9", "1.10.0"))
    repo.set_var("valid", [semver.valid("1.2.3"), semver.valid("one")])
    repo.set_var("major", [semver.major("v3.4.5"), semver.major_minor("3.4.5")])
    repo.set_var("max", semver.max("1.2.0", "v1.10.0", "1.9.0"))
    repo.set_var("lowers", [semver.lowers(a, b) for a, b in [("^13.17", "^11.0"), ("3.15", "3.12"), ("v1.9", "v1.10.0"), ("main", "v2"), ("22-alpine", "22"), ("1.2", "1.2.0"), ("2024.01.15.1", "2024.01.15"), ("9f8e7d6c5b4a", "v1.2.0"), ("v9", "sha256:1234abcd")]])

    repo.set_var("re", [
        re.matches(r"^v\d+$", "v12"),
        re.search(r"\d+", "go 1.22 now"),
        re.search(r"x", "abc") == None,
        re.groups(r"(\w+)@(v\d+)", "uses: actions/checkout@v4"),
        re.groups(r"z", "abc") == None,
        re.find_all(r"\d", "a1b2"),
        re.split(r",\s*", "a, b,c"),
        re.sub(r"node:(\d+)", "node:22", "FROM node:18 AS a\nFROM node:20\n"),
        re.sub(r"(\w)", "${1}${1}", "abc", count = 2),
        re.sub(r"v(\d+)", lambda m: "v" + str(int(m[1]) + 1), "v1 v9 $"),
    ])
`
	dir, res, _, err := run(t, src, map[string]string{
		"package.json": `{"name": "web", "version": "1.0.0", "scripts": {"build": "tsc"}}`,
		"ci.yml":       "on: push\njobs:\n  test:\n    runs-on: ubuntu-latest\n",
	}, RepoInfo{})
	if err != nil {
		t.Fatal(err)
	}
	pkg, _ := os.ReadFile(filepath.Join(dir, "package.json"))
	if string(pkg) != "{\n  \"name\": \"web\",\n  \"version\": \"1.0.0\",\n  \"scripts\": {\n    \"build\": \"tsc\"\n  },\n  \"engines\": {\n    \"node\": \">=22\"\n  }\n}\n" {
		t.Fatalf("package.json keeps key order:\n%s", pkg)
	}
	ci, _ := os.ReadFile(filepath.Join(dir, "ci.yml"))
	// yaml.encode writes a whole new document: key order survives, but
	// formatting doesn't (here `on` is quoted, as YAML 1.1 reads it as a
	// boolean), and comments are dropped.
	if string(ci) != "\"on\": push\njobs:\n  test:\n    runs-on: ubuntu-latest\npermissions:\n  contents: read\n" {
		t.Fatalf("ci.yml:\n%s", ci)
	}
	check := func(name, want string) {
		t.Helper()
		if got := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(fmtAny(res.Vars[name]), "\n", "|"), "  ", " ")); got != want {
			t.Errorf("%s = %s, want %s", name, got, want)
		}
	}
	check("empty", "true")
	check("compact", `{"b":[1,2.5,null,true],"a":"<&>"}`)
	check("indented", `{| "z": 1|}`)
	check("alias", "[1 2]")
	check("cmp", "[-1 0]")
	check("less", "true")
	check("valid", "[true false]")
	check("major", "[3 3.4]")
	check("max", "v1.10.0")
	check("lowers", "[true true false false false false true false false]")
	check("re", "[true 1 true [checkout v4] true [1 2] [a b c] FROM node:22 AS a|FROM node:22| aabbc v2 v10 $]")
}

func TestModuleErrors(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct{ src, want string }{
		{"semver.compare('x', '1.0.0')", `"x" is not a semantic version`},
		{"semver.max()", "needs one or more versions"},
		{"semver.max('x')", "argument 1 is not a semantic version"},
		{"semver.valid()", "got 0 arguments, want 1"},
		{"semver.less('1.0.0')", "got 1 arguments, want 2"},
		{"semver.major('x')", "not a semantic version"},
		{"semver.major_minor('x')", "not a semantic version"},
		{"re.matches('(', 'x')", "missing closing )"},
		{"re.sub('a', 1, 'a')", "repl must be a string or a function, not int"},
		{"re.sub('a', lambda m: 1, 'a')", "repl function returned int, want string"},
		{"re.sub('(', 'x', 'y')", "missing closing )"},
		{"re.sub(1, 'x', 'y')", "for parameter pattern"},
		{"re.search('a')", "got 1 arguments, want 2"},
		{"re.groups('(', 'a')", "missing closing"},
		{"re.find_all('(', 'a')", "missing closing"},
		{"re.split('(', 'a')", "missing closing"},
		{"yaml.decode('a: [')", "yaml.decode:"},
		{"yaml.decode()", "got 0 arguments, want 1"},
		{"yaml.encode()", "got 0 arguments, want 1"},
		{"yaml.encode({'a': repo})", "cannot use a repo here"},
		{"yaml.encode([repo])", "cannot use a repo here"},
		{"yaml.encode({repo.name: repo})", "cannot use a repo here"},
		{"repo.set_var('x', float('inf'))", "not a finite number"},
		{"repo.set_var('x', {1: 2})", "is not a string"},
		{"repo.set_var('x', [repo])", "cannot use a repo here"},
		{"repo.read(1)", "for parameter path"},
		{"repo.write('a')", "missing argument for content"},
		{"repo.append('a')", "missing argument for content"},
		{"repo.set_var('x')", "missing argument for value"},
		{"repo.delete()", "needs at least one path"},
		{"repo.delete('../x')", "not a path inside the repo"},
		{"repo.exists('../x')", "not a path inside the repo"},
		{"repo.append('../x', 'y')", "not a path inside the repo"},
		{"json.encode(repo)", "cannot encode repo as JSON"},
		{"json.encode({1: 2})", "dict key 1 is not a string"},
		{"json.encode([float('nan')])", "non-finite float"},
		{"json.encode()", "got 0 arguments, want 1"},
		{"json.encode_indent(repo)", "cannot encode repo"},
		{"json.encode_indent()", "missing argument for x"},
	} {
		s, err := load(t, map[string]string{"change.star": "def change(repo):\n  " + tt.src + "\n"}, Options{})
		if err != nil {
			t.Fatalf("%s: load: %v", tt.src, err)
		}
		_, err = s.Run(context.Background(), t.TempDir(), RepoInfo{Name: "a"}, nil, 0)
		if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: err = %v, want %q", tt.src, err, tt.want)
		}
	}
}

func TestConversions(t *testing.T) {
	t.Parallel()
	if _, err := toStarlark(struct{}{}); err == nil {
		t.Fatal("unknown type converted")
	}
	if _, err := toStarlark([]any{struct{}{}}); err == nil {
		t.Fatal("unknown list element converted")
	}
	if _, err := toStarlark(map[string]any{"a": struct{}{}}); err == nil {
		t.Fatal("unknown map value converted")
	}
	for _, v := range []any{uint64(1), int64(2)} {
		if _, err := toStarlark(v); err != nil {
			t.Fatal(err)
		}
	}
	r := newRepo(nil, RepoInfo{Owner: "a", Name: "b"}, nil)
	if r.Type() != "repo" || !bool(r.Truth()) || len(r.AttrNames()) < 10 {
		t.Fatal("repo value")
	}
	r.Freeze()
	if v, _ := r.Attr("nope"); v != nil {
		t.Fatal("unknown attribute")
	}
	if relTo("/a", "/b/c") != "/b/c" {
		t.Fatal("relTo outside")
	}
}

// fmtAny prints a converted value compactly for comparisons.
func fmtAny(v any) string {
	switch t := v.(type) {
	case []any:
		parts := make([]string, len(t))
		for i, e := range t {
			parts[i] = fmtAny(e)
		}
		return "[" + strings.Join(parts, " ") + "]"
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case int64:
		return strconv.FormatInt(t, 10)
	}
	return "?"
}
