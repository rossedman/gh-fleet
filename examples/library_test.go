package examples_test

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
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

// runTwice runs a campaign's script, then runs it again and fails if the
// second run changed anything: a re-run of a campaign must be a no-op.
func runTwice(t *testing.T, name, dir string, env ...string) {
	t.Helper()
	runScript(t, name, dir, env...)
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
}

// gitRepo makes dir a git repository with every file committed.
func gitRepo(t *testing.T, dir string) {
	t.Helper()
	for _, args := range [][]string{{"init", "-q"}, {"add", "-A"}, {"-c", "user.name=t", "-c", "user.email=t@example.com", "commit", "-qm", "init"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
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
		"README.md":     "See https://docs.acme.io/start and https://docs.acme.io.\nNot https://docs.acmeXio.\n",
		"src/app.ts":    "const u = 'https://docs.acme.io/api'\n",
		"logo.bin":      "\x00https://docs.acme.io\x00",
		"untracked.txt": "https://docs.acme.io\n",
	})
	gitRepo(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("https://docs.acme.io\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runTwice(t, "find-replace", dir)
	if got := read(t, filepath.Join(dir, "README.md")); got != "See https://docs.acme.dev/start and https://docs.acme.dev.\nNot https://docs.acmeXio.\n" {
		t.Errorf("README = %q", got)
	}
	if !strings.Contains(read(t, filepath.Join(dir, "src/app.ts")), "docs.acme.dev/api") {
		t.Error("nested file not replaced")
	}
	if !strings.Contains(read(t, filepath.Join(dir, "logo.bin")), "docs.acme.io") {
		t.Error("binary file changed")
	}
	if !strings.Contains(read(t, filepath.Join(dir, "new.txt")), "docs.acme.io") {
		t.Error("untracked file changed")
	}
	// No matches at all is a no-op, not a failure.
	clean := scratch(t, map[string]string{"a.txt": "nothing here\n"})
	gitRepo(t, clean)
	runScript(t, "find-replace", clean)
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
	cmd := exec.Command("bash", "remove-files/scripts/remove.sh")
	cmd.Dir = t.TempDir()
	cmd.Env = append(os.Environ(), "REMOVE=../escape")
	if err := cmd.Run(); err == nil {
		t.Error("path outside the repo accepted")
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
	runScript(t, "go-module-bump", dir, path)
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
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq not installed")
	}
	path, log := stub(t, "npm")
	dir := scratch(t, map[string]string{
		"package.json":                  `{"dependencies":{"axios":"^0.27.2"}}`,
		"web/package.json":              `{"devDependencies":{"axios":"^1.2.0"}}`,
		"done/package.json":             `{"dependencies":{"axios":"^1.7.9"}}`,
		"none/package.json":             `{"dependencies":{"react":"^18"}}`,
		"web/node_modules/package.json": `{"dependencies":{"axios":"^0.1"}}`,
	})
	runScript(t, "npm-package-bump", dir, path)
	calls := read(t, log)
	want := []string{
		dir + " install axios@^1.7.9 --save-prod --package-lock-only --ignore-scripts --no-audit --no-fund",
		filepath.Join(dir, "web") + " install axios@^1.7.9 --save-dev --package-lock-only --ignore-scripts --no-audit --no-fund",
	}
	for _, w := range want {
		if !strings.Contains(calls, w+"\n") {
			t.Errorf("missing call %q in:\n%s", w, calls)
		}
	}
	if n := strings.Count(calls, "\n"); n != 2 {
		t.Errorf("want 2 npm calls, got %d:\n%s", n, calls)
	}
}

func TestAddCodeowners(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	runTwice(t, "add-codeowners", dir)
	if got := read(t, filepath.Join(dir, ".github/CODEOWNERS")); !strings.HasSuffix(got, "* @acme/platform\n") {
		t.Errorf("CODEOWNERS = %q", got)
	}
	existing := scratch(t, map[string]string{"docs/CODEOWNERS": "* @someone\n"})
	runScript(t, "add-codeowners", existing)
	if _, err := os.Stat(filepath.Join(existing, ".github/CODEOWNERS")); err == nil {
		t.Error("CODEOWNERS added next to an existing one")
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
