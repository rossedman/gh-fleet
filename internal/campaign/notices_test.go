package campaign

import (
	"slices"
	"testing"
)

func TestWarningsAndNotes(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		targets  string
		warnings []string
		notes    []string
	}{
		{name: "orgs only", targets: `{orgs: [acme]}`},
		{name: "repos only", targets: `{orgs: [acme], include: {repos: [acme/api]}}`},
		{name: "filters only", targets: `{orgs: [acme], include: {topics: [svc], names: [api, acme/web, "*/x", "other-*/y"]}}`},
		{
			name:    "repos beside one filter",
			targets: `{orgs: [acme, tools], include: {repos: [acme/api], files: [go.mod]}}`,
			notes:   []string{"include.repos adds its repos to what include.files selects in acme, tools, so every repo there is still checked; with include.repos alone, the targets are only the repos it names"},
		},
		{
			name:    "repos beside several filters",
			targets: `{orgs: [acme], include: {repos: [acme/api], names: [web], search: x, properties: {team: a}}}`,
			notes:   []string{"include.repos adds its repos to what include.names, include.search and include.properties select in acme, so every repo there is still checked; with include.repos alone, the targets are only the repos it names"},
		},
		{
			name:     "a filter without orgs",
			targets:  `{include: {repos: [acme/api], languages: [Go]}}`,
			warnings: []string{"include.languages filters the repos of orgs, and targets has no orgs, so it selects nothing; the targets are only what include.repos names"},
		},
		{
			name:     "filters without orgs",
			targets:  `{include: {repos: [acme/api], names: [api], topics: [svc]}}`,
			warnings: []string{"include.names and include.topics filter the repos of orgs, and targets has no orgs, so they select nothing; the targets are only what include.repos names"},
		},
		{
			name:     "a name for another owner",
			targets:  `{orgs: [acme], include: {names: [ACME/api, other/web]}}`,
			warnings: []string{`include.names "other/web" names an owner that isn't in orgs, so it can't match; to add a repo from another owner, list it in include.repos`},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c, err := Parse([]byte("apiVersion: fleet.gh/v1\nname: x\ntargets: "+tc.targets+"\nchange: {steps: [{run: \"true\"}]}\npr: {title: t}\n"), "x.yaml")
			if err != nil {
				t.Fatal(err)
			}
			if got := c.Warnings(); !slices.Equal(got, tc.warnings) {
				t.Errorf("warnings = %q, want %q", got, tc.warnings)
			}
			if got := c.Notes(); !slices.Equal(got, tc.notes) {
				t.Errorf("notes = %q, want %q", got, tc.notes)
			}
		})
	}
}
