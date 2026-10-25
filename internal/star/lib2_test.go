package star

import (
	"strings"
	"testing"
)

func TestLibLanguages(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]libCase{
		"python requirements": {
			repo: map[string]string{
				"requirements.txt":       "# pinned\nDjango==4.2.1  # web\nrequests[socks]>=2.0 ; python_version > \"3\"\nflask\nnumpy==1.0\n",
				"requirements/dev.txt":   "django_extensions==1\ndjango==4.1\n",
				"pyproject.toml":         "[project]\ndependencies = [\n  \"django>=4.0,<5\",\n  'requests',\n]\n",
				".venv/requirements.txt": "django==1.0\n",
			},
			script: `load("@fleet//python.star", "set_requirement")
def change(repo):
    repo.set_var("changed", set_requirement(repo, "django", "==4.2.16"))
    set_requirement(repo, "Requests", ">=2.32")
    set_requirement(repo, "flask", "==3.0")
`,
			want: map[string]string{
				"requirements.txt":       "# pinned\nDjango==4.2.16  # web\nrequests[socks]>=2.32 ; python_version > \"3\"\nflask==3.0\nnumpy==1.0\n",
				"requirements/dev.txt":   "django_extensions==1\ndjango==4.2.16\n",
				"pyproject.toml":         "[project]\ndependencies = [\n  \"django==4.2.16\",\n  'requests>=2.32',\n]\n",
				".venv/requirements.txt": "django==1.0\n",
			},
			vars: map[string]string{"changed": "[requirements.txt requirements/dev.txt pyproject.toml]"},
		},
		"python version": {
			repo: map[string]string{"pyproject.toml": "[project]\nrequires-python = \">=3.8\"\n", "lib/pyproject.toml": "[project]\nname = \"x\"\n"},
			script: `load("@fleet//python.star", "requires_python", "set_requires_python")
def change(repo):
    repo.set_var("before", requires_python(repo))
    repo.set_var("a", set_requires_python(repo, ">=3.10"))
    repo.set_var("b", set_requires_python(repo, ">=3.10", path = "lib/pyproject.toml"))
    repo.set_var("c", set_requires_python(repo, ">=3.10", path = "missing/pyproject.toml"))
`,
			want: map[string]string{"pyproject.toml": "[project]\nrequires-python = \">=3.10\"\n", "lib/pyproject.toml": "[project]\nname = \"x\"\n"},
			vars: map[string]string{"before": ">=3.8", "a": "true", "b": "false", "c": "false"},
		},
		"cargo": {
			repo: map[string]string{
				"Cargo.toml": `[package]
name = "a"
rust-version = "1.70"

[dependencies]
serde = { version = "1.0.100", features = ["derive"] }
tokio = "1.30"
local = { path = "../local" }

[dev-dependencies]
serde = "1.0"

[target.'cfg(unix)'.dependencies]
serde = "0.9"

[features]
serde = []
`,
				"crates/b/Cargo.toml": "[package]\nname = \"b\"\nrust-version = \"1.80\"\n\n[workspace.dependencies]\nserde = \"1\"\n",
				"target/x/Cargo.toml": "[dependencies]\nserde = \"0.1\"\n",
			},
			script: `load("@fleet//cargo.star", "rust_version", "set_dependency", "set_rust_version")
def change(repo):
    repo.set_var("changed", set_dependency(repo, "serde", "1.0.210"))
    repo.set_var("rust", set_rust_version(repo, "1.75"))
    repo.set_var("b", rust_version(repo, "crates/b/Cargo.toml"))
`,
			want: map[string]string{
				"Cargo.toml": `[package]
name = "a"
rust-version = "1.75"

[dependencies]
serde = { version = "1.0.210", features = ["derive"] }
tokio = "1.30"
local = { path = "../local" }

[dev-dependencies]
serde = "1.0.210"

[target.'cfg(unix)'.dependencies]
serde = "1.0.210"

[features]
serde = []
`,
				"crates/b/Cargo.toml": "[package]\nname = \"b\"\nrust-version = \"1.80\"\n\n[workspace.dependencies]\nserde = \"1.0.210\"\n",
				"target/x/Cargo.toml": "[dependencies]\nserde = \"0.1\"\n",
			},
			vars: map[string]string{"changed": "[Cargo.toml crates/b/Cargo.toml]", "rust": "[Cargo.toml]", "b": "1.80"},
		},
		"terraform": {
			repo: map[string]string{
				"main.tf": `terraform {
  required_version = ">= 1.5"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.0" # pinned
    }
    google = { source = "hashicorp/google", version = "~> 5.0" }
    random = {
      source = "hashicorp/random"
    }
  }
}
`,
				"mod/versions.tf":         "terraform {\n  required_providers {\n    aws = { version = \"~> 3.0\", source = \"registry.terraform.io/hashicorp/aws\" }\n  }\n}\n",
				".terraform/modules/x.tf": "required_version = \"0.12\"\n",
			},
			script: `load("@fleet//terraform.star", "set_provider_version", "set_required_version")
def change(repo):
    repo.set_var("aws", set_provider_version(repo, "hashicorp/aws", "~> 5.0"))
    set_provider_version(repo, "hashicorp/random", "~> 3.0")
    set_required_version(repo, ">= 1.9")
`,
			want: map[string]string{
				"main.tf": `terraform {
  required_version = ">= 1.9"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0" # pinned
    }
    google = { source = "hashicorp/google", version = "~> 5.0" }
    random = {
      source = "hashicorp/random"
    }
  }
}
`,
				"mod/versions.tf":         "terraform {\n  required_providers {\n    aws = { version = \"~> 5.0\", source = \"registry.terraform.io/hashicorp/aws\" }\n  }\n}\n",
				".terraform/modules/x.tf": "required_version = \"0.12\"\n",
			},
			vars: map[string]string{"aws": "[main.tf mod/versions.tf]"},
		},
	} {
		runLib(t, name, tc)
	}
}

