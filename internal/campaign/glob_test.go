package campaign

import (
	"strings"
	"testing"
)

func TestMatchRepo(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		pattern, repo string
		want          bool
	}{
		{"acme/api", "acme/api", true},
		{"ACME/Api", "acme/api", true},
		{"acme/api", "acme/api-v2", false},
		{"acme/svc-*", "acme/svc-billing", true},
		{"acme/svc-*", "acme/svc-", true},
		{"acme/svc-*", "other/svc-billing", false},
		{"*/legacy-*", "other/legacy-api", true},
		{"*/legacy-*", "other/new-legacy-api", false},
		{"acme/api-v?", "acme/api-v2", true},
		{"acme/api-v?", "acme/api-v10", false},
		{"acme/[a-c]*", "acme/billing", true},
		{"acme/[^a-c]*", "acme/billing", false},
		{"service-*", "acme/service-mesh", true},
		{"service-*", "acme/my-service-mesh", false},
		{"*", "acme/anything", true},
		{"acme/*", "acme", false},
		{"acme/[", "acme/[", false},
	} {
		if got := MatchRepo(tt.pattern, tt.repo); got != tt.want {
			t.Errorf("MatchRepo(%q, %q) = %v, want %v", tt.pattern, tt.repo, got, tt.want)
		}
	}
	if IsPattern("acme/api") || !IsPattern("acme/a*") || !IsPattern("acme/a?") || !IsPattern("acme/[ab]") {
		t.Error("IsPattern")
	}
	if got := MatchAny([]string{"acme/web", "acme/a*", "acme/api"}, "acme/api"); got != "acme/a*" {
		t.Errorf("MatchAny = %q, want the first match", got)
	}
	if got := MatchAny(nil, "acme/api"); got != "" {
		t.Errorf("MatchAny(nil) = %q", got)
	}
}

func TestLoadPatterns(t *testing.T) {
	t.Parallel()
	good := writeCampaign(t, `apiVersion: fleet.gh/v1
name: globs
targets:
  orgs: [acme]
  include:
    names: [service-*, "acme/api-v[0-9]"]
    repos: ["*/tool-*", other/cli-*]
  exclude:
    repos: ["*/legacy-*"]
vars:
  match:
    - when: {repos: [acme/payments-*]}
      set: {team: payments}
change:
  steps: [{run: "true"}]
pr:
  title: x
limits:
  waves:
    - repos: [acme/canary-*]
`, nil)
	c, err := Load(good)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(c.Targets.Include.Names, ",") != "service-*,acme/api-v[0-9]" || c.Limits.Waves[0].Repos[0] != "acme/canary-*" {
		t.Fatalf("campaign = %+v", c.Targets)
	}

	for name, tt := range map[string]struct{ yaml, want string }{
		"bad pattern": {`apiVersion: fleet.gh/v1
name: globs
targets:
  orgs: [acme]
  exclude:
    repos: ["acme/[ab"]
change:
  steps: [{run: "true"}]
pr:
  title: x
`, `fleet.yaml:6:13: targets.exclude.repos.0: "acme/[ab" is not a valid glob pattern`},
		"bad pattern in a wave": {`apiVersion: fleet.gh/v1
name: globs
targets:
  orgs: [acme]
change:
  steps: [{run: "true"}]
pr:
  title: x
vars:
  match:
    - when: {repos: ["acme/[x"]}
      set: {a: 1}
limits:
  waves:
    - repos: ["acme/[y"]
`, "limits.waves.0.repos.0"},
		"owner pattern without orgs": {`apiVersion: fleet.gh/v1
name: globs
targets:
  include:
    repos: ["*/tool-*"]
change:
  steps: [{run: "true"}]
pr:
  title: x
`, `targets.include.repos.0: "*/tool-*" has a pattern for the owner, which needs orgs to match against`},
		"slash in a name pattern is owner/name only": {`apiVersion: fleet.gh/v1
name: globs
targets:
  orgs: [acme]
  include:
    names: [a/b/c]
change:
  steps: [{run: "true"}]
pr:
  title: x
`, "targets.include.names.0"},
	} {
		_, err := Load(writeCampaign(t, tt.yaml, nil))
		if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: err = %v, want %q", name, err, tt.want)
		}
	}
}
