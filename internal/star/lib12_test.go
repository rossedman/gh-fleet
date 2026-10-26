package star

import "testing"

// TestLibRealRepoFixes covers what running the examples against real
// repositories turned up.
func TestLibRealRepoFixes(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]libCase{
		"eslint keeps rule options": {
			repo: map[string]string{
				".eslintrc.yml":  "rules:\n  eqeqeq: [error, allow-null]  # legacy\n  indent: [warn, 2, { SwitchCase: 1 }]\n  no-restricted-globals:\n    - error\n    - name: Buffer\n  semi: warn\n",
				"a/package.json": "{}",
			},
			script: `load("@fleet//eslint.star", "set_rule")
def change(repo):
    set_rule(repo, "eqeqeq", "warn")
    set_rule(repo, "indent", "error")
    set_rule(repo, "no-restricted-globals", "warn")
    set_rule(repo, "semi", "error")
    set_rule(repo, "no-console", "error")
`,
			want: map[string]string{".eslintrc.yml": "rules:\n  eqeqeq: [warn, allow-null]  # legacy\n  indent: [error, 2, { SwitchCase: 1 }]\n  no-restricted-globals:\n    - warn\n    - name: Buffer\n  semi: error\n  no-console: error\n"},
		},
		"eslint json keeps rule options": {
			repo: map[string]string{".eslintrc.json": "{\n  \"rules\": {\"eqeqeq\": [\"error\", \"smart\"]}\n}\n"},
			script: `load("@fleet//eslint.star", "set_rule")
def change(repo):
    set_rule(repo, "eqeqeq", "warn")
`,
			want: map[string]string{".eslintrc.json": "{\n  \"rules\": {\"eqeqeq\": [\"warn\", \"smart\"]}\n}\n"},
		},
		"gitlab default after a spec header": {
			repo: map[string]string{".gitlab-ci.yml": "spec:\n  inputs:\n    release:\n      default: false\n---\nvariables:\n  GO: \"1.23\"\n"},
			script: `load("@fleet//gitlab.star", "set_default")
def change(repo):
    set_default(repo, "interruptible", True)
`,
			want: map[string]string{".gitlab-ci.yml": "spec:\n  inputs:\n    release:\n      default: false\n---\nvariables:\n  GO: \"1.23\"\ndefault:\n  interruptible: true\n"},
		},
		"switching package managers and aliases": {
			repo: map[string]string{"package.json": "{\"packageManager\": \"yarn@4.5.0\"}", ".nvmrc": "lts/iron\n", ".python-version": "3.11\n"},
			script: `load("@fleet//npm.star", "set_package_manager")
load("@fleet//runtime.star", "set_version")
def change(repo):
    set_package_manager(repo, "pnpm@9.12.0")
    set_version(repo, "node", "22")
    set_version(repo, "python", "3.12")
`,
			want: map[string]string{"package.json": "{\"packageManager\": \"pnpm@9.12.0\"}\n", ".nvmrc": "lts/iron\n", ".python-version": "3.12\n"},
		},
		"bumps never lower unless asked exactly": {
			repo: map[string]string{
				".github/workflows/ci.yml": "steps:\n  - uses: actions/setup-node@v5\n  - uses: actions/setup-node@v3\n  - uses: actions/setup-node@main\n    with:\n      node-version: 24\n      python-version: \"3.11\"\n",
				".buildkite/pipeline.yml":  "plugins:\n  - docker-compose#v5.6.0: ~\n",
			},
			script: `load("@fleet//gha.star", "bump_action", "set_input")
load("@fleet//refs.star", "bump")
def change(repo):
    bump_action(repo, "actions/setup-node", "v4")
    set_input(repo, "node-version", "22", never_lower = True)
    set_input(repo, "python-version", "3.12", never_lower = True)
    bump(repo, [".buildkite/*.yml"], "docker-compose", "v5.4.1", sep = "#")
    repo.set_var("exact", bump(repo, [".buildkite/*.yml"], "docker-compose", "v5.4.1", sep = "#", from_ref = "v5.6.0"))
`,
			want: map[string]string{
				".github/workflows/ci.yml": "steps:\n  - uses: actions/setup-node@v5\n  - uses: actions/setup-node@v4\n  - uses: actions/setup-node@v4\n    with:\n      node-version: 24\n      python-version: \"3.12\"\n",
				".buildkite/pipeline.yml":  "plugins:\n  - docker-compose#v5.4.1: ~\n",
			},
			vars: map[string]string{"exact": "[.buildkite/pipeline.yml]"},
		},
		"never_lower siblings": {
			repo: map[string]string{"helmfile.yaml": "releases:\n  - chart: bitnami/redis\n    version: 21.0.0\n  - chart: bitnami/postgresql\n    version: 15.0.0\n"},
			script: `load("@fleet//helmfile.star", "set_chart_version")
def change(repo):
    set_chart_version(repo, "bitnami/redis", "20.2.1")
    set_chart_version(repo, "bitnami/postgresql", "16.1.0")
`,
			want: map[string]string{"helmfile.yaml": "releases:\n  - chart: bitnami/redis\n    version: 21.0.0\n  - chart: bitnami/postgresql\n    version: 16.1.0\n"},
		},
		"badges join existing ones under a logo": {
			repo: map[string]string{
				"README.md":      "![Logo](docs/logo.png)\n\nTagline\n\n[![License](https://img.shields.io/github/license/acme/api)](LICENSE)\n\nIntro.\n\n## Get started\n\n- [Docs](https://acme.dev)\n",
				"docs/README.md": "## Usage\n\nRun it.\n",
			},
			script: `load("@fleet//markdown.star", "ensure_badge")
def change(repo):
    ensure_badge(repo, "README.md", "[![CI](https://github.com/acme/api/actions/workflows/ci.yml/badge.svg)](https://github.com/acme/api/actions)")
    ensure_badge(repo, "docs/README.md", "[![CI](x/badge.svg)](x)")
`,
			want: map[string]string{
				"README.md":      "![Logo](docs/logo.png)\n\nTagline\n\n[![License](https://img.shields.io/github/license/acme/api)](LICENSE)\n[![CI](https://github.com/acme/api/actions/workflows/ci.yml/badge.svg)](https://github.com/acme/api/actions)\n\nIntro.\n\n## Get started\n\n- [Docs](https://acme.dev)\n",
				"docs/README.md": "[![CI](x/badge.svg)](x)\n\n## Usage\n\nRun it.\n",
			},
		},
	} {
		runLib(t, name, tc)
	}
}
