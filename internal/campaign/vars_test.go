package campaign

import (
	"os"
	"strings"
	"testing"
)

// varsCampaign is a valid campaign with a vars block appended.
func varsCampaign(vars string) string {
	return "apiVersion: fleet.gh/v1\nname: v\ntargets:\n  orgs: [acme]\n  include:\n    properties: {tier: [1, 2], team: payments}\n  exclude:\n    properties: {pci: true}\nchange: {run: x}\npr: {title: x}\n" + vars
}

func TestParseVarsAndProperties(t *testing.T) {
	t.Parallel()
	c, err := Parse([]byte(varsCampaign(`vars:
  default: {node: 22, reviewers: [a, b], ok: true, ratio: 1.5}
  match:
    - when: {topics: [legacy], properties: {tier: 3}}
      set: {node: "20"}
    - when: {}
      set: {all: yes}
`)), "v.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if got := c.Targets.Include.Properties["tier"]; len(got) != 2 || got[0] != "1" || c.Targets.Include.Properties["team"][0] != "payments" {
		t.Fatalf("include.properties = %v", c.Targets.Include.Properties)
	}
	if c.Targets.Exclude.Properties["pci"][0] != "true" {
		t.Fatalf("exclude.properties = %v", c.Targets.Exclude.Properties)
	}
	if c.Vars.Default["node"] != 22 || len(c.Vars.Match) != 2 || c.Vars.Match[0].When.Properties["tier"][0] != "3" {
		t.Fatalf("vars = %+v", c.Vars)
	}
}

func TestVarsValidationErrors(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct{ vars, want string }{
		{"vars:\n  default:\n    bad-name: 1\n", "v.yaml:13:5: vars.default.bad-name: invalid name"},
		{"vars:\n  default:\n    obj: {a: 1}\n", "v.yaml:13:10: vars.default.obj: got object, want boolean, number, string, a list"},
		{"vars:\n  match:\n    - set: {a: 1}\n", "vars.match.0: missing property 'when'"},
		{"vars:\n  bogus: 1\n", "vars.bogus: unknown field"},
	} {
		_, err := Parse([]byte(varsCampaign(tt.vars)), "v.yaml")
		if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("vars %q: err = %v, want %q", tt.vars, err, tt.want)
		}
	}
	bad := "apiVersion: fleet.gh/v1\nname: v\ntargets:\n  orgs: [a]\n  include:\n    properties: {tier: {x: 1}}\nchange: {run: x}\npr: {title: x}\n"
	if _, err := Parse([]byte(bad), "v.yaml"); err == nil {
		t.Fatal("object property filter accepted")
	}
}

func TestStringList(t *testing.T) {
	t.Parallel()
	c, err := Parse([]byte("apiVersion: fleet.gh/v1\nname: v\ntargets:\n  orgs: [a]\n  include:\n    properties: {tier: 1}\nchange: {run: x}\npr: {title: x}\n"), "v.yaml")
	if err != nil || c.Targets.Include.Properties["tier"][0] != "1" {
		t.Fatalf("scalar filter: %v %v", c, err)
	}
}

func TestVarsFile(t *testing.T) {
	t.Parallel()
	for name, content := range map[string]string{
		"owners.csv":  "repo, team, oncall\nacme/API,payments,#pay\nacme/web,web,\n",
		"owners.json": `{"acme/api": {"team": "payments", "oncall": "#pay"}, "acme/web": {"team": "web", "n": 2, "l": [1, "a"]}}`,
		"owners.yaml": "acme/api: {team: payments, oncall: '#pay'}\nacme/web: {team: web}\n",
		"owners.yml":  "acme/api: {team: payments, oncall: '#pay'}\nacme/web: {team: web}\n",
	} {
		p := writeCampaign(t, varsCampaign("vars:\n  file: ./"+name+"\n"), map[string]string{name: content})
		c, err := Load(p)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		api := c.Vars.Row("acme/api")
		if api["team"] != "payments" || api["oncall"] != "#pay" {
			t.Errorf("%s: acme/api = %v", name, api)
		}
		if web := c.Vars.Row("ACME/web"); web["team"] != "web" || web["oncall"] != nil {
			t.Errorf("%s: acme/web = %v (empty cells must stay unset)", name, web)
		}
		if got := c.Vars.RowRepos(); len(got) != 2 || got[0] != "acme/api" {
			t.Errorf("%s: RowRepos = %v", name, got)
		}
	}
}

