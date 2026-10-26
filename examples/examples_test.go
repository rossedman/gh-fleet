// Package examples_test checks that every example campaign loads and that
// its change script does what the campaign says, on a scratch repo.
package examples_test

import (
	"bytes"
	"context"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rossedman/gh-fleet/internal/campaign"
	"github.com/rossedman/gh-fleet/internal/runner"
	"github.com/rossedman/gh-fleet/internal/star"
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

// runSteps loads a campaign and runs its steps in order in dir, the way
// gh fleet does: each step sees the variables earlier steps set. env holds
// KEY=VALUE pairs: FLEET_REPO names the repo, FLEET_VAR_<NAME> sets a
// variable, and any other key overrides the campaign's env. It returns
// the variables the steps set.
func runSteps(t *testing.T, name, dir string, env ...string) (map[string]any, error) {
	t.Helper()
	c, err := campaign.Load(filepath.Join(name, "fleet.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	info := star.RepoInfo{Owner: "acme", Name: "web", Vars: map[string]any{}}
	overrides := map[string]string{}
	for _, kv := range env {
		k, v, _ := strings.Cut(kv, "=")
		switch {
		case k == "FLEET_REPO":
			info.Owner, info.Name, _ = strings.Cut(v, "/")
		case strings.HasPrefix(k, "FLEET_VAR_"):
			info.Vars[strings.ToLower(strings.TrimPrefix(k, "FLEET_VAR_"))] = v
		default:
			overrides[k] = v
		}
	}
	set := map[string]any{}
	ctx := context.Background()
	for _, st := range c.Change.Steps {
		stepEnv := c.StepEnv(st)
		maps.Copy(stepEnv, overrides)
		vars := maps.Clone(info.Vars)
		maps.Copy(vars, set)
		if st.Starlark != "" {
			s, err := star.Load(ctx, c.StarlarkFile(st), c.Dir(), star.Options{Env: stepEnv})
			if err != nil {
				t.Fatal(err)
			}
			run := info
			run.Vars = vars
			res, err := s.Run(ctx, dir, run, nil, 0)
			if err != nil {
				return nil, err
			}
			maps.Copy(set, res.Vars)
			continue
		}
		spec := runner.Spec{Shell: st.Shell, File: c.ScriptFile(st), Env: stepEnv}
		if spec.File == "" {
			spec.Script = st.Run
		}
		fleetEnv := []string{"FLEET_REPO=" + info.Owner + "/" + info.Name}
		for k, v := range vars {
			fleetEnv = append(fleetEnv, "FLEET_VAR_"+campaign.EnvName(k)+"="+campaign.FormatValue(v))
		}
		for _, kv := range env {
			if k, _, _ := strings.Cut(kv, "="); k == "PATH" {
				fleetEnv = append(fleetEnv, kv)
			}
		}
		var log bytes.Buffer
		res, err := runner.Run(ctx, dir, spec, fleetEnv, &log)
		if err != nil {
			return nil, fmt.Errorf("%w\n%s", err, log.String())
		}
		for k, v := range res.Vars {
			set[k] = v
		}
	}
	return set, nil
}

// runScript runs a campaign's steps in dir and fails the test on error.
func runScript(t *testing.T, name, dir string, env ...string) map[string]any {
	t.Helper()
	vars, err := runSteps(t, name, dir, env...)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return vars
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
	if len(dirs) != 46 {
		t.Fatalf("want 46 examples, got %v", dirs)
	}
	for _, f := range dirs {
		c, err := campaign.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		for i, st := range c.Change.Steps {
			// Shell stays a few lines of glue around a tool; the logic
			// belongs in Starlark.
			if st.Run != "" && strings.Count(strings.TrimSpace(st.Run), "\n") >= 6 {
				t.Errorf("%s: step %s has more than 6 lines of shell; move the logic to Starlark", f, st.Label(i))
			}
			if c.ScriptFile(st) != "" {
				t.Errorf("%s: step %s runs a script file; examples keep shell inline and short", f, st.Label(i))
			}
			if st.Name == "" {
				t.Errorf("%s: step %d has no name", f, i)
			}
		}
		vars := map[string]any{"node": 22, "previous": "unset", "files": "requirements.txt", "jobs": "test", "license": "MIT", "overlays": "deploy/prod", "resources": "CronJob/nightly"}
		for k, v := range c.Vars.Default {
			vars[k] = v
		}
		repo := campaign.RepoData{Owner: "acme", Name: "api", FullName: "acme/api", Vars: vars}
		if _, err := c.RenderBody(repo); err != nil {
			t.Errorf("%s: %v", f, err)
		}
		if _, err := c.RenderTitle(repo); err != nil {
			t.Errorf("%s: %v", f, err)
		}
		if _, err := c.RenderList("reviewers", c.PR.Reviewers, repo); err != nil {
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
	vars := runTwice(t, "pin-actions", dir, "PATH="+bin+":"+os.Getenv("PATH"))
	if got := fmt.Sprint(vars["refs"]); got != "[actions/checkout@v4 github/codeql-action@v3]" {
		t.Errorf("refs = %s", got)
	}
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
	vars := runTwice(t, "add-dependabot", dir)
	if vars["ecosystems"] != "gomod, docker, github-actions" {
		t.Errorf("ecosystems = %v", vars["ecosystems"])
	}
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
	runTwice(t, "bump-go-version", dir)
	if got := read(t, filepath.Join(dir, "go.mod")); got != "module x\n\ngo 1.27.3\n" {
		t.Errorf("go.mod = %q", got)
	}
	if got := read(t, filepath.Join(dir, "tools/go.mod")); !strings.Contains(got, "go 1.28") {
		t.Errorf("newer go directive lowered: %q", got)
	}
	if got := read(t, filepath.Join(dir, ".github/workflows/ci.yml")); !strings.Contains(got, "go-version: '1.27.3'") {
		t.Errorf("workflow = %q", got)
	}
}
