package examples_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPythonRequirementBump(t *testing.T) {
	t.Parallel()
	dir := scratch(t, map[string]string{
		"requirements.txt":       "Django==4.2.1  # web\ncelery==5.3\n",
		"pyproject.toml":         "[project]\ndependencies = [\"django>=4.0\"]\n",
		"other/requirements.txt": "flask==3\n",
	})
	vars := runTwice(t, "python-requirement-bump", dir)
	if got := read(t, filepath.Join(dir, "requirements.txt")); got != "Django==4.2.16  # web\ncelery==5.3\n" {
		t.Errorf("requirements.txt = %q", got)
	}
	if got := read(t, filepath.Join(dir, "pyproject.toml")); !strings.Contains(got, `"django==4.2.16"`) {
		t.Errorf("pyproject.toml = %q", got)
	}
	if vars["files"] != "requirements.txt, pyproject.toml" {
		t.Errorf("files = %v", vars["files"])
	}
}

func TestPythonVersion(t *testing.T) {
	t.Parallel()
	dir := scratch(t, map[string]string{
		".python-version":          "3.10\n",
		"pyproject.toml":           "[project]\nrequires-python = \">=3.8\"\n",
		".github/workflows/ci.yml": "      - uses: actions/setup-python@v5\n        with:\n          python-version: \"3.10\"\n      - with:\n          python-version: ${{ matrix.py }}\n",
	})
	runTwice(t, "python-version", dir)
	if got := read(t, filepath.Join(dir, ".python-version")); got != "3.12\n" {
		t.Errorf(".python-version = %q", got)
	}
	if got := read(t, filepath.Join(dir, "pyproject.toml")); !strings.Contains(got, `requires-python = ">=3.12"`) {
		t.Errorf("pyproject.toml = %q", got)
	}
	if got := read(t, filepath.Join(dir, ".github/workflows/ci.yml")); !strings.Contains(got, `python-version: "3.12"`) || !strings.Contains(got, "matrix.py") {
		t.Errorf("workflow = %q", got)
	}
}

func TestNodeVersion(t *testing.T) {
	t.Parallel()
	dir := scratch(t, map[string]string{".tool-versions": "nodejs 18.19.0\n", ".github/workflows/ci.yml": "        with:\n          node-version: 18\n"})
	runTwice(t, "node-version", dir, "FLEET_VAR_NODE=22")
	if got := read(t, filepath.Join(dir, ".tool-versions")); got != "nodejs 22\n" {
		t.Errorf(".tool-versions = %q", got)
	}
	if got := read(t, filepath.Join(dir, ".github/workflows/ci.yml")); !strings.Contains(got, "node-version: 22") {
		t.Errorf("workflow = %q", got)
	}
	// Nothing pins Node yet: .nvmrc is created.
	bare := t.TempDir()
	runScript(t, "node-version", bare, "FLEET_VAR_NODE=22")
	if got := read(t, filepath.Join(bare, ".nvmrc")); got != "22\n" {
		t.Errorf(".nvmrc = %q", got)
	}
}

func TestCargoDependencyBump(t *testing.T) {
	t.Parallel()
	path, log := stub(t, "cargo")
	dir := scratch(t, map[string]string{
		"Cargo.toml": "[dependencies]\ntokio = { version = \"1.30\", features = [\"full\"] }\n",
		"Cargo.lock": "# lock\n",
	})
	runScript(t, "cargo-dependency-bump", dir, path)
	if got := read(t, filepath.Join(dir, "Cargo.toml")); !strings.Contains(got, `tokio = { version = "1.40", features = ["full"] }`) {
		t.Errorf("Cargo.toml = %q", got)
	}
	// Contains, not equality: macOS reports /var temp dirs as /private/var.
	if got := read(t, log); !strings.Contains(got, dir+" update --package tokio\n") || strings.Count(got, "\n") != 1 {
		t.Errorf("cargo calls = %q", got)
	}
	// No lockfile, no cargo.
	path, log = stub(t, "cargo")
	runScript(t, "cargo-dependency-bump", scratch(t, map[string]string{"Cargo.toml": "[dependencies]\ntokio = \"1\"\n"}), path)
	if strings.TrimSpace(readOr(log)) != "" {
		t.Error("cargo ran without a Cargo.lock")
	}
	// A repo that doesn't depend on the crate: cargo update would fail.
	path, log = stub(t, "cargo")
	runScript(t, "cargo-dependency-bump", scratch(t, map[string]string{"Cargo.toml": "[dependencies]\nserde = \"1\"\n", "Cargo.lock": "# lock\n"}), path)
	if strings.TrimSpace(readOr(log)) != "" {
		t.Error("cargo ran although no dependency changed")
	}
}

