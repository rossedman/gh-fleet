package campaign

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const specExample = `apiVersion: fleet.gh/v1
name: pin-actions
description: Pin all GitHub Actions to commit SHAs

targets:
  orgs: [acme]
  include:
    topics: [service]
    languages: [Go]
    files: [.github/workflows]
    search: "uses: actions/ path:.github/workflows"
    repos: [acme/legacy-api]
  exclude:
    repos: [acme/sandbox]
    archived: true
    forks: true
    topics: [no-fleet]

change:
  steps:
    - name: pin
      run: ./scripts/pin.sh
      timeout: 2m
  shell: bash
  env:
    PIN_MODE: sha

pr:
  title: "ci: pin GitHub Actions to SHAs"
  body: ./pr-body.md
  branch: fleet/pin-actions
  base: ""
  draft: false
  labels: [fleet, security]
  reviewers: []
  commit:
    message: "ci: pin GitHub Actions to SHAs"
    signoff: true

limits:
  concurrency: 8
  maxOpenPRs: 200
`

// writeCampaign writes files into a temp dir and returns the campaign path.
func writeCampaign(t *testing.T, yaml string, extra map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range extra {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	p := filepath.Join(dir, "fleet.yaml")
	if err := os.WriteFile(p, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadSpecExample(t *testing.T) {
	t.Parallel()
	p := writeCampaign(t, specExample, map[string]string{
		"scripts/pin.sh": "#!/bin/sh\n",
		"pr-body.md":     "Pins actions in {{ .Repo.FullName }} for {{ .Campaign.Name }}.\n",
	})
	c, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if c.Name != "pin-actions" || c.Targets.Orgs[0] != "acme" || c.Limits.MaxOpenPRs != 200 {
		t.Fatalf("unexpected campaign: %+v", c)
	}
	if c.Change.Steps[0].Timeout.Std() != 2*time.Minute || c.Change.Env["PIN_MODE"] != "sha" {
		t.Fatalf("change: %+v", c.Change)
	}
	if got := c.ScriptFile(c.Change.Steps[0]); got != filepath.Join(filepath.Dir(p), "scripts", "pin.sh") {
		t.Fatalf("ScriptFile = %q", got)
	}
	body, err := c.RenderBody(RepoData{FullName: "acme/api"})
	if err != nil {
		t.Fatal(err)
	}
	want := "Pins actions in acme/api for pin-actions.\n\n<!-- gh-fleet:campaign=pin-actions -->"
	if body != want {
		t.Fatalf("body = %q", body)
	}
	if c.Dir() != filepath.Dir(p) {
		t.Fatalf("Dir = %q", c.Dir())
	}
}

func TestDefaults(t *testing.T) {
	t.Parallel()
	c, err := Parse([]byte(`apiVersion: fleet.gh/v1
name: bump
targets: {orgs: [acme]}
change: {steps: [{run: "true"}]}
pr: {title: "chore: bump {{ .Repo.Name }}"}
`), "x.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if c.PR.Branch != "fleet/bump" || c.Change.Shell != "bash" ||
		c.Limits.Concurrency != 8 || !*c.Targets.Exclude.Archived || !*c.Targets.Exclude.Forks {
		t.Fatalf("defaults not applied: %+v", c)
	}
	if c.PR.Commit.Message != c.PR.Title {
		t.Fatalf("commit message default = %q", c.PR.Commit.Message)
	}
	if c.ScriptFile(c.Change.Steps[0]) != "" || c.Change.Steps[0].Shell != "bash" || c.Change.Steps[0].Timeout.Std() != DefaultTimeout {
		t.Fatal("inline script reported as file")
	}
	title, err := c.RenderTitle(RepoData{Name: "api"})
	if err != nil || title != "chore: bump api" {
		t.Fatalf("title = %q, %v", title, err)
	}
	msg, err := c.RenderCommitMessage(RepoData{Name: "api"})
	if err != nil || msg != "chore: bump api" {
		t.Fatalf("commit = %q, %v", msg, err)
	}
	body, err := c.RenderBody(RepoData{})
	if err != nil || body != Marker("bump") {
		t.Fatalf("empty body = %q, %v", body, err)
	}
	if (&Campaign{}).Dir() != "." {
		t.Fatal("Dir without path")
	}
}

func TestValidationErrorsHaveLines(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		yaml string
		want []string
	}{
		{
			name: "wrong types and unknown key",
			yaml: `apiVersion: fleet.gh/v1
name: Bad_Name
targets:
  orgs: acme
  bogus: 1
change:
  steps: [{run: "true", timeout: later}]
  timeout: 5m
pr:
  title: x
`,
			want: []string{
				"f.yaml:2:7: name:",
				"f.yaml:4:9: targets.orgs:",
				"f.yaml:5:3: targets.bogus: unknown field",
				"f.yaml:7:34: change.steps.0.timeout: use a Go duration",
				"f.yaml:8:3: change.timeout: unknown field",
			},
		},
		{
			name: "missing required",
			yaml: "apiVersion: fleet.gh/v1\nname: x\n",
			want: []string{"f.yaml:1:1: missing properties 'targets', 'change', 'pr'"},
		},
		{
			name: "no orgs or repos",
			yaml: "apiVersion: fleet.gh/v1\nname: x\ntargets:\n  include: {topics: [a]}\nchange: {steps: [{run: x}]}\npr: {title: x}\n",
			want: []string{"f.yaml:4:3: targets: needs orgs, include.repos, or both"},
		},
		{
			name: "bad branch",
			yaml: "apiVersion: fleet.gh/v1\nname: x\ntargets: {orgs: [a]}\nchange: {steps: [{run: x}]}\npr: {title: x, branch: 'a..b'}\n",
			want: []string{"f.yaml:5:24: pr.branch: is not a valid git branch name"},
		},
		{
			name: "syntax error",
			yaml: "name: [\n",
			want: []string{"f.yaml:1:1: did not find expected node content"},
		},
		{
			name: "empty",
			yaml: "",
			want: []string{"f.yaml: file is empty"},
		},
		{
			name: "wrong version",
			yaml: "apiVersion: v2\nname: x\ntargets: {orgs: [a]}\nchange: {steps: [{run: x}]}\npr: {title: x}\n",
			want: []string{"f.yaml:1:13: apiVersion:"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := Parse([]byte(tt.yaml), "f.yaml")
			var ve *ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("want ValidationError, got %v", err)
			}
			msg := err.Error()
			for _, w := range tt.want {
				if !strings.Contains(msg, w) {
					t.Errorf("error %q does not contain %q", msg, w)
				}
			}
		})
	}
}

func TestLoadMissingReferencedFile(t *testing.T) {
	t.Parallel()
	p := writeCampaign(t, "apiVersion: fleet.gh/v1\nname: x\ntargets: {orgs: [a]}\nchange:\n  steps:\n    - run: ./missing.sh\npr: {title: x, body: ../nope.md}\n", nil)
	_, err := Load(p)
	if err == nil {
		t.Fatal("expected error")
	}
	for _, w := range []string{":6:12: change.steps.0.run: file \"./missing.sh\" not found", "pr.body: file \"../nope.md\""} {
		if !strings.Contains(err.Error(), w) {
			t.Errorf("%q missing %q", err, w)
		}
	}
}

func TestLoadErrors(t *testing.T) {
	t.Parallel()
	if _, err := Load(filepath.Join(t.TempDir(), "none.yaml")); err == nil {
		t.Fatal("expected error for missing file")
	}
	p := writeCampaign(t, "name: [\n", nil)
	if _, err := Load(p); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestTemplateErrors(t *testing.T) {
	t.Parallel()
	c := &Campaign{Name: "x", PR: PR{Title: "{{ .Nope }}", Body: "{{ .Repo.Nope }}"}}
	if _, err := c.RenderTitle(RepoData{}); err == nil {
		t.Fatal("expected execute error")
	}
	if _, err := c.RenderBody(RepoData{}); err == nil {
		t.Fatal("expected execute error")
	}
	c.PR.Title = "{{"
	if _, err := c.RenderTitle(RepoData{}); err == nil {
		t.Fatal("expected parse error")
	}
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "body.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	c = &Campaign{Name: "x", Path: filepath.Join(dir, "fleet.yaml"), PR: PR{Body: "body.md"}}
	// A directory is not a file reference, so the value is used inline.
	body, err := c.RenderBody(RepoData{})
	if err != nil || !strings.HasPrefix(body, "body.md") {
		t.Fatalf("body = %q, %v", body, err)
	}
}

func TestScaffoldIsValid(t *testing.T) {
	t.Parallel()
	for _, org := range []string{"", "acme"} {
		b, err := Scaffold("pin-actions", org)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(string(b), "# yaml-language-server: $schema="+SchemaURL) {
			t.Fatal("missing schema header")
		}
		c, err := Parse(b, "fleet.yaml")
		if err != nil {
			t.Fatal(err)
		}
		if c.Name != "pin-actions" {
			t.Fatalf("name = %q", c.Name)
		}
		body, err := c.RenderBody(RepoData{})
		if err != nil || !strings.Contains(body, "part of the pin-actions campaign") {
			t.Fatalf("body = %q, %v", body, err)
		}
	}
	if _, err := Scaffold("Bad Name", ""); err == nil {
		t.Fatal("expected invalid name to fail")
	}
}

func TestValidBranch(t *testing.T) {
	t.Parallel()
	for b, want := range map[string]bool{
		"fleet/x": true, "a.lock": false, "/a": false, "a/": false, "-a": false, "a.": false,
		"a//b": false, "a@{b": false, "a b": false, "a~b": false, "": false, "a\x01": false,
	} {
		if got := validBranch(b); got != want {
			t.Errorf("validBranch(%q) = %v", b, got)
		}
	}
}

func TestDurationText(t *testing.T) {
	t.Parallel()
	var d Duration
	if err := d.UnmarshalText([]byte("90s")); err != nil || d.Std() != 90*time.Second {
		t.Fatalf("d = %v, %v", d, err)
	}
	if b, _ := d.MarshalText(); string(b) != "1m30s" {
		t.Fatalf("marshal = %s", b)
	}
	if err := d.UnmarshalText([]byte("x")); err == nil {
		t.Fatal("expected error")
	}
}

func TestNormalize(t *testing.T) {
	t.Parallel()
	v := normalize(map[any]any{1: []any{map[any]any{"a": 1, "b": nil}}, "n": nil})
	m, ok := v.(map[string]any)
	if !ok || m["1"] == nil {
		t.Fatalf("normalize = %#v", v)
	}
}

func TestFieldErrorWithoutPath(t *testing.T) {
	t.Parallel()
	if got := (FieldError{File: "f", Line: 1, Column: 2, Message: "m"}).Error(); got != "f:1:2: m" {
		t.Fatal(got)
	}
}

func TestParseYAMLLine(t *testing.T) {
	t.Parallel()
	if fe := parseYAMLLine("f", "yaml: something odd"); fe.Line != 0 || fe.Message != "something odd" {
		t.Fatalf("%+v", fe)
	}
	if fe := parseYAMLLine("f", "line x: odd"); fe.Line != 0 {
		t.Fatalf("%+v", fe)
	}
}

func FuzzParse(f *testing.F) {
	seeds := []string{
		specExample,
		"apiVersion: fleet.gh/v1\nname: x\ntargets: {orgs: [a]}\nchange: {steps: [{run: x}]}\npr: {title: x}\n",
		"name: [\n",
		"",
		"a: &x [*x]\n",
		"targets: {orgs: [a]}\n",
		"1: 2\n",
		varsCampaign("vars:\n  default: {a: 1, b: [x, 2]}\n  match:\n    - when: {properties: {tier: 1}}\n      set: {a: 2}\n"),
		varsCampaign("vars:\n  default:\n    bad-name: 1\n"),
	}
	for _, s := range seeds {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		c, err := Parse(data, "fuzz.yaml")
		if err != nil {
			var ve *ValidationError
			if errors.As(err, &ve) && len(ve.Errors) == 0 {
				t.Fatal("ValidationError with no errors")
			}
			return
		}
		// Invariants of a successfully parsed campaign.
		if c.APIVersion != APIVersion || c.Name == "" || c.PR.Branch == "" || c.Change.Shell == "" ||
			c.Limits.Concurrency < 1 || c.Targets.Exclude.Archived == nil || c.Targets.Exclude.Forks == nil {
			t.Fatalf("invariant broken: %+v", c)
		}
		if len(c.Targets.Orgs) == 0 && len(c.Targets.Include.Repos) == 0 {
			t.Fatal("campaign without targets accepted")
		}
		if !validBranch(c.PR.Branch) {
			t.Fatalf("invalid branch accepted: %q", c.PR.Branch)
		}
	})
}
