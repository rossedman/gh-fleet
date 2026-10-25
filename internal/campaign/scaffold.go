package campaign

import (
	"bytes"
	"fmt"
	"text/template"
)

// scaffoldTemplate is the commented campaign file written by gh fleet init.
var scaffoldTemplate = template.Must(template.New("scaffold").Parse(`# yaml-language-server: $schema={{ .SchemaURL }}
apiVersion: fleet.gh/v1
name: {{ .Name }}
description: Describe what this campaign changes and why

targets:
  orgs: [{{ .Org }}]
  include:
    # Keys combine with AND, values within a key with OR.
    # topics: [service]
    # languages: [Go]
    # files: [.github/workflows]   # path must exist on the default branch
    # search: "uses: actions/ path:.github/workflows"
    # repos: [{{ .Org }}/always-included]
  exclude:
    archived: true
    forks: true
    # repos: [{{ .Org }}/sandbox]
    # topics: [no-fleet]

change:
  # Steps run in order in one checkout of each repo. Each step is a shell
  # script (run) or a Starlark script (starlark). Shell steps run at the
  # repo root with FLEET_REPO, FLEET_CAMPAIGN, FLEET_DEFAULT_BRANCH,
  # FLEET_DRY_RUN and FLEET_OUTPUT set. A diff at the end is a change; no
  # diff is a no-op.
  steps:
    # - name: edit files
    #   starlark: ./change.star   # defines change(repo); load("@fleet//files.star", ...)
    - name: change
      run: |                       # or a script next to this file: ./scripts/change.sh
        echo "replace me" >/dev/null
  shell: bash
  timeout: 5m                      # per step
  # env:
  #   KEY: value

pr:
  title: "chore: {{ .Name }}"
  # File or inline text; a Go template with .Repo and .Campaign.
  body: |
    This PR was opened by [gh fleet](https://github.com/rossedman/gh-fleet)
    as part of the {{ "{{" }} .Campaign.Name {{ "}}" }} campaign.
  # branch: fleet/{{ .Name }}
  # base: ""          # default: the repo's default branch
  draft: false
  labels: [fleet]
  # reviewers: [octocat, {{ .Org }}/platform]
  commit:
    # message: defaults to the PR title
    signoff: false

limits:
  concurrency: 8
  maxOpenPRs: 0       # 0 means no limit
`))

// Scaffold returns a commented campaign file for a new campaign. org fills
// in targets.orgs; when empty, a placeholder org the user must edit is used.
func Scaffold(name, org string) ([]byte, error) {
	if org == "" {
		org = "my-org"
	}
	var buf bytes.Buffer
	err := scaffoldTemplate.Execute(&buf, struct{ Name, Org, SchemaURL string }{name, org, SchemaURL})
	if err != nil {
		return nil, err
	}
	if _, err := Parse(buf.Bytes(), name+".yaml"); err != nil {
		return nil, fmt.Errorf("scaffold for %q is invalid: %w", name, err)
	}
	return buf.Bytes(), nil
}
