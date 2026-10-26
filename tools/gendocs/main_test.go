package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rossedman/gh-fleet/internal/campaign"
	"github.com/rossedman/gh-fleet/internal/star"
)

func TestRunWritesEveryPage(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := run(dir, "", "../../examples"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"campaign.md", "starlark-library.md", "starlark/files.md", "starlark/gha.md", "commands/gh_fleet.md", "commands/gh_fleet_apply.md", "commands/gh_fleet_exec.md", "examples/pin-actions.md", "report/report.html"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("%s not written", name)
		}
	}
	file := filepath.Join(dir, "campaign.md")
	if err := run(filepath.Join(file, "x"), "", ""); err == nil {
		t.Fatal("expected error for an unwritable dir")
	}
}

func TestSchemaMarkdown(t *testing.T) {
	t.Parallel()
	b, err := SchemaMarkdown(campaign.SchemaJSON)
	if err != nil {
		t.Fatal(err)
	}
	page := string(b)
	for _, want := range []string{
		"| `apiVersion` (required) | string | `fleet.gh/v1` |",
		"## `targets.include`",
		"| `repos` | list of repo or pattern |",
		"| `names` | list of repo or pattern |",
		"| `env` | map of string |",
		"| `concurrency` | integer | `8` |",
		"Between 1 and 256.",
		"At least 0.",
		"| `base` | string | \"\" |",
		"## `pr.commit`",
		"## `change.steps[]`",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("reference missing %q", want)
		}
	}
	if _, err := SchemaMarkdown([]byte("{")); err == nil {
		t.Fatal("expected error for bad JSON")
	}
	if typeOf(&schemaNode{}, nil) != "-" || cell("a|b\nc") != `a\|b c` {
		t.Fatal("helpers")
	}
}

func TestLibPages(t *testing.T) {
	t.Parallel()
	docs, err := libDocs()
	if err != nil {
		t.Fatal(err)
	}
	used, err := usedIn("../../examples")
	if err != nil {
		t.Fatal(err)
	}
	if got := used["gha.star/bump_action"]; len(got) == 0 || got[0] != "upgrade-action" {
		t.Errorf("used[gha.star/bump_action] = %v", got)
	}
	index := string(LibIndex(docs))
	for _, want := range []string{
		"| [`files.star`](starlark/files.md) |",
		"| [`files.ensure_lines`](starlark/files.md#ensure_lines) |",
		"every example on these pages runs as a test",
	} {
		if !strings.Contains(index, want) {
			t.Errorf("index missing %q", want)
		}
	}
	var files star.LibDoc
	for _, d := range docs {
		if d.Name == "files.star" {
			files = d
		}
	}
	page := string(LibPage(files, used))
	for _, want := range []string{
		"# files.star",
		"## ensure_lines\n\n```python\nensure_lines(repo, path, lines, header = None)\n```",
		"| `repo` | The repo passed to `change(repo)`. |",
		"| `path` | The file to edit, relative to the repo root. |",
		"**Returns:** The list of lines that were added",
		"# returns [\".env.local\"]",
		"=== \"Before\"\n\n    `.gitignore`\n\n    ```text\n    node_modules\n",
		"`files/.editorconfig` next to the campaign file",
		"**Used in:** [`gitignore-entries`](../examples/gitignore-entries.md)",
		"[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/files.star#L",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("files page missing %q", want)
		}
	}
	page = string(LibPage(star.LibDoc{Name: "x.star", Summary: "S.", Funcs: []star.LibFunc{{
		Name: "f", Signature: "f(repo)", Params: []string{"repo"}, Summary: "F.", Line: 3,
		Example: &star.LibExample{Call: "f(repo)", Deleted: []string{"old.txt"}},
	}}}, nil))
	if !strings.Contains(page, "`old.txt` is deleted.") || strings.Contains(page, "Used in") {
		t.Errorf("deleted example:\n%s", page)
	}
	if _, err := usedIn("[bad"); err == nil {
		t.Error("bad glob accepted")
	}
}

func TestUpdateNav(t *testing.T) {
	t.Parallel()
	entries := []string{"Overview: reference/starlark-library.md", "a.star: reference/starlark/a.md", "b.star: reference/starlark/b.md"}
	p := filepath.Join(t.TempDir(), "mkdocs.yml")
	in := "nav:\n  - Lib:\n      " + navBegin + "\n      - old: x.md\n      " + navEnd + "\n  - Next: n.md\n"
	if err := os.WriteFile(p, []byte(in), 0o644); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := updateNavBlock(p, navBegin, navEnd, entries); err != nil {
			t.Fatal(err)
		}
	}
	want := "nav:\n  - Lib:\n      " + navBegin + "\n      - Overview: reference/starlark-library.md\n      - a.star: reference/starlark/a.md\n      - b.star: reference/starlark/b.md\n      " + navEnd + "\n  - Next: n.md\n"
	if got, _ := os.ReadFile(p); string(got) != want {
		t.Errorf("mkdocs.yml =\n%s", got)
	}
	if err := os.WriteFile(p, []byte("nav: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := updateNavBlock(p, navBegin, navEnd, entries); err == nil || !strings.Contains(err.Error(), "block") {
		t.Errorf("missing markers: %v", err)
	}
	if err := updateNavBlock(filepath.Join(t.TempDir(), "none.yml"), navBegin, navEnd, entries); err == nil {
		t.Error("missing file accepted")
	}
	dir := t.TempDir()
	if err := writeLib(dir, filepath.Join(dir, "none.yml"), nil); err == nil {
		t.Error("writeLib ignored a missing mkdocs.yml")
	}
	if err := writeExamples(dir, filepath.Join(dir, "none.yml"), "../../examples", nil); err == nil {
		t.Error("writeExamples ignored a missing mkdocs.yml")
	}
}

func TestFormatting(t *testing.T) {
	t.Parallel()
	for path, want := range map[string]string{
		"Dockerfile": "dockerfile", "web/app.Dockerfile": "dockerfile", "go.mod": "text", ".nvmrc": "text",
		"ci.yml": "yaml", "a.json": "json", "Cargo.toml": "toml", "main.tf": "hcl", "README.md": "markdown",
		"app.ts": "typescript", "requirements.txt": "text",
	} {
		if got := lang(path); got != want {
			t.Errorf("lang(%s) = %s, want %s", path, got, want)
		}
	}
	if got := fence("md", "a\n```\nb\n\nc\n", "  "); got != "  ````md\n  a\n  ```\n  b\n\n  c\n  ````\n" {
		t.Errorf("fence = %q", got)
	}
	if got := wrapSignature(`bump_image(repo, image, from_tag, to_tag, include = ["**/*.yml", "**/*.yaml"], exclude = [])`); got != "bump_image(\n    repo,\n    image,\n    from_tag,\n    to_tag,\n    include = [\"**/*.yml\", \"**/*.yaml\"],\n    exclude = [],\n)" {
		t.Errorf("wrapSignature = %q", got)
	}
	if got := wrapSignature("f(repo)"); got != "f(repo)" {
		t.Errorf("short signature wrapped: %q", got)
	}
	if sentence("") != "" || sentence("x.") != "x." || sentence("x") != "x." || upperFirst("") != "" || upperFirst("ab") != "Ab" {
		t.Error("text helpers")
	}
}
