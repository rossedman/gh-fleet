package star

import (
	"strings"
	"testing"
)

func TestLibBatchA(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]libCase{
		"gradle plugins and android": {
			repo: map[string]string{
				"build.gradle":              "plugins {\n  id 'com.android.library' version '8.1.0' apply false\n  id 'org.jetbrains.kotlin.android' version '1.9.0' apply false\n}\n",
				"app/build.gradle.kts":      "android {\n  compileSdk = 33\n  defaultConfig {\n    minSdk(26)\n    targetSdk = 36\n  }\n}\n",
				"gradle/libs.versions.toml": "[versions]\nkotlin = \"1.9.0\"\ncompileSdk = \"33\"\n\n[plugins]\nkotlin-android = \"org.jetbrains.kotlin.android:1.9.0\"\n",
			},
			script: `load("@fleet//android.star", "sdk_levels", "set_agp_version", "set_sdk")
load("@fleet//gradle.star", "set_kotlin_version")
def change(repo):
    set_agp_version(repo, "8.7.2")
    set_kotlin_version(repo, "2.0.21")
    set_sdk(repo, compile = 35, target = 35, min = 24)
    repo.set_var("levels", repr(sdk_levels(repo)))
`,
			want: map[string]string{
				"build.gradle":              "plugins {\n  id 'com.android.library' version '8.7.2' apply false\n  id 'org.jetbrains.kotlin.android' version '2.0.21' apply false\n}\n",
				"app/build.gradle.kts":      "android {\n  compileSdk = 35\n  defaultConfig {\n    minSdk(26)\n    targetSdk = 36\n  }\n}\n",
				"gradle/libs.versions.toml": "[versions]\nkotlin = \"2.0.21\"\ncompileSdk = \"35\"\n\n[plugins]\nkotlin-android = \"org.jetbrains.kotlin.android:2.0.21\"\n",
			},
			vars: map[string]string{"levels": `{"app/build.gradle.kts": {"compile": 35, "target": 36, "min": 26}}`},
		},
		"devcontainer": {
			repo: map[string]string{
				".devcontainer/devcontainer.json":       "{\n  /* base image */\n  \"image\": \"mcr.microsoft.com/devcontainers/base:ubuntu\",\n  \"customizations\": { \"vscode\": { \"settings\": { \"x\": \"https://example.com/a//b\" } } },\n  \"features\": {}\n}\n",
				".devcontainer/py/devcontainer.json":    "{ \"image\": \"python:3.12\" }\n",
				".devcontainer/empty/devcontainer.json": "{\n}\n",
			},
			script: `load("@fleet//devcontainer.star", "add_feature", "read", "set")
def change(repo):
    add_feature(repo, "ghcr.io/devcontainers/features/node:1", {"version": "22"})
    add_feature(repo, "ghcr.io/devcontainers/features/node:1", path = ".devcontainer/py/devcontainer.json")
    set(repo, "remoteUser", "vscode", path = ".devcontainer/empty/devcontainer.json")
    repo.set_var("url", read(repo)["customizations"]["vscode"]["settings"]["x"])
    repo.set_var("missing", read(repo, "nope.json") == None)
`,
			want: map[string]string{
				".devcontainer/devcontainer.json":       "{\n  /* base image */\n  \"image\": \"mcr.microsoft.com/devcontainers/base:ubuntu\",\n  \"customizations\": { \"vscode\": { \"settings\": { \"x\": \"https://example.com/a//b\" } } },\n  \"features\": { \"ghcr.io/devcontainers/features/node:1\": {\"version\":\"22\"} }\n}\n",
				".devcontainer/py/devcontainer.json":    "{ \"features\": { \"ghcr.io/devcontainers/features/node:1\": {} }, \"image\": \"python:3.12\" }\n",
				".devcontainer/empty/devcontainer.json": "{\n  \"remoteUser\": \"vscode\"\n}\n",
			},
			vars: map[string]string{"url": "https://example.com/a//b", "missing": "true"},
		},
		"golangci": {
			repo: map[string]string{".golangci.yml": "run:\n  timeout: 5m\nlinters-settings:\n  govet:\n    enable-all: false\n"},
			script: `load("@fleet//golangci.star", "config_version", "enable_linter", "set_setting")
def change(repo):
    enable_linter(repo, "gosec")
    set_setting(repo, "govet", "enable-all", True)
    repo.set_var("v", config_version(repo))
`,
			want: map[string]string{".golangci.yml": "run:\n  timeout: 5m\nlinters-settings:\n  govet:\n    enable-all: true\nlinters:\n  enable: [gosec]\n"},
			vars: map[string]string{"v": "1"},
		},
		"ruby and renovate": {
			repo: map[string]string{
				"x.gemspec":     "Gem::Specification.new do |s|\n  s.add_development_dependency(\"rspec\", \"~> 3.10\")\n  s.add_dependency \"rake\"\nend\n",
				"renovate.json": "{\"extends\": [\"config:recommended\"], \"packageRules\": [{\"automerge\": true}]}\n",
			},
			script: `load("@fleet//ruby.star", "set_gem", "set_rubocop_target")
load("@fleet//renovate.star", "add_package_rule", "add_preset", "remove_preset")
def change(repo):
    set_gem(repo, "rspec", "~> 3.13")
    repo.set_var("rake", set_gem(repo, "rake", ">= 13"))
    repo.set_var("rubocop", set_rubocop_target(repo, "3.3"))
    repo.set_var("preset", add_preset(repo, "config:recommended"))
    repo.set_var("rule", add_package_rule(repo, {"automerge": True}))
    repo.set_var("rm", remove_preset(repo, "nope"))
`,
			want: map[string]string{"x.gemspec": "Gem::Specification.new do |s|\n  s.add_development_dependency(\"rspec\", \"~> 3.13\")\n  s.add_dependency \"rake\"\nend\n"},
			vars: map[string]string{"rake": "[]", "rubocop": "false", "preset": "false", "rule": "false", "rm": "false"},
		},
		"dependabot and github": {
			repo: map[string]string{".github/dependabot.yml": "version: 2\nupdates:\n  - package-ecosystem: docker\n    directory: \"/\"\n    schedule:\n      interval: weekly\n"},
			script: `load("@fleet//dependabot.star", "add_update", "set_option")
load("@fleet//github.star", "add_contact_link", "add_release_category", "set_blank_issues", "set_label")
def change(repo):
    repo.set_var("dup", add_update(repo, "docker"))
    set_option(repo, "rebase-strategy", "disabled")
    add_release_category(repo, "Security", ["security"])
    add_release_category(repo, "Other", ["*"])
    set_blank_issues(repo, False)
    add_contact_link(repo, "Docs", "https://docs.acme.dev", "Read the docs: start here.")
    repo.set_var("again", add_contact_link(repo, "Docs", "https://docs.acme.dev", "x"))
    set_label(repo, "fleet", "000000", path = ".github/new-labels.yml")
`,
			want: map[string]string{
				".github/dependabot.yml":            "version: 2\nupdates:\n  - package-ecosystem: docker\n    directory: \"/\"\n    schedule:\n      interval: weekly\n    rebase-strategy: disabled\n",
				".github/release.yml":               "changelog:\n  categories:\n    - title: Security\n      labels: [security]\n    - title: Other\n      labels: [\"*\"]\n",
				".github/ISSUE_TEMPLATE/config.yml": "blank_issues_enabled: false\ncontact_links:\n  - name: Docs\n    url: https://docs.acme.dev\n    about: \"Read the docs: start here.\"\n",
				".github/new-labels.yml":            "- name: fleet\n  color: \"000000\"\n",
			},
			vars: map[string]string{"dup": "false", "again": "false"},
		},
	} {
		runLib(t, name, tc)
	}
}

func TestLibBatchAErrors(t *testing.T) {
	t.Parallel()
	for name, tt := range map[string]struct {
		script string
		repo   map[string]string
		want   string
	}{
		"devcontainer multi-line": {"load('@fleet//devcontainer.star', 'set')\ndef change(repo):\n  set(repo, 'postCreateCommand', 'x')\n", map[string]string{".devcontainer/devcontainer.json": "{\n  \"postCreateCommand\": [\n    \"a\"\n  ]\n}\n"}, "spans several lines"},
		"dependabot nested":       {"load('@fleet//dependabot.star', 'set_option')\ndef change(repo):\n  set_option(repo, 'groups.all.patterns', 'x')\n", map[string]string{".github/dependabot.yml": "version: 2\nupdates:\n  - package-ecosystem: npm\n    directory: /\n"}, "npm has no groups.all to set patterns in"},
	} {
		s, err := load(t, map[string]string{"change.star": tt.script}, Options{})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if _, err := s.Run(t.Context(), files(t, tt.repo), RepoInfo{}, nil, 0); err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: err = %v, want %q", name, err, tt.want)
		}
	}
}
