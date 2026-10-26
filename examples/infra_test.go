package examples_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// contains fails the test unless the file at path holds every want.
func contains(t *testing.T, dir, path string, want ...string) {
	t.Helper()
	got := read(t, filepath.Join(dir, path))
	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Errorf("%s missing %q:\n%s", path, w, got)
		}
	}
}

func TestBuildkiteExamples(t *testing.T) {
	t.Parallel()
	pipeline := "agents:\n  queue: default\nsteps:\n  - label: test\n    command: make test\n    plugins:\n      - docker-compose#v4.14.0:\n          run: app\n  - label: slow\n    command: make e2e\n    timeout_in_minutes: 90\n    agents: { queue: gpu }\n"
	dir := scratch(t, map[string]string{".buildkite/pipeline.yml": pipeline})
	vars := runTwice(t, "buildkite-plugin-bump", dir)
	contains(t, dir, ".buildkite/pipeline.yml", "docker-compose#v5.4.1:")
	if vars["previous"] != "v4.14.0" {
		t.Errorf("previous = %v", vars["previous"])
	}
	runTwice(t, "buildkite-agents", dir)
	contains(t, dir, ".buildkite/pipeline.yml", "  queue: linux-small\n", "queue: gpu", "    command: make test\n", "          run: app\n    timeout_in_minutes: 30\n", "timeout_in_minutes: 90")
	if n := strings.Count(read(t, filepath.Join(dir, ".buildkite/pipeline.yml")), "timeout_in_minutes"); n != 2 {
		t.Errorf("want 2 timeouts, got %d", n)
	}
}

func TestBuildFileExamples(t *testing.T) {
	t.Parallel()
	dir := scratch(t, map[string]string{"Makefile": "GO_VERSION ?= 1.22  # toolchain\n", "go.mod": "module a\n\ngo 1.22\n"})
	runTwice(t, "make-go-version", dir)
	contains(t, dir, "Makefile", "GO_VERSION ?= 1.27.3  # toolchain")
	contains(t, dir, "go.mod", "go 1.27.3")

	dir = scratch(t, map[string]string{"CMakeLists.txt": "cmake_minimum_required(VERSION 3.14)\nFetchContent_Declare(\n  googletest\n  GIT_REPOSITORY https://github.com/google/googletest.git\n  GIT_TAG v1.13.0\n)\n"})
	runTwice(t, "cmake-minimum", dir)
	contains(t, dir, "CMakeLists.txt", "VERSION 3.20)", "GIT_TAG v1.15.2")
}

func TestDeployExamples(t *testing.T) {
	t.Parallel()
	dir := scratch(t, map[string]string{
		"deploy/prod/kustomization.yaml": "images:\n  - name: ghcr.io/acme/api\n    newTag: \"2.3.1\"\n",
		"deploy/dev/kustomization.yaml":  "images:\n  - name: ghcr.io/acme/web\n    newTag: \"1.0\"\n",
	})
	vars := runTwice(t, "kustomize-image", dir)
	contains(t, dir, "deploy/prod/kustomization.yaml", `newTag: "2.4.0"`)
	contains(t, dir, "deploy/dev/kustomization.yaml", `newTag: "1.0"`)
	if vars["overlays"] != "deploy/prod" {
		t.Errorf("overlays = %v", vars["overlays"])
	}

	dir = scratch(t, map[string]string{"apps/api.yaml": "apiVersion: argoproj.io/v1alpha1\nkind: Application\nspec:\n  source:\n    repoURL: https://github.com/acme/platform-manifests.git\n    targetRevision: v2.9.0\n"})
	runTwice(t, "argocd-revision", dir)
	contains(t, dir, "apps/api.yaml", "targetRevision: v3.0.0")

	dir = scratch(t, map[string]string{"clusters/ingress.yaml": "apiVersion: helm.toolkit.fluxcd.io/v2\nkind: HelmRelease\nspec:\n  chart:\n    spec:\n      chart: ingress-nginx\n      version: 4.8.x\n"})
	runTwice(t, "flux-chart-bump", dir)
	contains(t, dir, "clusters/ingress.yaml", "version: 4.11.x")

	dir = scratch(t, map[string]string{"k8s/jobs.yaml": "apiVersion: batch/v1beta1\nkind: CronJob\nmetadata:\n  name: nightly\n---\napiVersion: networking.k8s.io/v1beta1\nkind: Ingress\nmetadata:\n  name: web\n"})
	vars = runTwice(t, "k8s-api-migration", dir)
	contains(t, dir, "k8s/jobs.yaml", "apiVersion: batch/v1\nkind: CronJob", "apiVersion: networking.k8s.io/v1beta1\nkind: Ingress")
	if vars["resources"] != "CronJob/nightly" {
		t.Errorf("resources = %v", vars["resources"])
	}

	dir = scratch(t, map[string]string{"k8s/api.yaml": "apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: api\nspec:\n  template:\n    metadata:\n      labels:\n        app: api\n"})
	runTwice(t, "k8s-labels", dir, "FLEET_REPO=acme/billing", "FLEET_VAR_TEAM=payments")
	contains(t, dir, "k8s/api.yaml", "  labels:\n    app.kubernetes.io/part-of: billing\n    acme.dev/team: payments\n", "        app: api\n        app.kubernetes.io/part-of: billing\n        acme.dev/team: payments\n")

	dir = scratch(t, map[string]string{"requirements.yml": "collections:\n  - name: community.general\n"})
	runTwice(t, "ansible-collections", dir)
	contains(t, dir, "requirements.yml", "  - name: community.general\n    version: 9.4.0\n")

	dir = scratch(t, map[string]string{
		"live/prod/vpc/terragrunt.hcl": "terraform {\n  source = \"git::https://github.com/acme/terraform-modules.git//vpc?ref=v1.4.0\"\n}\n",
		"stacks/db/main.tf":            "module \"db\" {\n  source = \"git::https://github.com/acme/terraform-modules.git//rds?ref=v1.2.0\"\n}\n",
		"stacks/other/main.tf":         "module \"x\" {\n  source = \"git::https://github.com/other/mods.git//x?ref=v1.0.0\"\n}\n",
	})
	vars = runTwice(t, "terragrunt-module-ref", dir)
	contains(t, dir, "live/prod/vpc/terragrunt.hcl", "?ref=v2.0.0")
	contains(t, dir, "stacks/db/main.tf", "?ref=v2.0.0")
	contains(t, dir, "stacks/other/main.tf", "?ref=v1.0.0")
	if vars["previous"] != "v1.2.0, v1.4.0" {
		t.Errorf("previous = %v", vars["previous"])
	}
}