func TestLibConfigFiles(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]libCase{
		"precommit": {
			repo: map[string]string{".pre-commit-config.yaml": `repos:
  - repo: https://github.com/pre-commit/pre-commit-hooks
    rev: v4.5.0  # keep
    hooks:
      - id: trailing-whitespace
  - repo: https://github.com/psf/black.git
    rev: "23.1.0"
    hooks:
      - id: black
  - repo: local
    hooks:
      - id: x
`},
			script: `load("@fleet//precommit.star", "revs", "set_rev")
def change(repo):
    repo.set_var("a", set_rev(repo, "https://github.com/pre-commit/pre-commit-hooks", "v5.0.0"))
    repo.set_var("b", set_rev(repo, "https://github.com/psf/black", "24.10.0"))
    repo.set_var("c", set_rev(repo, "https://github.com/nope/nope", "v1"))
    repo.set_var("d", set_rev(repo, "x", "v1", path = "missing.yaml"))
    repo.set_var("revs", ["%s=%s" % (k, v) for k, v in revs(repo).items()])
`,
			want: map[string]string{".pre-commit-config.yaml": `repos:
  - repo: https://github.com/pre-commit/pre-commit-hooks
    rev: v5.0.0  # keep
    hooks:
      - id: trailing-whitespace
  - repo: https://github.com/psf/black.git
    rev: "24.10.0"
    hooks:
      - id: black
  - repo: local
    hooks:
      - id: x
`},
			vars: map[string]string{"a": "true", "b": "true", "c": "false", "d": "false", "revs": "[https://github.com/pre-commit/pre-commit-hooks=v5.0.0 https://github.com/psf/black.git=24.10.0]"},
		},
		"markdown sections": {
			repo: map[string]string{
				"README.md": "# api\n\nIntro.\n\n## Security\n\nOld text.\n\n### Details\n\nMore.\n\n## License\n\nMIT\n",
				"NEW.md":    "# new\n\n```\n## Security\n```\n",
				"ONLY.md":   "## security\nold",
			},
			script: `load("@fleet//markdown.star", "has_section", "set_section")
def change(repo):
    body = "Report vulnerabilities to security@acme.dev."
    repo.set_var("a", set_section(repo, "README.md", "Security", body))
    repo.set_var("b", set_section(repo, "NEW.md", "Security", body))
    repo.set_var("c", set_section(repo, "README.md", "License", "Apache-2.0", replace = False))
    set_section(repo, "ONLY.md", "Security", body)
    set_section(repo, "EMPTY.md", "Security", body, level = 3)
    repo.set_var("has", has_section(repo, "README.md", "details"))
`,
			want: map[string]string{
				"README.md": "# api\n\nIntro.\n\n## Security\n\nReport vulnerabilities to security@acme.dev.\n\n## License\n\nMIT\n",
				"NEW.md":    "# new\n\n```\n## Security\n```\n\n## Security\n\nReport vulnerabilities to security@acme.dev.\n",
				"ONLY.md":   "## security\n\nReport vulnerabilities to security@acme.dev.\n",
				"EMPTY.md":  "### Security\n\nReport vulnerabilities to security@acme.dev.\n",
			},
			vars: map[string]string{"a": "true", "b": "true", "c": "false", "has": "false"},
		},
		"markdown badges": {
			repo: map[string]string{
				"a.md": "# api\n\nIntro.\n",
				"b.md": "# api\n\n[![Go](go.svg)](x)\n\nIntro.\n",
				"c.md": "Intro.\n",
				"d.md": "# api\nIntro.\n",
				"e.md": "# api",
			},
			script: `load("@fleet//markdown.star", "ensure_badge")
def change(repo):
    badge = "[![CI](ci.svg)](ci)"
    for p in ["a.md", "b.md", "c.md", "d.md", "e.md"]:
        ensure_badge(repo, p, badge)
    repo.set_var("missing", ensure_badge(repo, "nope.md", badge))
`,
			want: map[string]string{
				"a.md": "# api\n\n[![CI](ci.svg)](ci)\n\nIntro.\n",
				"b.md": "# api\n\n[![Go](go.svg)](x)\n[![CI](ci.svg)](ci)\n\nIntro.\n",
				"c.md": "[![CI](ci.svg)](ci)\n\nIntro.\n",
				"d.md": "# api\n\n[![CI](ci.svg)](ci)\n\nIntro.\n",
				"e.md": "# api\n\n[![CI](ci.svg)](ci)",
			},
			vars: map[string]string{"missing": "false"},
		},
		"runtime": {
			repo: map[string]string{".nvmrc": "18\n", ".tool-versions": "nodejs 18.20.0\npython 3.11.4 # main\n", "web/.nvmrc": "16\n"},
			script: `load("@fleet//runtime.star", "set_version", "version")
def change(repo):
    repo.set_var("before", version(repo, "node"))
    repo.set_var("node", set_version(repo, "node", "22"))
    repo.set_var("python", set_version(repo, "python", "3.12.7"))
    repo.set_var("ruby", set_version(repo, "ruby", "3.3"))
    repo.set_var("go", set_version(repo, "go", "1.27.3", create = ".go-version"))
    repo.set_var("tv", version(repo, "python"))
`,
			want: map[string]string{".nvmrc": "22\n", ".tool-versions": "nodejs 22\npython 3.12.7 # main\n", "web/.nvmrc": "16\n", ".go-version": "1.27.3\n"},
			vars: map[string]string{"before": "18", "node": "[.nvmrc .tool-versions]", "python": "[.tool-versions]", "ruby": "[]", "go": "[.go-version]", "tv": "3.12.7"},
		},
	} {
		runLib(t, name, tc)
	}
}

