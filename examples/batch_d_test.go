package examples_test

import "testing"

func TestBatchDExamples(t *testing.T) {
	t.Parallel()
	dir := scratch(t, map[string]string{".github/workflows/ci.yml": "on: push\n"})
	vars := runTwice(t, "codeql-setup", dir)
	if vars["added"] != true || vars["languages"] != "actions" {
		t.Errorf("vars = %v", vars)
	}
	contains(t, dir, ".github/workflows/codeql.yml", "- language: actions\n            build-mode: none", "github/codeql-action/analyze@v4")

	dir = scratch(t, map[string]string{".github/workflows/codeql.yml": "jobs:\n  analyze:\n    strategy:\n      matrix:\n        language: [go]\n    steps:\n      - uses: github/codeql-action/init@v3\n"})
	vars = runTwice(t, "codeql-setup", dir)
	if vars["added"] != false || vars["languages"] != "actions, go" {
		t.Errorf("vars = %v", vars)
	}
	contains(t, dir, ".github/workflows/codeql.yml", "language: [go, actions]", "codeql-action/init@v4")

	dir = scratch(t, map[string]string{"README.md": "# web\n\nThe web app.\n"})
	runTwice(t, "scorecard-rollout", dir)
	contains(t, dir, ".github/workflows/scorecard.yml", "publish_results: true", "id-token: write")
	contains(t, dir, "README.md", "# web\n\n[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/acme/web/badge)]")

	dir = scratch(t, map[string]string{".trivyignore": "# Waiting on the base image.\nCVE-2024-1234 exp:2020-01-31\n\nCVE-2025-0001 exp:2999-12-31\n"})
	vars = runTwice(t, "trivy-ignore-prune", dir)
	if vars["expired"] != "CVE-2024-1234" {
		t.Errorf("expired = %v", vars["expired"])
	}
	if got := read(t, dir+"/.trivyignore"); got != "CVE-2025-0001 exp:2999-12-31\n" {
		t.Errorf(".trivyignore = %q", got)
	}
}
