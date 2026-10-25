package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rossedman/gh-fleet/internal/campaign"
)

func TestRunWritesEveryPage(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := run(dir); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"campaign.md", "starlark-library.md", "commands/gh_fleet.md", "commands/gh_fleet_apply.md", "commands/gh_fleet_exec.md"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("%s not written", name)
		}
	}
	if err := run("/dev/null/x"); err == nil {
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
		"| `repos` | list of `owner/name` |",
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

func TestLibMarkdown(t *testing.T) {
	t.Parallel()
	b, err := LibMarkdown()
	if err != nil {
		t.Fatal(err)
	}
	page := string(b)
	for _, want := range []string{
		"| [`@fleet//files.star`](#filesstar) |",
		"## gha.star",
		"### files.ensure_lines",
		"ensure_lines(repo, path, lines, header = None)",
		`load("@fleet//docker.star", "dockerfiles", "bump_base_image", "bump_image")`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("library reference missing %q", want)
		}
	}
	if strings.Contains(page, "._lines") {
		t.Error("private helpers documented")
	}
}