func TestLibPinAndGoDeps(t *testing.T) {
	t.Parallel()
	sha := "1111111111111111111111111111111111111111"
	for name, tc := range map[string]libCase{
		"gha pin": {
			repo: map[string]string{
				".github/workflows/ci.yml": "steps:\n  - uses: actions/checkout@v4\n  - uses: \"github/codeql-action/init@v3\" # old\n  - uses: ./local\n  - uses: docker://alpine:3\n  - uses: actions/setup-go@0aaccfd150d50ccaeb58ebd88d36e91967a5f35b # v5\n  - uses: actions/cache@v4\n",
			},
			script: `load("@fleet//gha.star", "pin", "unpinned")
def change(repo):
    refs = unpinned(repo)
    repo.set_var("refs", refs)
    pin(repo, " ".join([r + "=` + sha + `" for r in refs if "cache" not in r]))
`,
			want: map[string]string{
				".github/workflows/ci.yml": "steps:\n  - uses: actions/checkout@" + sha + " # v4\n  - uses: \"github/codeql-action/init@" + sha + "\" # v3\n  - uses: ./local\n  - uses: docker://alpine:3\n  - uses: actions/setup-go@0aaccfd150d50ccaeb58ebd88d36e91967a5f35b # v5\n  - uses: actions/cache@v4\n",
			},
			vars: map[string]string{"refs": "[actions/cache@v4 actions/checkout@v4 github/codeql-action@v3]"},
		},
		"gomod deps": {
			repo: map[string]string{
				"go.mod":   "module a\n\nrequire (\n\tgolang.org/x/net v0.20.0 // indirect\n\tgithub.com/x/y v1.0.0\n)\n",
				"b/go.mod": "module b\n\nrequire golang.org/x/net v0.40.0\n",
				"c/go.mod": "module c\n\nrequire golang.org/x/net v0.9.0\n",
				"d/go.mod": "module d\n",
			},
			script: `load("@fleet//gomod.star", "outdated", "requires")
def change(repo):
    repo.set_var("dirs", outdated(repo, "golang.org/x/net", "v0.33.0"))
    repo.set_var("y", requires(repo)["github.com/x/y"])
`,
			want: map[string]string{"go.mod": "module a\n\nrequire (\n\tgolang.org/x/net v0.20.0 // indirect\n\tgithub.com/x/y v1.0.0\n)\n"},
			vars: map[string]string{"dirs": "[c .]", "y": "v1.0.0"},
		},
		"docker compose": {
			repo: map[string]string{
				"docker-compose.yml": "services:\n  cache:\n    image: redis:7-alpine\n  db:\n    image: \"postgres:15\"\n  other:\n    image: redis:7.2\n",
				"k8s/deploy.yaml":    "spec:\n  containers:\n    - name: c\n      image: redis:7\n",
				"node_modules/x.yml": "image: redis:7\n",
			},
			script: `load("@fleet//docker.star", "bump_image")
def change(repo):
    repo.set_var("changed", bump_image(repo, "redis", "7", "8"))
    bump_image(repo, "postgres", "15", "17")
`,
			want: map[string]string{
				"docker-compose.yml": "services:\n  cache:\n    image: redis:8-alpine\n  db:\n    image: \"postgres:17\"\n  other:\n    image: redis:7.2\n",
				"k8s/deploy.yaml":    "spec:\n  containers:\n    - name: c\n      image: redis:8\n",
				"node_modules/x.yml": "image: redis:7\n",
			},
			vars: map[string]string{"changed": "[docker-compose.yml k8s/deploy.yaml]"},
		},
	} {
		runLib(t, name, tc)
	}
}

func TestLibMoreErrors(t *testing.T) {
	t.Parallel()
	for name, tt := range map[string]struct{ script, want string }{
		"pin not a sha":   {"load('@fleet//gha.star', 'pin')\ndef change(repo):\n  pin(repo, {'a/b@v1': 'main'})\n", `a/b@v1 resolved to "main"`},
		"unknown runtime": {"load('@fleet//runtime.star', 'version')\ndef change(repo):\n  version(repo, 'cobol')\n", `unknown runtime "cobol"; have go, java, node`},
	} {
		if _, _, _, err := run(t, tt.script, nil, RepoInfo{}); err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: err = %v, want %q", name, err, tt.want)
		}
	}
}
