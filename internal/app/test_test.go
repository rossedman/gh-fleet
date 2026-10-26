package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// testCampaign sets a version in VERSION from env, records the old one,
// and lists tracked files with git in a shell step.
const testCampaign = `apiVersion: fleet.gh/v1
name: pin-version
targets:
  orgs: [acme]
change:
  steps:
    - name: pin
      starlark: ./change.star
    - name: list
      run: echo "files=$(git ls-files | tr '\n' ' ')" >> "$FLEET_OUTPUT/vars"
  env: {VERSION: "2.0"}
vars:
  default: {team: platform}
pr:
  title: "chore: version {{ .Vars.version }} for {{ .Repo.Name }} ({{ .Vars.team }})"
  body: "Was {{ .Vars.old }}."
`

const testScript = `def change(repo):
    if repo.vars.get("boom"):
        fail("boom")
    old = repo.read("VERSION", default = "none").strip()
    repo.set_var("old", old)
    repo.set_var("version", env["VERSION"])
    if repo.exists("VERSION"):
        repo.write("VERSION", env["VERSION"] + "\n")
    if repo.exists("grow"):
        repo.append("grow", "x\n")
`

// testCases writes case files under tests/ in the harness's campaign dir.
func (h *harness) cases(files map[string]string) {
	h.t.Helper()
	for name, content := range files {
		p := filepath.Join(h.dir, TestDir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			h.t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			h.t.Fatal(err)
		}
	}
}

func (h *harness) test(o TestOptions) error {
	h.t.Helper()
	h.reset()
	o.File = filepath.Join(h.dir, "fleet.yaml")
	return h.app.Test(context.Background(), o)
}

func TestTestCases(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.campaign(testCampaign, map[string]string{"change.star": testScript})
	h.cases(map[string]string{
		"bump/before/VERSION":  "1.0\n",
		"bump/after/VERSION":   "2.0\n",
		"bump/case.yaml":       "repo: acme/api\nwantVars:\n  old: \"1.0\"\n  files: \"VERSION \"\nwantTitle: \"chore: version 2.0 for api (platform)\"\n",
		"noop/before/README":   "hi\n",
		"noop/case.yaml":       "vars: {team: web}\n",
		"env/before/VERSION":   "1.0\n",
		"env/after/VERSION":    "3.0\n",
		"env/case.yaml":        "env: {VERSION: \"3.0\"}\n",
		"error/before/x":       "",
		"error/case.yaml":      "vars: {boom: true}\nwantError: boom\n",
		"nested/before/a/b/c":  "deep\n",
		"nested/case.yaml":     "language: Go\ntopics: [service]\nproperties: {tier: \"1\"}\ndefaultBranch: trunk\n",
		"skip-me/before/x":     "",
		"skip-me/case.yaml":    "wantError: never\n",
		"not-a-case.txt":       "",
		"nested/before/.git/x": "ignored",
	})
	if err := h.test(TestOptions{Run: "^(bump|noop|env|error|nested)$", Verbose: true}); err != nil {
		t.Fatalf("%v\n%s%s", err, h.out, h.errOut)
	}
	for _, want := range []string{"ok   bump", "ok   env", "ok   error", "ok   nested", "ok   noop", "title: chore: version 2.0 for api (platform)", "## step pin"} {
		if !strings.Contains(h.out.String(), want) {
			t.Errorf("output missing %q:\n%s", want, h.out)
		}
	}
	if strings.Contains(h.out.String(), "skip-me") || h.errOut.String() != "5 passed, 0 failed\n" {
		t.Errorf("summary = %q\n%s", h.errOut, h.out)
	}

	// JSON output.
	if err := h.test(TestOptions{Common: Common{JSON: true}, Run: "bump"}); err != nil {
		t.Fatal(err)
	}
	var res []TestResult
	if err := json.Unmarshal(h.out.Bytes(), &res); err != nil || len(res) != 1 || res[0].Status != "pass" || res[0].Title == "" || res[0].Vars["old"] != "1.0" {
		t.Errorf("json = %s (%v)", h.out, err)
	}
}

func TestTestFailures(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.campaign(testCampaign, map[string]string{"change.star": testScript})
	h.cases(map[string]string{
		"wrong-after/before/VERSION": "1.0\n",
		"wrong-after/after/VERSION":  "9.9\n",
		"no-after/before/VERSION":    "1.0\n",
		"extra-file/before/VERSION":  "1.0\n",
		"extra-file/after/VERSION":   "2.0\n",
		"extra-file/after/NEW":       "x\n",
		"bad-vars/before/VERSION":    "1.0\n",
		"bad-vars/after/VERSION":     "2.0\n",
		"bad-vars/case.yaml":         "wantVars: {old: \"0.1\", missing: 1}\nwantTitle: nope\n",
		"no-title/before/x":          "",
		"no-title/case.yaml":         "wantTitle: something\n",
		"not-idempotent/before/grow": "",
		"not-idempotent/after/grow":  "x\n",
		"wrong-error/before/x":       "",
		"wrong-error/case.yaml":      "vars: {boom: true}\nwantError: bang\n",
		"no-error/before/x":          "",
		"no-error/case.yaml":         "wantError: boom\n",
		"unexpected-error/before/x":  "",
		"unexpected-error/case.yaml": "vars: {boom: true}\n",
		"bad-yaml/before/x":          "",
		"bad-yaml/case.yaml":         "nope: 1\n",
		"bad-repo/before/x":          "",
		"bad-repo/case.yaml":         "repo: just-a-name\n",
		"no-before/after/x":          "",
	})
	err := h.test(TestOptions{})
	var fe *FailedError
	if !errors.As(err, &fe) || fe.N != 12 || err.Error() != "12 test cases failed" {
		t.Fatalf("err = %v\n%s", err, h.out)
	}
	for _, want := range []string{
		"FAIL wrong-after\n     files differ from after/",
		"-9.9\n+2.0",
		"FAIL no-after\n     the change edited files, but the case has no after/",
		"--- want/NEW\n+++ /dev/null",
		`var missing = "", want "1"`,
		`var old = "1.0", want "0.1"`,
		`PR title = "chore: version 2.0 for test (platform)", want "nope"`,
		`the change edited nothing, so there is no PR title; want "something"`,
		"a second run changed files again",
		`the error does not contain "bang": step pin: change.star:3:13: fail: boom`,
		`the change succeeded; want an error containing "boom"`,
		"FAIL unexpected-error\n     step pin: change.star:3:13: fail: boom\n     Traceback",
		"field nope not found",
		`repo "just-a-name" is not owner/name`,
		"FAIL no-before\n     no before/ directory",
	} {
		if !strings.Contains(h.out.String(), want) {
			t.Errorf("output missing %q:\n%s", want, h.out)
		}
	}
}

