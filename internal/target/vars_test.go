package target

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/rossedman/gh-fleet/internal/campaign"
	"github.com/rossedman/gh-fleet/internal/ghapi"
	"github.com/rossedman/gh-fleet/internal/ghapi/fake"
)

func TestEvaluateProperties(t *testing.T) {
	t.Parallel()
	r := ghapi.Repo{Owner: "acme", Name: "api", Properties: map[string]string{"Tier": "1", "langs": "go,rust", "pci": "true"}}
	tests := []struct {
		name    string
		tg      campaign.Targets
		match   bool
		reasons []string
		exclude string
	}{
		{"single", campaign.Targets{Include: campaign.Include{Properties: map[string]campaign.StringList{"tier": {"1", "2"}}}}, true, []string{"property:tier=1"}, ""},
		{"multi-select", campaign.Targets{Include: campaign.Include{Properties: map[string]campaign.StringList{"langs": {"RUST"}}}}, true, []string{"property:langs=rust"}, ""},
		{"and across keys", campaign.Targets{Include: campaign.Include{Properties: map[string]campaign.StringList{"tier": {"1"}, "langs": {"java"}}}}, false, nil, ""},
		{"missing property", campaign.Targets{Include: campaign.Include{Properties: map[string]campaign.StringList{"team": {"x"}}}}, false, nil, ""},
		{"excluded", campaign.Targets{Exclude: campaign.Exclude{Properties: map[string]campaign.StringList{"pci": {"true"}}}}, false, nil, "property:pci=true"},
		{"exclude multi-select", campaign.Targets{Exclude: campaign.Exclude{Properties: map[string]campaign.StringList{"langs": {"go"}}}}, false, nil, "property:langs=go"},
		{"exclude no hit", campaign.Targets{Exclude: campaign.Exclude{Properties: map[string]campaign.StringList{"pci": {"false"}}}}, true, []string{"org:acme"}, ""},
	}
	for _, tt := range tests {
		d := Evaluate(tt.tg, r, Facts{})
		if d.Match != tt.match || !slices.Equal(d.Reasons, tt.reasons) || d.Excluded != tt.exclude {
			t.Errorf("%s: %+v", tt.name, d)
		}
	}
}

