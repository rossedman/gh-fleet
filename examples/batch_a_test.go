package examples_test

import (
	"strings"
	"testing"
)

func TestBatchAExamples(t *testing.T) {
	t.Parallel()
	dir := scratch(t, map[string]string{"app/build.gradle": "android {\n    compileSdk 34\n    defaultConfig {\n        targetSdk 34\n        minSdk 24\n    }\n}\n"})
	runTwice(t, "android-target-sdk", dir)
	contains(t, dir, "app/build.gradle", "compileSdk 35", "targetSdk 35", "minSdk 24")

	dir = scratch(t, map[string]string{".devcontainer/devcontainer.json": "{\n  // Go\n  \"image\": \"mcr.microsoft.com/devcontainers/go:1.22-bookworm\",\n  \"features\": {\n    \"ghcr.io/devcontainers/features/node:1\": {}\n  }\n}\n"})
	runTwice(t, "devcontainer-image", dir)
	contains(t, dir, ".devcontainer/devcontainer.json", "// Go", "go:1.23-bookworm", "\"ghcr.io/devcontainers/features/github-cli:1\": {},\n    \"ghcr.io/devcontainers/features/node:1\": {}")

	path, log := stub(t, "go")
	dir = scratch(t, map[string]string{".golangci.yml": "linters:\n  enable:\n    - errcheck\n"})
	vars := runScript(t, "golangci-v2", dir, path)
	if vars["format"] != "1" || !strings.Contains(read(t, log), " run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0 migrate") {
		t.Errorf("format = %v, go calls = %q", vars["format"], read(t, log))
	}
	contains(t, dir, ".golangci.yml", "    - errcheck\n    - gosec\n")
	v2 := scratch(t, map[string]string{".golangci.yml": "version: \"2\"\n"})
	path, log = stub(t, "go")
	runScript(t, "golangci-v2", v2, path)
	if strings.TrimSpace(readOr(log)) != "" {
		t.Error("migrate ran on a v2 config")
	}

	path, log = stub(t, "bundle")
	dir = scratch(t, map[string]string{"Gemfile": "gem \"rails\", \"~> 7.0.8\"\n", "Gemfile.lock": "x\n"})
	runScript(t, "ruby-rails-bump", dir, path)
	contains(t, dir, "Gemfile", `gem "rails", "~> 7.2"`)
	if !strings.Contains(read(t, log), " lock --update rails") {
		t.Errorf("bundle calls = %q", read(t, log))
	}

	dir = scratch(t, map[string]string{"renovate.json": "{\n  \"extends\": [\"config:base\"]\n}\n"})
	runTwice(t, "renovate-automerge", dir)
	contains(t, dir, "renovate.json", "\"config:recommended\"", "\"automerge\": true")
	if strings.Contains(read(t, dir+"/renovate.json"), "config:base") {
		t.Error("config:base kept")
	}

	dir = scratch(t, map[string]string{".github/dependabot.yml": "version: 2\nupdates:\n  - package-ecosystem: gomod\n    directory: /\n    schedule:\n      interval: weekly\n", "Dockerfile": "FROM scratch\n", "go.mod": "module x\n"})
	vars = runTwice(t, "dependabot-ecosystems", dir)
	contains(t, dir, ".github/dependabot.yml", "package-ecosystem: docker", "open-pull-requests-limit: 5")
	if vars["added"] != "docker" {
		t.Errorf("added = %v", vars["added"])
	}

	dir = scratch(t, map[string]string{".github/release.yml": "changelog:\n  categories:\n    - title: Features\n      labels: [feature]\n"})
	runTwice(t, "github-repo-files", dir, "FLEET_REPO=acme/api")
	contains(t, dir, ".github/release.yml", "- title: Features", "- title: Security", "- title: Other changes\n      labels: [\"*\"]")
	contains(t, dir, ".github/labels.yml", "- name: fleet")
	contains(t, dir, ".github/ISSUE_TEMPLATE/config.yml", "https://github.com/acme/api/security/advisories/new")
}
