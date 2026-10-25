package app

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/rossedman/gh-fleet/internal/target"
)

// varsCampaign targets every service repo and uses variables everywhere:
// the script environment, the title, body, labels and reviewers.
const varsCampaign = `apiVersion: fleet.gh/v1
name: owners
targets:
  orgs: [acme]
  include:
    topics: [service]
change:
  steps:
    - run: |
        echo "team=$FLEET_VAR_TEAM tier=${FLEET_PROP_TIER:-none} tags=$FLEET_VAR_TAGS" >> README.md
        echo "note=from script" > "$FLEET_OUTPUT/vars"
pr:
  title: "chore: owners for {{ .Repo.Name }} ({{ .Vars.team }})"
  body: "Ping {{ .Vars.oncall }}. {{ .Vars.note }}."
  labels: ["team:{{ .Vars.team }}", "{{ join .Vars.tags \",\" }}"]
  reviewers: ["acme/{{ .Vars.team }}"]
vars:
  default: {team: platform, oncall: "#platform", tags: [fleet, owners]}
  match:
    - when: {languages: [TypeScript]}
      set: {team: web}
  file: ./owners.csv
`

func TestVarsEndToEnd(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	api := h.gh.Repos["acme/api"]
	api.Properties = map[string]string{"tier": "1"}
	h.gh.Repos["acme/api"] = api
	f := h.campaign(varsCampaign, map[string]string{"owners.csv": "repo,oncall\nacme/api,#payments\nacme/gone,#x\n"})
	ctx := context.Background()

	if err := h.app.Targets(ctx, Common{File: f}); err != nil {
		t.Fatal(err)
	}
	out := h.out.String()
	for _, want := range []string{"VARS", "oncall=#payments tags=fleet,owners team=platform", "oncall=#platform tags=fleet,owners team=web", "warning: ./owners.csv names 1 repo that is not a target: acme/gone"} {
		if !strings.Contains(out, want) {
			t.Errorf("targets missing %q:\n%s", want, out)
		}
	}
	h.reset()
	if err := h.app.Targets(ctx, Common{File: f, JSON: true}); err != nil {
		t.Fatal(err)
	}
	var res target.Result
	if err := json.Unmarshal(h.out.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if res.Targets[0].Vars["oncall"] != "#payments" || res.Targets[0].Repo.Properties["tier"] != "1" {
		t.Fatalf("json = %+v", res.Targets[0])
	}

	if err := h.app.Apply(ctx, ApplyOptions{Common: Common{File: f, Yes: true}}); err != nil {
		t.Fatalf("%v\n%s", err, h.errOut.String())
	}
	byRepo := map[string]int{}
	for i, p := range h.gh.PRs {
		byRepo[p.Repo] = i
	}
	p := h.gh.PRs[byRepo["acme/api"]]
	if p.Title != "chore: owners for api (platform)" || !strings.HasPrefix(p.Body, "Ping #payments. from script.") {
		t.Fatalf("api PR = %q / %q", p.Title, p.Body)
	}
	if strings.Join(p.Labels, ",") != "team:platform,fleet,owners" {
		t.Fatalf("labels = %v", p.Labels)
	}
	if got := h.gh.Reviewers["acme/api#1"]; len(got) != 1 || got[0] != "acme/platform" {
		t.Fatalf("reviewers = %v", got)
	}
	if web := h.gh.PRs[byRepo["acme/web"]]; web.Title != "chore: owners for web (web)" {
		t.Fatalf("web PR = %q", web.Title)
	}
	if got := h.remotes["acme/api"].Show("fleet/owners", "README.md"); !strings.Contains(got, "team=platform tier=1 tags=fleet,owners") {
		t.Fatalf("script env: %q", got)
	}
	if got := h.remotes["acme/web"].Show("fleet/owners", "README.md"); !strings.Contains(got, "team=web tier=none") {
		t.Fatalf("script env: %q", got)
	}
}

func TestMissingVariableFailsBeforePush(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	f := h.campaign(strings.Replace(varsCampaign, "Ping {{ .Vars.oncall }}", "Ping {{ .Vars.pager }}", 1), map[string]string{"owners.csv": "repo,oncall\n"})
	ctx := context.Background()
	err := h.app.Plan(ctx, Common{File: f})
	if err == nil || !strings.Contains(h.errOut.String(), `missing variable "pager" for acme/api`) {
		t.Fatalf("plan: %v\n%s", err, h.errOut.String())
	}
	h.reset()
	if err := h.app.Apply(ctx, ApplyOptions{Common: Common{File: f, Yes: true}}); err == nil {
		t.Fatal("apply succeeded with a missing variable")
	}
	if len(h.gh.PRs) != 0 || h.remotes["acme/api"].Branch("fleet/owners") != "" {
		t.Fatal("pushed or opened a PR despite the template error")
	}
}

func TestExecGetsCampaignVars(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	f := h.campaign(varsCampaign, map[string]string{"owners.csv": "repo,oncall\nacme/api,#payments\n"})
	if err := h.app.Exec(context.Background(), ExecOptions{Campaign: f, Argv: []string{"sh", "-c", "echo $FLEET_VAR_ONCALL"}}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.out.String(), "#payments") || !strings.Contains(h.out.String(), "#platform") {
		t.Fatalf("exec output:\n%s", h.out.String())
	}
}

func TestOptionalPropertiesDoNotFailTargets(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.gh.Errors["OwnerProperties"] = errors.New("403 forbidden")
	f := h.campaign(basic, files)
	if err := h.app.Targets(context.Background(), Common{File: f}); err != nil {
		t.Fatalf("a campaign without properties failed: %v", err)
	}
	f = h.campaign(strings.Replace(basic, "topics: [service]", "properties: {tier: 1}", 1), files)
	if err := h.app.Targets(context.Background(), Common{File: f}); err == nil {
		t.Fatal("a campaign filtering on properties ignored the error")
	}
}