func TestHygieneExamples(t *testing.T) {
	t.Parallel()
	dir := scratch(t, map[string]string{"Jenkinsfile": "@Library('ci-lib@v2.4') _\nbuild()\n", ".circleci/config.yml": "orbs:\n  ci: acme/ci-orb@2.1.0\n"})
	runTwice(t, "jenkins-shared-library", dir)
	contains(t, dir, "Jenkinsfile", "@Library('ci-lib@v3.0.0')")
	contains(t, dir, ".circleci/config.yml", "acme/ci-orb@3.0.0")

	dir = scratch(t, map[string]string{"package.json": "{\n  \"dependencies\": {\n    \"lodash\": \"^4.17.15\"\n  }\n}\n", "CHANGELOG.md": "# Changelog\n\n## [1.0.0] - 2026-01-01\n\n- First.\n"})
	runTwice(t, "changelog-entry", dir)
	contains(t, dir, "CHANGELOG.md", "## [Unreleased]\n\n### Security\n\n- Update lodash to ^4.17.21.\n\n## [1.0.0]")
	untouched := scratch(t, map[string]string{"package.json": "{\"dependencies\": {\"react\": \"18\"}}", "CHANGELOG.md": "# Changelog\n"})
	runScript(t, "changelog-entry", untouched)
	if read(t, filepath.Join(untouched, "CHANGELOG.md")) != "# Changelog\n" {
		t.Error("changelog entry added without a bump")
	}

	dir = scratch(t, map[string]string{"LICENSE": "MIT License\n\nCopyright (c) 2021 Acme\n\nPermission is hereby granted, free of charge, to any person\n"})
	vars := runTwice(t, "license-year", dir)
	contains(t, dir, "LICENSE", "Copyright (c) 2021-2026 Acme")
	if vars["license"] != "MIT" {
		t.Errorf("license = %v", vars["license"])
	}
	gpl := scratch(t, map[string]string{"COPYING": "GNU GENERAL PUBLIC LICENSE\nVersion 3\nCopyright (C) 2007\n"})
	runScript(t, "license-year", gpl)
	contains(t, gpl, "COPYING", "Copyright (C) 2007\n")

	dir = scratch(t, map[string]string{".editorconfig": "[*.py]\nindent_size = 4\n"})
	runTwice(t, "editor-defaults", dir)
	contains(t, dir, ".editorconfig", "root = true\n\n[*.py]\nindent_size = 4\n\n[*]\nend_of_line = lf\ninsert_final_newline = true\n\n[Makefile]\nindent_style = tab\n")
	contains(t, dir, ".gitattributes", "* text=auto\n*.sh text eol=lf\n")

	dir = scratch(t, map[string]string{".github/workflows/ci.yml": "on:\n  push:\n    branches: [master]\n"})
	runTwice(t, "branch-rename", dir)
	contains(t, dir, ".github/workflows/ci.yml", "branches: [main]")
	if _, err := os.Stat(filepath.Join(dir, ".github/workflows/ci.yml")); err != nil {
		t.Fatal(err)
	}
}
