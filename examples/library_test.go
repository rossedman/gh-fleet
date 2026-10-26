package examples_test

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rossedman/gh-fleet/internal/campaign"
)

// snapshot returns every file under dir (except .git) and its content.
func snapshot(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		out[rel] = string(b)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// runTwice runs a campaign's steps, then runs them again and fails if the
// second run changed anything: a re-run of a campaign must be a no-op. It
// returns the variables the first run set.
func runTwice(t *testing.T, name, dir string, env ...string) map[string]any {
	t.Helper()
	vars := runScript(t, name, dir, env...)
	before := snapshot(t, dir)
	runScript(t, name, dir, env...)
	after := snapshot(t, dir)
	for f, c := range after {
		if before[f] != c {
			t.Errorf("%s: second run changed %s", name, f)
		}
	}
	if len(before) != len(after) {
		t.Errorf("%s: second run added or removed files", name)
	}
	return vars
}

// stub puts a fake command on PATH that appends its arguments, and its
// working directory, to a log file. It returns the PATH entry and the log.
func stub(t *testing.T, name string) (string, string) {
	t.Helper()
	bin := t.TempDir()
	log := filepath.Join(bin, name+".log")
	script := "#!/bin/sh\necho \"$(pwd) $*\" >> " + log + "\n"
	if err := os.WriteFile(filepath.Join(bin, name), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return "PATH=" + bin + ":" + os.Getenv("PATH"), log
}

func TestSyncFiles(t *testing.T) {
	t.Parallel()
	dir := scratch(t, map[string]string{".editorconfig": "root = false\n", "main.go": "package main\n"})
	runTwice(t, "sync-files", dir)
	want := read(t, "sync-files/files/.editorconfig")
	if read(t, filepath.Join(dir, ".editorconfig")) != want {
		t.Error(".editorconfig not overwritten")
	}
	if !strings.Contains(read(t, filepath.Join(dir, ".github/pull_request_template.md")), "## What and why") {
		t.Error("PR template not copied")
	}
	if read(t, filepath.Join(dir, "main.go")) != "package main\n" {
		t.Error("unrelated file changed")
	}
}

func TestFindReplace(t *testing.T) {
	t.Parallel()
	dir := scratch(t, map[string]string{
		"README.md":                "See https://docs.acme.io/start and https://docs.acme.io.\nNot https://docs.acmeXio.\n",
		"src/app.ts":               "const u = 'https://docs.acme.io/api'\n",
		"logo.bin":                 "\x00https://docs.acme.io\x00",
		"node_modules/x/README.md": "https://docs.acme.io\n",
	})
	vars := runTwice(t, "find-replace", dir)
	if vars["files"] != int64(2) {
		t.Errorf("files = %v", vars["files"])
	}
	if got := read(t, filepath.Join(dir, "README.md")); got != "See https://docs.acme.dev/start and https://docs.acme.dev.\nNot https://docs.acmeXio.\n" {
		t.Errorf("README = %q", got)
	}
	if !strings.Contains(read(t, filepath.Join(dir, "src/app.ts")), "docs.acme.dev/api") {
		t.Error("nested file not replaced")
	}
	if !strings.Contains(read(t, filepath.Join(dir, "logo.bin")), "docs.acme.io") {
		t.Error("binary file changed")
	}
	if !strings.Contains(read(t, filepath.Join(dir, "node_modules/x/README.md")), "docs.acme.io") {
		t.Error("node_modules changed")
	}
	// No matches at all is a no-op, not a failure.
	runScript(t, "find-replace", scratch(t, map[string]string{"a.txt": "nothing here\n"}))
}

func TestRemoveFiles(t *testing.T) {
	t.Parallel()
	dir := scratch(t, map[string]string{".travis.yml": "language: go\n", ".travis/deploy.sh": "x\n", "keep.txt": "k\n"})
	runTwice(t, "remove-files", dir)
	for _, gone := range []string{".travis.yml", ".travis"} {
		if _, err := os.Stat(filepath.Join(dir, gone)); err == nil {
			t.Errorf("%s not removed", gone)
		}
	}
	if read(t, filepath.Join(dir, "keep.txt")) != "k\n" {
		t.Error("kept file changed")
	}
	if _, err := runSteps(t, "remove-files", t.TempDir(), "REMOVE=../escape"); err == nil || !strings.Contains(err.Error(), "not a path inside the repo") {
		t.Errorf("path outside the repo: %v", err)
	}
}

func TestDockerBaseImage(t *testing.T) {
	t.Parallel()
	dir := scratch(t, map[string]string{
		"Dockerfile":         "FROM node:18-alpine AS build\nRUN npm ci\nFROM --platform=linux/amd64 node:18\nFROM node:180\nFROM nginx:1.27\n",
		"web/Dockerfile.dev": "from node:18@sha256:abc\n",
		"docs/notes.md":      "FROM node:18\n",
	})
	runTwice(t, "docker-base-image", dir)
	if got := read(t, filepath.Join(dir, "Dockerfile")); got != "FROM node:22-alpine AS build\nRUN npm ci\nFROM --platform=linux/amd64 node:22\nFROM node:180\nFROM nginx:1.27\n" {
		t.Errorf("Dockerfile = %q", got)
	}
	if got := read(t, filepath.Join(dir, "web/Dockerfile.dev")); got != "from node:22@sha256:abc\n" {
		t.Errorf("Dockerfile.dev = %q", got)
	}
	if !strings.Contains(read(t, filepath.Join(dir, "docs/notes.md")), "node:18") {
		t.Error("non-Dockerfile changed")
	}
}

func TestUpgradeAction(t *testing.T) {
	t.Parallel()
	wf := "steps:\n  - uses: actions/checkout@v3\n  - uses: 'actions/checkout@v3' # quoted\n  - uses: actions/checkout@v3.5.0\n  - uses: actions/checkout-extra@v3\n  - uses: actions/setup-go@v3\n"
	dir := scratch(t, map[string]string{".github/workflows/ci.yml": wf, ".github/actions/setup/action.yaml": "runs:\n  steps:\n    - uses: actions/checkout@v3\n"})
	runTwice(t, "upgrade-action", dir)
	want := "steps:\n  - uses: actions/checkout@v4\n  - uses: 'actions/checkout@v4' # quoted\n  - uses: actions/checkout@v3.5.0\n  - uses: actions/checkout-extra@v3\n  - uses: actions/setup-go@v3\n"
	if got := read(t, filepath.Join(dir, ".github/workflows/ci.yml")); got != want {
		t.Errorf("workflow = %q", got)
	}
	if !strings.Contains(read(t, filepath.Join(dir, ".github/actions/setup/action.yaml")), "checkout@v4") {
		t.Error("composite action not upgraded")
	}
	runScript(t, "upgrade-action", t.TempDir())
}

func TestGitignoreEntries(t *testing.T) {
	t.Parallel()
	dir := scratch(t, map[string]string{".gitignore": "node_modules\n.env"})
	runTwice(t, "gitignore-entries", dir)
	if got := read(t, filepath.Join(dir, ".gitignore")); got != "node_modules\n.env\n\n# Local environment files\n.env.*\n!.env.example\n" {
		t.Errorf(".gitignore = %q", got)
	}
	empty := t.TempDir()
	runScript(t, "gitignore-entries", empty)
	if got := read(t, filepath.Join(empty, ".gitignore")); got != "# Local environment files\n.env\n.env.*\n!.env.example\n" {
		t.Errorf("new .gitignore = %q", got)
	}
}

func TestGoModuleBump(t *testing.T) {
	t.Parallel()
	path, log := stub(t, "go")
	dir := scratch(t, map[string]string{
		"go.mod":          "module a\n\nrequire (\n\tgolang.org/x/net v0.20.0 // indirect\n)\n",
		"tools/go.mod":    "module b\n\nrequire golang.org/x/net v0.40.0\n",
		"other/go.mod":    "module c\n\nrequire golang.org/x/text v0.1.0\n",
		"single/go.mod":   "module d\n\nrequire golang.org/x/net v0.9.0\n",
		"vendor/x/go.mod": "module e\n\nrequire golang.org/x/net v0.1.0\n",
	})
	vars := runScript(t, "go-module-bump", dir, path)
	if got := fmt.Sprint(vars["dirs"]); got != "[. single]" {
		t.Errorf("dirs = %s", got)
	}
	calls := read(t, log)
	for _, want := range []string{dir + " get golang.org/x/net@v0.33.0", dir + " mod tidy", filepath.Join(dir, "single") + " get golang.org/x/net@v0.33.0"} {
		if !strings.Contains(calls, want+"\n") {
			t.Errorf("missing call %q in:\n%s", want, calls)
		}
	}
	for _, not := range []string{"tools", "other", "vendor"} {
		if strings.Contains(calls, filepath.Join(dir, not)+" ") {
			t.Errorf("%s should be left alone:\n%s", not, calls)
		}
	}
}

func TestNpmPackageBump(t *testing.T) {
	t.Parallel()
	path, log := stub(t, "npm")
	dir := scratch(t, map[string]string{
		"package.json":                  "{\n  \"dependencies\": {\n    \"axios\": \"^0.27.2\"\n  }\n}\n",
		"package-lock.json":             "{}\n",
		"web/package.json":              "{\"devDependencies\":{\"axios\":\"^1.2.0\"}}",
		"web/package-lock.json":         "{}\n",
		"nolock/package.json":           "{\"dependencies\":{\"axios\":\"^1.2.0\"}}",
		"done/package.json":             "{\"dependencies\":{\"axios\":\"^1.7.9\"}}",
		"none/package.json":             "{\"dependencies\":{\"react\":\"^18\"}}",
		"web/node_modules/package.json": "{\"dependencies\":{\"axios\":\"^0.1\"}}",
	})
	vars := runScript(t, "npm-package-bump", dir, path)
	if got := read(t, filepath.Join(dir, "package.json")); got != "{\n  \"dependencies\": {\n    \"axios\": \"^1.7.9\"\n  }\n}\n" {
		t.Errorf("package.json = %q", got)
	}
	if got := read(t, filepath.Join(dir, "web/package.json")); !strings.Contains(got, `"axios":"^1.7.9"`) {
		t.Errorf("web/package.json = %q", got)
	}
	if got := read(t, filepath.Join(dir, "web/node_modules/package.json")); strings.Contains(got, "1.7.9") {
		t.Error("node_modules changed")
	}
	if got := fmt.Sprint(vars["dirs"]); got != "[. web]" {
		t.Errorf("dirs = %s", got)
	}
	calls := read(t, log)
	for _, d := range []string{dir, filepath.Join(dir, "web")} {
		if w := d + " install --package-lock-only --ignore-scripts --no-audit --no-fund\n"; !strings.Contains(calls, w) {
			t.Errorf("missing call %q in:\n%s", w, calls)
		}
	}
	if n := strings.Count(calls, "\n"); n != 2 {
		t.Errorf("want 2 npm calls, got %d:\n%s", n, calls)
	}
}

func TestAddCodeowners(t *testing.T) {
	t.Parallel()
	env := []string{"FLEET_REPO=acme/billing", "FLEET_VAR_TEAM=payments"}
	dir := t.TempDir()
	runTwice(t, "add-codeowners", dir, env...)
	if got := read(t, filepath.Join(dir, ".github/CODEOWNERS")); !strings.HasSuffix(got, "* @acme/payments\n") {
		t.Errorf("CODEOWNERS = %q", got)
	}
	existing := scratch(t, map[string]string{"docs/CODEOWNERS": "* @someone\n"})
	runScript(t, "add-codeowners", existing, env...)
	if _, err := os.Stat(filepath.Join(existing, ".github/CODEOWNERS")); err == nil {
		t.Error("CODEOWNERS added next to an existing one")
	}
	// The campaign's vars resolve per repo: the data file names billing's team.
	c, err := campaign.Load("add-codeowners/fleet.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if got := c.Vars.Row("acme/billing")["team"]; got != "payments" {
		t.Errorf("owners.csv team for acme/billing = %v", got)
	}
}

func TestWorkflowPermissions(t *testing.T) {
	t.Parallel()
	dir := scratch(t, map[string]string{
		".github/workflows/ci.yml":       "name: ci\non: push\n\njobs:\n  test:\n    permissions:\n      checks: write\n    runs-on: ubuntu-latest\n",
		".github/workflows/release.yaml": "on: push\npermissions:\n  contents: write\njobs:\n  r: {}\n",
		".github/workflows/odd.yml":      "# no jobs key\non: push\n",
	})
	runTwice(t, "workflow-permissions", dir)
	if got := read(t, filepath.Join(dir, ".github/workflows/ci.yml")); got != "name: ci\non: push\n\npermissions:\n  contents: read\n\njobs:\n  test:\n    permissions:\n      checks: write\n    runs-on: ubuntu-latest\n" {
		t.Errorf("ci.yml = %q", got)
	}
	if got := read(t, filepath.Join(dir, ".github/workflows/release.yaml")); strings.Contains(got, "contents: read") {
		t.Error("existing permissions overridden")
	}
	if got := read(t, filepath.Join(dir, ".github/workflows/odd.yml")); strings.Contains(got, "permissions") {
		t.Error("file without jobs changed")
	}
}