func TestVarsPrecedence(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "v.csv"), []byte("repo,team\nacme/api,from-file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	yaml := "apiVersion: fleet.gh/v1\nname: v\ntargets: {orgs: [acme]}\nchange: {steps: [{run: x}]}\npr: {title: x}\n" +
		"vars:\n  default: {team: default, node: 22, list: [a]}\n  file: v.csv\n  match:\n" +
		"    - when: {topics: [web]}\n      set: {team: web, node: 20}\n" +
		"    - when: {languages: [Go], properties: {tier: 1}}\n      set: {node: 18}\n" +
		"    - when: {repos: [acme/other]}\n      set: {team: other}\n"
	p := filepath.Join(dir, "fleet.yaml")
	if err := os.WriteFile(p, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := campaign.Load(p)
	if err != nil {
		t.Fatal(err)
	}
	api := ghapi.Repo{Owner: "acme", Name: "api", Topics: []string{"web"}, Language: "Go", Properties: map[string]string{"tier": "1"}}
	v := Vars(&c.Vars, api)
	if v["team"] != "from-file" || v["node"] != 18 {
		t.Fatalf("api vars = %v (file beats rules, later rules beat earlier ones)", v)
	}
	web := ghapi.Repo{Owner: "acme", Name: "web", Topics: []string{"web"}, Language: "TypeScript"}
	if v := Vars(&c.Vars, web); v["team"] != "web" || v["node"] != 20 {
		t.Fatalf("web vars = %v", v)
	}
	plain := Vars(&c.Vars, ghapi.Repo{Owner: "acme", Name: "lib"})
	plain["list"].([]any)[0] = "changed"
	if again := Vars(&c.Vars, ghapi.Repo{Owner: "acme", Name: "lib"}); again["list"].([]any)[0] != "a" {
		t.Fatal("repos share a list value")
	}
	if !When(campaign.When{Repos: []string{"acme/w*"}}, web) || When(campaign.When{Repos: []string{"*/api"}}, web) {
		t.Error("When with repo patterns")
	}
	if !When(campaign.When{}, web) || When(campaign.When{Languages: []string{"Go"}}, ghapi.Repo{}) {
		t.Fatal("When")
	}
	if m, ok := cloneValue(map[string]any{"a": 1}).(map[string]any); !ok || m["a"] != 1 {
		t.Fatal("cloneValue map")
	}

	res := &Result{Targets: []Target{{Repo: api}, {Repo: web}}}
	res.SetVars(&c.Vars)
	if res.Targets[1].Vars["team"] != "web" {
		t.Fatalf("SetVars = %+v", res.Targets)
	}
}

func TestResolveFetchesProperties(t *testing.T) {
	t.Parallel()
	f := fake.New()
	f.AddRepo(ghapi.Repo{Owner: "acme", Name: "api", CanPush: true, Properties: map[string]string{"tier": "1"}})
	f.AddRepo(ghapi.Repo{Owner: "acme", Name: "web", CanPush: true, Properties: map[string]string{"tier": "3"}})
	f.AddRepo(ghapi.Repo{Owner: "other", Name: "x", CanPush: true, Properties: map[string]string{"tier": "1"}})
	rs := &Resolver{Client: f, Cache: &Cache{Dir: t.TempDir()}}
	tg := campaign.Targets{Orgs: []string{"acme"}, Include: campaign.Include{
		Repos:      []string{"other/x"},
		Properties: map[string]campaign.StringList{"tier": {"1"}},
	}}
	res, err := rs.Resolve(context.Background(), tg)
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Names(); !slices.Equal(got, []string{"acme/api", "other/x"}) {
		t.Fatalf("targets = %v", got)
	}
	if res.Targets[1].Repo.Properties["tier"] != "1" {
		t.Fatal("include.repos repo has no properties")
	}
	// The cached listing keeps properties, so a second run needs no calls.
	if _, err := rs.Resolve(context.Background(), campaign.Targets{Orgs: []string{"acme"}, Include: campaign.Include{Properties: map[string]campaign.StringList{"tier": {"1"}}}}); err != nil {
		t.Fatal(err)
	}
	if f.CallCount("OwnerProperties") != 1 || f.CallCount("ListOwnerRepos") != 1 {
		t.Fatalf("calls = %v", f.Calls)
	}

	for _, method := range []string{"OwnerProperties", "RepoProperties"} {
		f := fake.New()
		f.AddRepo(ghapi.Repo{Owner: "acme", Name: "api"})
		f.AddRepo(ghapi.Repo{Owner: "other", Name: "x"})
		f.Errors[method] = errors.New("boom")
		// A filter, so the org is listed as well as other/x fetched.
		tg := campaign.Targets{Orgs: []string{"acme"}, Include: campaign.Include{Names: []string{"*"}, Repos: []string{"other/x"}}}
		if _, err := (&Resolver{Client: f, RequireProperties: true}).Resolve(context.Background(), tg); err == nil {
			t.Errorf("%s error not returned", method)
		}
		// When the campaign doesn't use properties, unreadable ones are none.
		res, err := (&Resolver{Client: f}).Resolve(context.Background(), tg)
		if err != nil || len(res.Targets)+len(res.Skipped) != 2 {
			t.Errorf("%s: optional properties failed the run: %v", method, err)
		}
	}
}

func TestCacheVersion(t *testing.T) {
	t.Parallel()
	c := &Cache{Dir: t.TempDir()}
	if err := os.WriteFile(c.path("acme"), []byte(`{"fetched":"2999-01-01T00:00:00Z","repos":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, ok := c.Get("acme"); ok {
		t.Fatal("entry from an older version returned")
	}
}
