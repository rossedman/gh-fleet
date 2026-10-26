package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTree writes files under dir.
func writeTree(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestExamplePage(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"demo/fleet.yaml":                    "name: demo\ndescription: Do the thing\n",
		"demo/change.star":                   "load(\"@fleet//files.star\", \"ensure_lines\")\n\ndef change(repo):\n    ensure_lines(repo, \".gitignore\", [\"x\"])\n",
		"demo/also.star":                     "def change(repo):\n    pass\n",
		"demo/files/a.txt":                   "a\n",
		"demo/logo.png":                      "\x89PNG\x00\x01",
		"demo/tests/edits/before/.gitignore": "y\n",
		"demo/tests/edits/after/.gitignore":  "y\nx\n",
		"demo/tests/alone/before/x.txt":      "x\n",
		"other/README.md":                    "not an example\n",
	})
	used := map[string][]string{"files.star/ensure_lines": {"demo"}, "files.star/LIMIT": {"demo"}, "files.star/sync": {"elsewhere"}}
	exs, err := readExamples(dir, used)
	if err != nil || len(exs) != 1 {
		t.Fatalf("examples = %+v, %v", exs, err)
	}
	ex := exs[0]
	ex.Funcs = map[string]bool{"files.star/ensure_lines": true}
	var order []string
	for _, f := range ex.Files {
		order = append(order, f.Path)
	}
	if got := strings.Join(order, " "); got != "change.star also.star fleet.yaml files/a.txt logo.png" {
		t.Errorf("file order = %s", got)
	}
	page := string(ExamplePage(ex))
	for _, want := range []string{
		"# demo\n\nDo the thing.\n",
		"**Uses:** [`files.LIMIT`](../starlark/files.md), [`files.ensure_lines`](../starlark/files.md#ensure_lines)\n",
		"gh fleet init --from demo --org your-org",
		"=== \"change.star\"\n\n    ```python\n    load(\"@fleet//files.star\", \"ensure_lines\")\n\n    def change(repo):\n",
		"=== \"files/a.txt\"\n\n    ```text\n    a\n    ```",
		"=== \"logo.png\"\n\n    _Empty or not text._",
		"### alone\n\n=== \"Before\"\n\n    `x.txt`",
		"No changes: the campaign leaves this repo alone.",
		"### edits\n\n=== \"Before\"\n\n    `.gitignore`\n\n    ```text\n    y\n    ```\n\n=== \"After\"\n\n    `.gitignore`\n\n    ```text\n    y\n    x\n    ```",
		"[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/demo)",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("page missing %q:\n%s", want, page)
		}
	}

	writeTree(t, dir, map[string]string{"bad/fleet.yaml": "name: [\n"})
	if _, err := readExamples(dir, nil); err == nil || !strings.Contains(err.Error(), "bad") {
		t.Errorf("bad fleet.yaml: %v", err)
	}
	if _, err := readExamples("[", nil); err == nil {
		t.Error("bad glob accepted")
	}
}
