// Package examples_test checks that every example campaign loads and that
// its change script does what the campaign says, on a scratch repo.
package examples_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rossedman/gh-fleet/internal/campaign"
)

// scratch writes files into a temp dir and returns it.
func scratch(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
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

// runScript loads a campaign and runs its script in dir.
func runScript(t *testing.T, name, dir string, env ...string) {
	t.Helper()
	c, err := campaign.Load(filepath.Join(name, "fleet.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(c.Change.Shell, c.ScriptFile())
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	for k, v := range c.Change.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s: %v\n%s", name, err, out)
	}
}

// read returns a file's content.
func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestExamplesLoad(t *testing.T) {
	t.Parallel()
	dirs, _ := filepath.Glob("*/fleet.yaml")
	if len(dirs) != 13 {
		t.Fatalf("want 13 examples, got %v", dirs)
	}
	for _, f := range dirs {
		c, err := campaign.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		if c.ScriptFile() == "" {
			t.Errorf("%s: change.run is not a script file", f)
		}
		if _, err := c.RenderBody(campaign.RepoData{Owner: "acme", Name: "api", FullName: "acme/api"}); err != nil {
			t.Errorf("%s: %v", f, err)
		}
		if _, err := c.RenderTitle(campaign.RepoData{Name: "api"}); err != nil {
			t.Errorf("%s: %v", f, err)
		}
	}
}

func TestPinActions(t *testing.T) {
	t.Parallel()
	// A stub gh that resolves every ref to a fixed SHA.
	bin := t.TempDir()
	stub := "#!/bin/sh\necho 1111111111111111111111111111111111111111\n"
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(stub), 0o755); err != nil {
		t.Fatal(err)
	}
	workflow := `jobs:
  build:
    steps:
      - uses: actions/checkout@v4
      - uses: github/codeql-action/init@v3 # keep
      - uses: ./local-action
      - uses: docker://alpine:3
      - uses: actions/setup-go@0aaccfd150d50ccaeb58ebd88d36e91967a5f35b # v5.4.0
`
	dir := scratch(t, map[string]string{".github/workflows/ci.yml": workflow})
	runScript(t, "pin-actions", dir, "PATH="+bin+":"+os.Getenv("PATH"))
	got := read(t, filepath.Join(dir, ".github/workflows/ci.yml"))
	for _, want := range []string{
		"- uses: actions/checkout@1111111111111111111111111111111111111111 # v4\n",
		"- uses: github/codeql-action/init@1111111111111111111111111111111111111111 # v3\n",
		"- uses: ./local-action\n",
		"- uses: docker://alpine:3\n",
		"- uses: actions/setup-go@0aaccfd150d50ccaeb58ebd88d36e91967a5f35b # v5.4.0\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	// No workflows: nothing to do.
	runScript(t, "pin-actions", t.TempDir())
}

func TestAddDependabot(t *testing.T) {
	t.Parallel()
	dir := scratch(t, map[string]string{"go.mod": "module x\n", "Dockerfile": "FROM scratch\n", ".github/workflows/ci.yml": "on: push\n"})
	runScript(t, "add-dependabot", dir)
	got := read(t, filepath.Join(dir, ".github/dependabot.yml"))
	for _, want := range []string{"package-ecosystem: gomod", "package-ecosystem: docker", "package-ecosystem: github-actions", `prefix: "ci(deps)"`, `prefix: "build(deps)"`} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "npm") {
		t.Error("npm added without package.json")
	}
	// An existing config is left alone.
	existing := scratch(t, map[string]string{"go.mod": "module x\n", ".github/dependabot.yml": "version: 2\n"})
	runScript(t, "add-dependabot", existing)
	if read(t, filepath.Join(existing, ".github/dependabot.yml")) != "version: 2\n" {
		t.Error("existing config overwritten")
	}
	empty := t.TempDir()
	runScript(t, "add-dependabot", empty)
	if _, err := os.Stat(filepath.Join(empty, ".github")); err == nil {
		t.Error("config written for a repo with no ecosystems")
	}
}

func TestBumpGoVersion(t *testing.T) {
	t.Parallel()
	dir := scratch(t, map[string]string{
		"go.mod":                   "module x\n\ngo 1.22.1\n\ntoolchain go1.22.5\n",
		"tools/go.mod":             "module y\n\ngo 1.28\n",
		".github/workflows/ci.yml": "      - uses: actions/setup-go@v5\n        with:\n          go-version: '1.22'\n",
	})
	runScript(t, "bump-go-version", dir)
	if got := read(t, filepath.Join(dir, "go.mod")); got != "module x\n\ngo 1.27.3\n\n" {
		t.Errorf("go.mod = %q", got)
	}
	if got := read(t, filepath.Join(dir, "tools/go.mod")); !strings.Contains(got, "go 1.28") {
		t.Errorf("newer go directive lowered: %q", got)
	}
	if got := read(t, filepath.Join(dir, ".github/workflows/ci.yml")); !strings.Contains(got, "go-version: '1.27.3'") {
		t.Errorf("workflow = %q", got)
	}
}