func TestVarsFileErrors(t *testing.T) {
	t.Parallel()
	for name, tt := range map[string]struct{ file, content, want string }{
		"missing":     {"none.csv", "", "vars.file: none.csv:"},
		"type":        {"v.txt", "x", `unsupported file type ".txt"`},
		"no repo col": {"v.csv", "name,team\nacme/a,x\n", "line 1: no `repo` column"},
		"bad column":  {"v.csv", "repo,bad-name\nacme/a,x\n", `column "bad-name" is not a valid variable name`},
		"bad repo":    {"v.csv", "repo,team\nnot-a-repo,x\n", `line 2: "not-a-repo" is not an owner/name repo`},
		"duplicate":   {"v.csv", "repo,team\nacme/a,x\nACME/a,y\n", "line 3: ACME/a appears twice"},
		"ragged":      {"v.csv", "repo,team\nacme/a,x,extra\n", "wrong number of fields"},
		"empty":       {"v.csv", "", "read header"},
		"json syntax": {"v.json", "{", "unexpected end"},
		"json repo":   {"v.json", `{"x": {"a": 1}}`, `"x" is not an owner/name repo`},
		"json name":   {"v.json", `{"a/b": {"bad-name": 1}}`, `"bad-name" is not a valid variable name`},
		"json value":  {"v.json", `{"a/b": {"x": {"y": 1}}}`, "x must be a string, number, boolean or list"},
		"json list":   {"v.json", `{"a/b": {"x": [{"y": 1}]}}`, "x must be a string, number, boolean or list"},
	} {
		files := map[string]string{}
		if tt.content != "" || name != "missing" {
			files[tt.file] = tt.content
		}
		p := writeCampaign(t, varsCampaign("vars:\n  file: "+tt.file+"\n"), files)
		_, err := Load(p)
		if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: err = %v, want %q", name, err, tt.want)
		}
	}
}

func TestTemplatesWithVars(t *testing.T) {
	t.Parallel()
	c := &Campaign{Name: "x", PR: PR{
		Title: "Node {{ .Vars.node }} for {{ .Repo.Name }} ({{ .Repo.Properties.tier }})",
		Body:  "Ping {{ .Vars.oncall }}; reviewers {{ join .Vars.reviewers \", \" }}; {{ upper .Vars.team }} {{ lower \"X\" }}",
	}}
	repo := RepoData{Name: "api", FullName: "acme/api", Properties: map[string]string{"tier": "1"},
		Vars: map[string]any{"node": 22, "oncall": "#pay", "reviewers": []any{"a", "b"}, "team": "pay"}}
	title, err := c.RenderTitle(repo)
	if err != nil || title != "Node 22 for api (1)" {
		t.Fatalf("title = %q, %v", title, err)
	}
	body, err := c.RenderBody(repo)
	if err != nil || !strings.HasPrefix(body, "Ping #pay; reviewers a, b; PAY x") {
		t.Fatalf("body = %q, %v", body, err)
	}

	c.PR.Title = "{{ .Vars.missing }}"
	if _, err := c.RenderTitle(repo); err == nil || err.Error() != `pr.title template: missing variable "missing" for acme/api` {
		t.Fatalf("err = %v", err)
	}
	c.PR.Title = "{{ .Repo.Properties.nope }}"
	if _, err := c.RenderTitle(repo); err == nil || !strings.Contains(err.Error(), `missing custom property "nope" for acme/api`) {
		t.Fatalf("err = %v", err)
	}
	c.PR.Title = "{{ .Vars.x }}"
	if _, err := c.RenderTitle(RepoData{FullName: "a/b"}); err == nil || !strings.Contains(err.Error(), `missing variable "x"`) {
		t.Fatalf("nil vars: %v", err)
	}

	got, err := c.RenderList("reviewers", []string{"{{ join .Vars.reviewers \",\" }}", "a", "", "acme/{{ .Vars.team }}"}, repo)
	if err != nil || strings.Join(got, "|") != "a|b|acme/pay" {
		t.Fatalf("list = %v, %v", got, err)
	}
	if _, err := c.RenderList("labels", []string{"{{ .Vars.nope }}"}, repo); err == nil {
		t.Fatal("expected missing variable error")
	}
}

func TestFormatValueAndEnvName(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		v    any
		want string
	}{
		{nil, ""}, {"s", "s"}, {true, "true"}, {3, "3"}, {int64(4), "4"}, {2.0, "2"}, {1.5, "1.5"},
		{[]any{"a", 1, false}, "a,1,false"}, {[]string{"x", "y"}, "x,y"}, {struct{}{}, "{}"},
	} {
		if got := FormatValue(tt.v); got != tt.want {
			t.Errorf("FormatValue(%#v) = %q", tt.v, got)
		}
	}
	if EnvName("team-name.v2") != "TEAM_NAME_V2" || EnvName("ok_1") != "OK_1" {
		t.Fatal("EnvName")
	}
	if !ValidVarName("_a1") || ValidVarName("1a") || ValidVarName("a-b") {
		t.Fatal("ValidVarName")
	}
}

func TestUsesProperties(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	body := dir + "/body.md"
	if err := os.WriteFile(body, []byte("tier {{ .Repo.Properties.tier }}"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		c    Campaign
		want bool
	}{
		{Campaign{}, false},
		{Campaign{Targets: Targets{Include: Include{Properties: map[string]StringList{"a": {"b"}}}}}, true},
		{Campaign{Targets: Targets{Exclude: Exclude{Properties: map[string]StringList{"a": {"b"}}}}}, true},
		{Campaign{Vars: Vars{Match: []VarRule{{When: When{Properties: map[string]StringList{"a": {"b"}}}}}}}, true},
		{Campaign{Vars: Vars{Match: []VarRule{{When: When{Topics: []string{"x"}}}}}}, false},
		{Campaign{PR: PR{Reviewers: []string{"acme/{{ .Repo.Properties.team }}"}}}, true},
		{Campaign{Path: dir + "/fleet.yaml", PR: PR{Body: "body.md"}}, true},
	} {
		if got := tt.c.UsesProperties(); got != tt.want {
			t.Errorf("UsesProperties(%+v) = %v", tt.c, got)
		}
	}
}