func TestPreCommitRev(t *testing.T) {
	t.Parallel()
	dir := scratch(t, map[string]string{".pre-commit-config.yaml": "repos:\n  - repo: https://github.com/pre-commit/pre-commit-hooks\n    rev: v4.6.0\n    hooks:\n      - id: check-yaml\n"})
	vars := runScript(t, "pre-commit-rev", dir)
	if got := read(t, filepath.Join(dir, ".pre-commit-config.yaml")); !strings.Contains(got, "rev: v5.0.0") {
		t.Errorf("config = %q", got)
	}
	if vars["previous"] != "v4.6.0" {
		t.Errorf("previous = %v", vars["previous"])
	}
	runTwice(t, "pre-commit-rev", dir)
}

func TestReadmeSection(t *testing.T) {
	t.Parallel()
	dir := scratch(t, map[string]string{"README.md": "# billing\n\nBills things.\n\n## Security\n\nEmail bob.\n\n## License\n\nMIT\n"})
	runTwice(t, "readme-section", dir, "FLEET_REPO=acme/billing")
	got := read(t, filepath.Join(dir, "README.md"))
	for _, want := range []string{
		"# billing\n\n[![CI](https://github.com/acme/billing/actions/workflows/ci.yml/badge.svg)]",
		"## Security\n\nPlease report vulnerabilities privately",
		"https://github.com/acme/billing/security/advisories/new",
		"Do not open a public issue.\n\n## License\n\nMIT\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("README missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Email bob") {
		t.Error("old Security section kept")
	}
}

func TestComposeImageBump(t *testing.T) {
	t.Parallel()
	dir := scratch(t, map[string]string{
		"compose.yaml":   "services:\n  cache:\n    image: redis:7-alpine\n  pinned:\n    image: redis:7.2.4\n",
		"k8s/redis.yaml": "containers:\n  - image: \"redis:7\"\n",
		"Dockerfile":     "FROM redis:7\n",
	})
	vars := runTwice(t, "compose-image-bump", dir)
	if got := read(t, filepath.Join(dir, "compose.yaml")); got != "services:\n  cache:\n    image: redis:8-alpine\n  pinned:\n    image: redis:7.2.4\n" {
		t.Errorf("compose.yaml = %q", got)
	}
	if got := read(t, filepath.Join(dir, "k8s/redis.yaml")); !strings.Contains(got, `"redis:8"`) {
		t.Errorf("manifest = %q", got)
	}
	if vars["files"] != "compose.yaml, k8s/redis.yaml, Dockerfile" {
		t.Errorf("files = %v", vars["files"])
	}
}

func TestTerraformProviderBump(t *testing.T) {
	t.Parallel()
	tf := "terraform {\n  required_version = \">= 1.5\"\n  required_providers {\n    aws = {\n      source  = \"hashicorp/aws\"\n      version = \"~> 4.67\"\n    }\n  }\n}\n"
	dir := scratch(t, map[string]string{"versions.tf": tf, "other/versions.tf": "terraform {\n  required_version = \">= 1.0\"\n}\n"})
	runTwice(t, "terraform-provider-bump", dir)
	got := read(t, filepath.Join(dir, "versions.tf"))
	if !strings.Contains(got, `version = "~> 5.0"`) || !strings.Contains(got, `required_version = ">= 1.9"`) {
		t.Errorf("versions.tf = %q", got)
	}
	// A repo without the provider keeps its required_version.
	clean := scratch(t, map[string]string{"main.tf": "terraform {\n  required_version = \">= 1.0\"\n}\n"})
	runScript(t, "terraform-provider-bump", clean)
	if !strings.Contains(read(t, filepath.Join(clean, "main.tf")), ">= 1.0") {
		t.Error("required_version raised without the provider")
	}
}

func TestSecurityPolicy(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	runTwice(t, "security-policy", dir, "FLEET_REPO=acme/billing")
	got := read(t, filepath.Join(dir, "SECURITY.md"))
	if !strings.Contains(got, "https://github.com/acme/billing/security/advisories/new") || strings.Contains(got, "{repo}") {
		t.Errorf("SECURITY.md = %q", got)
	}
	existing := scratch(t, map[string]string{".github/SECURITY.md": "ours\n"})
	runScript(t, "security-policy", existing, "FLEET_REPO=acme/billing")
	if s := snapshot(t, existing); len(s) != 1 {
		t.Errorf("policy added next to an existing one: %v", fmt.Sprint(s))
	}
}

// readOr returns a file's content, or "" when it does not exist.
func readOr(path string) string {
	b, _ := os.ReadFile(path)
	return string(b)
}