func TestTestUpdate(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.campaign(testCampaign, map[string]string{"change.star": testScript})
	h.cases(map[string]string{
		"bump/before/VERSION":     "1.0\n",
		"bump/after/VERSION":      "old\n",
		"bump/after/STALE":        "x\n",
		"noop/before/README":      "hi\n",
		"noop/after/README":       "stale\n",
		"bad-vars/before/VERSION": "1.0\n",
		"bad-vars/case.yaml":      "wantVars: {old: nope}\n",
	})
	err := h.test(TestOptions{Update: true})
	if err == nil || !strings.Contains(h.errOut.String(), "0 passed, 1 failed, 2 updated") {
		t.Fatalf("err = %v, summary %q\n%s", err, h.errOut, h.out)
	}
	if b, err := os.ReadFile(filepath.Join(h.dir, "tests/bump/after/VERSION")); err != nil || string(b) != "2.0\n" {
		t.Errorf("after/VERSION = %q, %v", b, err)
	}
	for _, gone := range []string{"tests/bump/after/STALE", "tests/noop/after"} {
		if _, err := os.Stat(filepath.Join(h.dir, gone)); err == nil {
			t.Errorf("%s still exists", gone)
		}
	}
	if err := h.test(TestOptions{Run: "bump|noop"}); err != nil {
		t.Errorf("after --update: %v\n%s", err, h.out)
	}
}

func TestTestDir(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.campaign(testCampaign, map[string]string{"change.star": testScript})
	repo := filepath.Join(t.TempDir(), "billing")
	for name, content := range map[string]string{"VERSION": "1.0\n", ".git/HEAD": "ignored", "sub/a": "a\n"} {
		_ = os.MkdirAll(filepath.Dir(filepath.Join(repo, name)), 0o755)
		_ = os.WriteFile(filepath.Join(repo, name), []byte(content), 0o644)
	}
	if err := h.test(TestOptions{Dir: repo, Verbose: true}); err != nil {
		t.Fatalf("%v\n%s%s", err, h.out, h.errOut)
	}
	for _, want := range []string{"── acme/billing", "-1.0\n+2.0", "var old=1.0", "title chore: version 2.0 for billing (platform)"} {
		if !strings.Contains(h.out.String(), want) {
			t.Errorf("output missing %q:\n%s", want, h.out)
		}
	}
	if b, _ := os.ReadFile(filepath.Join(repo, "VERSION")); string(b) != "1.0\n" {
		t.Errorf("--dir changed the directory itself: %q", b)
	}

	if err := h.test(TestOptions{Common: Common{JSON: true, Repos: []string{"acme/api"}}, Dir: repo}); err != nil {
		t.Fatal(err)
	}
	var r TestResult
	if err := json.Unmarshal(h.out.Bytes(), &r); err != nil || r.Name != "acme/api" || !strings.Contains(r.Diff, "+2.0") || r.Title != "chore: version 2.0 for api (platform)" {
		t.Errorf("json = %s (%v)", h.out, err)
	}

	empty := t.TempDir()
	if err := h.test(TestOptions{Dir: empty}); err != nil || !strings.Contains(h.errOut.String(), "no changes") {
		t.Errorf("empty dir: %v %q", err, h.errOut)
	}
	for o, want := range map[*TestOptions]string{
		{Dir: repo, Update: true}:                         "--dir can't be combined with --update or --run",
		{Dir: filepath.Join(repo, "VERSION")}:             "is not a directory",
		{Dir: filepath.Join(repo, "missing")}:             noSuchFile,
		{Run: "("}:                                        "--run:",
		{Run: "nothing-matches"}:                          "no test cases in",
		{Common: Common{Repos: []string{"x"}}, Dir: repo}: `repo "x" is not owner/name`,
	} {
		if err := h.test(*o); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%+v: err = %v, want %q", *o, err, want)
		}
	}
	h.cases(map[string]string{"fails/before/x": ""})
	boom := filepath.Join(h.dir, "fleet.yaml")
	_ = os.WriteFile(boom, []byte(strings.Replace(testCampaign, "vars:\n  default: {team: platform}", "vars:\n  default: {team: platform, boom: true}", 1)), 0o644)
	if err := h.test(TestOptions{Dir: repo}); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("failing change: %v", err)
	}
	if err := h.test(TestOptions{Common: Common{JSON: true}, Dir: repo}); err == nil || !strings.Contains(h.out.String(), `"status": "fail"`) {
		t.Errorf("failing change json: %v %s", err, h.out)
	}
}
