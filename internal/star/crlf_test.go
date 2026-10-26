package star

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCRLFFiles runs line-based helpers on files with Windows line
// endings: each must come out as the same edit as on \n files, with every
// line still ending in \r\n.
func TestCRLFFiles(t *testing.T) {
	t.Parallel()
	files := map[string]string{
		".github/workflows/ci.yml": "on: push\njobs:\n  test:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: actions/checkout@v3\n",
		"config.yaml":              "a:\n  b: 1  # keep\n",
		"pyproject.toml":           "[tool.ruff]\nline-length = 88\n",
		"README.md":                "# api\n\nIntro.\n",
		".gitignore":               "node_modules\n",
		"package.json":             "{\n  \"name\": \"x\",\n  \"keywords\": [\"a\"]\n}\n",
		"Dockerfile":               "FROM node:18\nRUN x\n",
		".tool-versions":           "nodejs 18.0.0\n",
		"notes.txt":                "one\n",
	}
	src := `load("@fleet//gha.star", "bump_action", "ensure_permissions")
load("@fleet//yamlpath.star", yset = "set")
load("@fleet//toml.star", tset = "set")
load("@fleet//markdown.star", "ensure_badge", "set_section")
load("@fleet//files.star", "ensure_lines")
load("@fleet//jsonfile.star", jset = "set")
load("@fleet//docker.star", "bump_base_image")
load("@fleet//runtime.star", "set_version")
def change(repo):
    bump_action(repo, "actions/checkout", "v4")
    ensure_permissions(repo)
    yset(repo, "config.yaml", "a.c", 2)
    yset(repo, "config.yaml", "a.b", 3)
    tset(repo, "pyproject.toml", "tool.ruff", "line-length", 100)
    tset(repo, "pyproject.toml", "tool.ruff.lint", "select", ["E"])
    set_section(repo, "README.md", "Security", "Report privately.")
    ensure_badge(repo, "README.md", "[![CI](x)](y)")
    ensure_lines(repo, ".gitignore", [".env"])
    jset(repo, "package.json", "version", "1.0.0")
    bump_base_image(repo, "node", "18", "22")
    set_version(repo, "node", "22.1.0")
    repo.append("notes.txt", "two\n")
    repo.set_var("read", repo.read("notes.txt"))
`
	lf, _, _, err := run(t, src, files, RepoInfo{})
	if err != nil {
		t.Fatal(err)
	}
	crlfFiles := map[string]string{}
	for p, c := range files {
		crlfFiles[p] = strings.ReplaceAll(c, "\n", "\r\n")
	}
	crlf, res, _, err := run(t, src, crlfFiles, RepoInfo{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Vars["read"] != "one\ntwo\n" {
		t.Errorf("read of a CRLF file = %q", res.Vars["read"])
	}
	for p := range files {
		want, _ := os.ReadFile(filepath.Join(lf, p))
		got, _ := os.ReadFile(filepath.Join(crlf, p))
		if string(want) == files[p] {
			t.Errorf("%s: the helper didn't change it, so this checks nothing", p)
		}
		if string(got) != strings.ReplaceAll(string(want), "\n", "\r\n") {
			t.Errorf("%s:\n got %q\nwant %q", p, got, strings.ReplaceAll(string(want), "\n", "\r\n"))
		}
	}

	// A second run on the CRLF result changes nothing.
	again := map[string]string{}
	for p := range files {
		b, _ := os.ReadFile(filepath.Join(crlf, p))
		again[p] = string(b)
	}
	dir, _, log, err := run(t, src, again, RepoInfo{})
	if err != nil {
		t.Fatal(err)
	}
	for p, c := range again {
		if b, _ := os.ReadFile(filepath.Join(dir, p)); string(b) != c && p != "notes.txt" {
			t.Errorf("%s changed on a second run\n%s", p, log)
		}
	}
}

func TestUsesCRLF(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]bool{
		"a\r\nb\r\n":     true,
		"a\r\nb\n":       true,
		"a\nb\nc\r\n":    false,
		"a\nb\n":         false,
		"":               false,
		"bin\x00\r\n":    false,
		"\xff\xfe\r\n\n": false,
	} {
		if got := usesCRLF([]byte(in)); got != want {
			t.Errorf("usesCRLF(%q) = %v", in, got)
		}
	}
	// Binary content is read and written as is.
	dir, res, _, err := run(t, "def change(repo):\n    repo.set_var('n', len(repo.read('b.bin')))\n    repo.write('b.bin', repo.read('b.bin') + 'x')\n", map[string]string{"b.bin": "\x00\r\n"}, RepoInfo{})
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "b.bin")); string(b) != "\x00\r\nx" || res.Vars["n"] != int64(3) {
		t.Errorf("binary = %q, n = %v", b, res.Vars["n"])
	}
}
