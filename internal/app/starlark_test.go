package app

import (
	"context"
	"strings"
	"testing"
)

// starCampaign runs a Starlark change, then a shell step, on service repos.
const starCampaign = `apiVersion: fleet.gh/v1
name: star
targets:
  orgs: [acme]
  include:
    topics: [service]
change:
  env: {LINE: "managed by fleet"}
  steps:
    - name: edit readme
      starlark: ./change.star
    - name: notes
      run: echo "shell saw $FLEET_VAR_TEAM after $FLEET_VAR_ACTION" >> NOTES.md
pr:
  title: "chore: {{ .Vars.action }} {{ .Repo.Name }}"
  body: "Owned by {{ .Vars.team }}."
vars:
  default: {team: platform}
`

// starScript appends a line to README.md unless it is there, and says what
// it did through a variable.
const starScript = `load("./lib/text.star", "ensure_line")

def change(repo):
    added = ensure_line(repo, "README.md", env["LINE"])
    repo.set_var("action", "add line to" if added else "touch")
`

// starLib is a helper loaded by starScript.
const starLib = `def ensure_line(repo, path, line):
    lines = repo.read(path, default = "").splitlines()
    if line in lines:
        return False
    repo.append(path, line + "\n")
    return True
`

func TestStarlarkPlanAndApply(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	f := h.campaign(starCampaign, map[string]string{"change.star": starScript, "lib/text.star": starLib})
	ctx := context.Background()

	if err := h.app.Plan(ctx, Common{File: f, Repos: []string{"acme/api"}}); err != nil {
		t.Fatalf("%v\n%s", err, h.errOut.String())
	}
	if out := h.out.String(); !strings.Contains(out, "+managed by fleet") || !strings.Contains(out, "+shell saw platform after add line to") {
		t.Fatalf("plan diff:\n%s", out)
	}
	log, _ := readFile(h.app.Store.LogPath("star", "acme/api"))
	if !strings.Contains(log, "starlark: append README.md (lib/text.star:5:16)") {
		t.Fatalf("log does not name the writing line:\n%s", log)
	}

	h.reset()
	if err := h.app.Apply(ctx, ApplyOptions{Common: Common{File: f, Yes: true}}); err != nil {
		t.Fatalf("%v\n%s", err, h.errOut.String())
	}
	if len(h.gh.PRs) != 2 {
		t.Fatalf("PRs = %d", len(h.gh.PRs))
	}
	for _, p := range h.gh.PRs {
		if !strings.HasPrefix(p.Title, "chore: add line to ") || !strings.HasPrefix(p.Body, "Owned by platform.") {
			t.Fatalf("PR = %q / %q", p.Title, p.Body)
		}
	}
	if got := h.remotes["acme/api"].Show("fleet/star", "README.md"); got != "# api\nmanaged by fleet" {
		t.Fatalf("README = %q", got)
	}
}

func TestStarlarkOnly(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	camp := strings.Replace(starCampaign, "    - name: notes\n      run: echo \"shell saw $FLEET_VAR_TEAM after $FLEET_VAR_ACTION\" >> NOTES.md\n", "", 1)
	f := h.campaign(camp, map[string]string{"change.star": starScript, "lib/text.star": starLib})
	if err := h.app.Plan(context.Background(), Common{File: f, Repos: []string{"acme/web"}}); err != nil {
		t.Fatalf("%v\n%s", err, h.errOut.String())
	}
	if out := h.out.String(); !strings.Contains(out, "+managed by fleet") || strings.Contains(out, "NOTES.md") {
		t.Fatalf("plan diff:\n%s", out)
	}
}

func TestStarlarkErrorsSurfaceEarly(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	f := h.campaign(starCampaign, map[string]string{"change.star": "def change(repo):\n    undefined_helper(repo)\n", "lib/text.star": starLib})
	ctx := context.Background()
	for name, run := range map[string]func() error{
		"targets": func() error { return h.app.Targets(ctx, Common{File: f}) },
		"plan":    func() error { return h.app.Plan(ctx, Common{File: f}) },
		"apply":   func() error { return h.app.Apply(ctx, ApplyOptions{Common: Common{File: f, Yes: true}}) },
	} {
		err := run()
		if err == nil || !strings.Contains(err.Error(), "change.steps[0] (edit readme): change.star:2:5: undefined: undefined_helper") {
			t.Errorf("%s: err = %v", name, err)
		}
	}
	if h.gh.CallCount("ListOwnerRepos") != 0 {
		t.Fatal("targets were resolved before the script was checked")
	}

	// A runtime failure fails only that repo, with the Starlark traceback.
	h.campaign(starCampaign, map[string]string{"change.star": "def change(repo):\n    repo.set_var('action', 'x')\n    if repo.name == 'web':\n        fail('web is special')\n", "lib/text.star": starLib})
	h.reset()
	err := h.app.Plan(ctx, Common{File: f})
	if err == nil || !strings.Contains(h.errOut.String(), "failed   acme/web step edit readme: change.star:4:13: fail: web is special") || !strings.Contains(h.errOut.String(), "1 failed") {
		t.Fatalf("plan: %v\n%s", err, h.errOut.String())
	}
}
