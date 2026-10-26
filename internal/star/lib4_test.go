package star

import (
	"strings"
	"testing"
)

func TestLibRefsAndCI(t *testing.T) {
	t.Parallel()
	sha := strings.Repeat("1", 40)
	for name, tc := range map[string]libCase{
		"refs": {
			repo: map[string]string{
				"main.tf":                   "module \"a\" { source = \"git::https://x/m.git//vpc?ref=v1.0.0&depth=1\" }\nmodule \"b\" { source = \"git::https://x/m.git//db?ref=" + sha + "\" }\n",
				".terraform/modules/x.tf":   "source = \"git::https://x/m.git//vpc?ref=v0.1\"\n",
				"Jenkinsfile":               "@Library(['ci-lib@v2', 'xci-lib@v2']) _\n",
				".circleci/config.yml":      "orbs:\n  node: circleci/node@5.1.0\n  slack: circleci/slack@4.12.0\n",
				".github/workflows/ci.yaml": "steps:\n  - uses: acme/act@v1 # old\n",
			},
			script: `load("@fleet//refs.star", "bump", "find", "pin")
def change(repo):
    repo.set_var("tf", bump(repo, ["**/*.tf"], "git::https://x/m.git//*", "v2.0.0", sep = "?ref=", exclude = [".terraform/**"]))
    bump(repo, ["Jenkinsfile"], "ci-lib", "v3")
    bump(repo, [".circleci/*.yml"], "circleci/*", "6.0.0", from_ref = "5.1.0")
    pin(repo, [".github/workflows/*"], "acme/*", "acme/act@v1=` + sha + `", comment = None)
    repo.set_var("found", len(find(repo, ["**/*.tf"], "*", sep = "?ref=")))
`,
			want: map[string]string{
				"main.tf":                   "module \"a\" { source = \"git::https://x/m.git//vpc?ref=v2.0.0&depth=1\" }\nmodule \"b\" { source = \"git::https://x/m.git//db?ref=" + sha + "\" }\n",
				".terraform/modules/x.tf":   "source = \"git::https://x/m.git//vpc?ref=v0.1\"\n",
				"Jenkinsfile":               "@Library(['ci-lib@v3', 'xci-lib@v2']) _\n",
				".circleci/config.yml":      "orbs:\n  node: circleci/node@6.0.0\n  slack: circleci/slack@4.12.0\n",
				".github/workflows/ci.yaml": "steps:\n  - uses: acme/act@" + sha + " # old\n",
			},
			vars: map[string]string{"tf": "[main.tf]", "found": "3"},
		},
		"buildkite": {
			repo: map[string]string{".buildkite/pipeline.yml": `agents:
  queue: default
steps:
  - group: tests
    steps:
      - label: unit
        commands:
          - make test
      - label: lint
        command: |
          make lint
        plugins: ["docker#v5.0.0", "https://github.com/acme/x-buildkite-plugin.git#v1.0.0"]
  - wait
  - block: release
  - label: deploy
    command: make deploy
    agents: { queue: "deploy", os: linux }
    retry: { automatic: true }
`},
			script: `load("@fleet//buildkite.star", "bump_plugin", "plugins", "set_queue", "set_step_defaults")
def change(repo):
    bump_plugin(repo, "docker", "v5.11.0")
    bump_plugin(repo, "https://github.com/acme/x-buildkite-plugin.git", "v1.1.0")
    set_queue(repo, "deploy-v2", from_queue = "deploy")
    set_step_defaults(repo, {"timeout_in_minutes": 30, "soft_fail": False, "priority": "low"})
    repo.set_var("plugins", plugins(repo))
`,
			want: map[string]string{".buildkite/pipeline.yml": `agents:
  queue: default
steps:
  - group: tests
    steps:
      - label: unit
        commands:
          - make test
        timeout_in_minutes: 30
        soft_fail: false
        priority: "low"
      - label: lint
        command: |
          make lint
        plugins: ["docker#v5.11.0", "https://github.com/acme/x-buildkite-plugin.git#v1.1.0"]
        timeout_in_minutes: 30
        soft_fail: false
        priority: "low"
  - wait
  - block: release
  - label: deploy
    command: make deploy
    agents: { queue: "deploy-v2", os: linux }
    retry: { automatic: true }
    timeout_in_minutes: 30
    soft_fail: false
    priority: "low"
`},
			vars: map[string]string{"plugins": "[]"},
		},
		"make and cmake": {
			repo: map[string]string{
				"Makefile":           "override GO_VERSION := 1.21\nGO_VERSION += -extra\nexport GO_VERSION ::= 1.21\nOTHER = 1\n",
				"sub/x.mk":           "GO_VERSION = 1.21\n",
				"CMakeLists.txt":     "cmake_minimum_required(VERSION 3.16...3.28)\nFetchContent_Declare(\n  fmt\n  GIT_TAG 9.0.0\n)\nFetchContent_Declare(json GIT_REPOSITORY x GIT_TAG v3.10.0)\n",
				"lib/CMakeLists.txt": "cmake_minimum_required(VERSION 3.30)\n",
			},
			script: `load("@fleet//make.star", "get_var", "set_var")
load("@fleet//cmake.star", "set_fetchcontent_tag", "set_minimum_version")
def change(repo):
    repo.set_var("make", set_var(repo, "GO_VERSION", "1.27", files = ["Makefile"]))
    repo.set_var("missing", get_var(repo, "Makefile", "NOPE") == None)
    repo.set_var("cmake", set_minimum_version(repo, "3.20"))
    set_fetchcontent_tag(repo, "fmt", "11.0.2")
    set_fetchcontent_tag(repo, "json", "v3.11.3")
`,
			want: map[string]string{
				"Makefile":           "override GO_VERSION := 1.27\nGO_VERSION += -extra\nexport GO_VERSION ::= 1.27\nOTHER = 1\n",
				"sub/x.mk":           "GO_VERSION = 1.21\n",
				"CMakeLists.txt":     "cmake_minimum_required(VERSION 3.20...3.28)\nFetchContent_Declare(\n  fmt\n  GIT_TAG 11.0.2\n)\nFetchContent_Declare(json GIT_REPOSITORY x GIT_TAG v3.11.3)\n",
				"lib/CMakeLists.txt": "cmake_minimum_required(VERSION 3.30)\n",
			},
			vars: map[string]string{"make": "[Makefile]", "missing": "true", "cmake": "[CMakeLists.txt]"},
		},
	} {
		runLib(t, name, tc)
	}
}

func TestLibDeploy(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]libCase{
		"kustomize and argocd": {
			repo: map[string]string{
				"k/kustomization.yaml": "images:\n  - name: api\n    newTag: v1\n  - name: worker\n",
				"apps/a.yaml":          "apiVersion: argoproj.io/v1alpha1\nkind: Application\nspec:\n  sources:\n    - repoURL: https://charts.acme.dev\n      chart: api\n      targetRevision: 1.0.0\n    - repoURL: https://github.com/acme/values.git\n      targetRevision: main\n",
				"apps/other.yaml":      "repoURL: https://charts.acme.dev\ntargetRevision: 1.0.0\n",
			},
			script: `load("@fleet//kustomize.star", "set_image")
load("@fleet//argocd.star", "set_target_revision")
def change(repo):
    set_image(repo, "worker", new_name = "ghcr.io/acme/worker", new_tag = "v2", digest = "sha256:abc")
    repo.set_var("none", set_image(repo, "nope", new_tag = "x"))
    set_target_revision(repo, "https://charts.acme.dev", "1.1.0")
`,
			want: map[string]string{
				"k/kustomization.yaml": "images:\n  - name: api\n    newTag: v1\n  - name: worker\n    newName: ghcr.io/acme/worker\n    newTag: v2\n    digest: \"sha256:abc\"\n",
				"apps/a.yaml":          "apiVersion: argoproj.io/v1alpha1\nkind: Application\nspec:\n  sources:\n    - repoURL: https://charts.acme.dev\n      chart: api\n      targetRevision: 1.1.0\n    - repoURL: https://github.com/acme/values.git\n      targetRevision: main\n",
				"apps/other.yaml":      "repoURL: https://charts.acme.dev\ntargetRevision: 1.0.0\n",
			},
			vars: map[string]string{"none": "[]"},
		},
		"flux, ansible, terragrunt": {
			repo: map[string]string{
				"c/src.yaml":                              "apiVersion: source.toolkit.fluxcd.io/v1\nkind: GitRepository\nspec:\n  url: https://github.com/acme/deploy\n  ref:\n    branch: main\n",
				"roles/requirements.yml":                  "roles:\n  - src: geerlingguy.docker\n    version: 6.0.0\n",
				"live/terragrunt.hcl":                     "terraform {\n  source = \"git::https://x/m.git//vpc?ref=v1\"\n}\n",
				"live/.terragrunt-cache/x/terragrunt.hcl": "source = \"git::https://x/m.git//vpc?ref=v0\"\n",
			},
			script: `load("@fleet//flux.star", "set_git_tag")
load("@fleet//ansible.star", "set_version")
load("@fleet//terragrunt.star", "set_source_ref")
def change(repo):
    repo.set_var("branch", set_git_tag(repo, "https://github.com/acme/deploy", "v2"))
    set_version(repo, "geerlingguy.docker", "7.4.1")
    set_source_ref(repo, "git::https://x/m.git//vpc", "v2")
`,
			want: map[string]string{
				"c/src.yaml":                              "apiVersion: source.toolkit.fluxcd.io/v1\nkind: GitRepository\nspec:\n  url: https://github.com/acme/deploy\n  ref:\n    branch: main\n",
				"roles/requirements.yml":                  "roles:\n  - src: geerlingguy.docker\n    version: 7.4.1\n",
				"live/terragrunt.hcl":                     "terraform {\n  source = \"git::https://x/m.git//vpc?ref=v2\"\n}\n",
				"live/.terragrunt-cache/x/terragrunt.hcl": "source = \"git::https://x/m.git//vpc?ref=v0\"\n",
			},
			vars: map[string]string{"branch": "[]"},
		},
		"k8s": {
			repo: map[string]string{
				"k8s/all.yaml": "# resources\napiVersion: batch/v1beta1\nkind: CronJob\nmetadata:\n  name: nightly\nspec:\n  jobTemplate:\n    spec:\n      template:\n        spec: {}\n\n---\napiVersion: v1\nkind: Service\nmetadata:\n  name: api\n  labels:\n    app: api\n---\n# empty\n",
				"k8s/cm.yaml":  "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: x\n",
			},
			script: `load("@fleet//k8s.star", "migrate_api_versions", "set_label")
def change(repo):
    migrate_api_versions(repo, {"batch/v1beta1 CronJob": "batch/v1"})
    set_label(repo, "app.kubernetes.io/part-of", "shop", kinds = ["CronJob", "Service"])
    set_label(repo, "tier", "web", kinds = ["Service"], pod_templates = False)
`,
			want: map[string]string{
				"k8s/all.yaml": "# resources\napiVersion: batch/v1\nkind: CronJob\nmetadata:\n  name: nightly\n  labels:\n    app.kubernetes.io/part-of: shop\nspec:\n  jobTemplate:\n    spec:\n      template:\n        spec: {}\n        metadata:\n          labels:\n            app.kubernetes.io/part-of: shop\n\n---\napiVersion: v1\nkind: Service\nmetadata:\n  name: api\n  labels:\n    app: api\n    app.kubernetes.io/part-of: shop\n    tier: web\n---\n# empty\n",
				"k8s/cm.yaml":  "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: x\n",
			},
		},
	} {
		runLib(t, name, tc)
	}
}

func TestLibHygiene(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]libCase{
		"changelog": {
			repo: map[string]string{
				"CHANGELOG.md":      "# Changelog\n\n## [Unreleased]\n\n### Fixed\n\n- A bug.\n\n## [1.0.0]\n",
				"docs/CHANGELOG.md": "# Changelog\n",
			},
			script: `load("@fleet//changelog.star", "add_entry")
def change(repo):
    add_entry(repo, "Another fix.", section = "Fixed")
    add_entry(repo, "New thing.", section = "Added")
    repo.set_var("dup", add_entry(repo, "A bug.", section = "Fixed"))
    add_entry(repo, "First.", path = "docs/CHANGELOG.md")
`,
			want: map[string]string{
				"CHANGELOG.md":      "# Changelog\n\n## [Unreleased]\n\n### Added\n\n- New thing.\n\n### Fixed\n\n- A bug.\n- Another fix.\n\n## [1.0.0]\n",
				"docs/CHANGELOG.md": "# Changelog\n\n## [Unreleased]\n\n### Changed\n\n- First.\n",
			},
			vars: map[string]string{"dup": "false"},
		},
		"no changelog or license": {
			script: `load("@fleet//changelog.star", "add_entry")
load("@fleet//license.star", "detect", "set_spdx", "set_year")
def change(repo):
    repo.set_var("c", add_entry(repo, "x"))
    repo.set_var("d", detect(repo) == None)
    repo.set_var("y", set_year(repo, 2026))
    repo.set_var("s", set_spdx(repo, "MIT"))
`,
			vars: map[string]string{"c": "false", "d": "true", "y": "false", "s": "[]"},
		},
		"license": {
			repo: map[string]string{
				"LICENSE":        "Copyright 2019 Acme\nCopyright (c) 2026 Other\nCopyright [yyyy] [name]\nSome custom text.\n",
				"pyproject.toml": "[project]\nlicense = { text = \"MIT\" }\n",
				"composer.json":  "{\"license\": \"MIT\"}",
			},
			script: `load("@fleet//license.star", "detect", "set_spdx", "set_year")
def change(repo):
    set_year(repo, "2026")
    repo.set_var("spdx", set_spdx(repo, "Apache-2.0"))
    repo.set_var("unknown", detect(repo) == None)
`,
			want: map[string]string{"LICENSE": "Copyright 2019-2026 Acme\nCopyright (c) 2026 Other\nCopyright [yyyy] [name]\nSome custom text.\n", "pyproject.toml": "[project]\nlicense = { text = \"MIT\" }\n"},
			vars: map[string]string{"spdx": "[composer.json]", "unknown": "true"},
		},
		"editorconfig and gitattributes": {
			repo: map[string]string{".editorconfig": "[*]\nindent_size = 2\n", ".gitattributes": "*.bin -text\n"},
			script: `load("@fleet//editorconfig.star", "get", "set")
load("@fleet//gitattributes.star", gitattr = "set")
def change(repo):
    set(repo, "", "root", True)
    set(repo, "*", "indent_size", 4)
    repo.set_var("same", set(repo, "*", "indent_size", "4"))
    repo.set_var("none", get(repo, "*.go", "x") == None)
    gitattr(repo, "*.bin", ["binary", "-diff"])
    gitattr(repo, "*.bin", "binary -diff")
`,
			want: map[string]string{".editorconfig": "root = true\n\n[*]\nindent_size = 4\n", ".gitattributes": "*.bin -text binary -diff\n"},
			vars: map[string]string{"same": "false", "none": "true"},
		},
		"rename branch": {
			repo: map[string]string{".github/workflows/a.yml": "on:\n  push:\n    branches-ignore:\n      - 'master'\n    tags: [master]\njobs: {}\n"},
			script: `load("@fleet//gha.star", "rename_branch")
def change(repo):
    rename_branch(repo, "master", "main")
`,
			want: map[string]string{".github/workflows/a.yml": "on:\n  push:\n    branches-ignore:\n      - 'main'\n    tags: [master]\njobs: {}\n"},
		},
		"yamlpath lists and siblings": {
			repo: map[string]string{"a.yaml": "metadata:\n  labels:\n    app.kubernetes.io/name: api\n"},
			script: `load("@fleet//yamlpath.star", "get", "get_in", "set_sibling")
def change(repo):
    repo.set_var("label", get(repo, "a.yaml", ["metadata", "labels", "app.kubernetes.io/name"]))
    repo.set_var("nofile", set_sibling(repo, "nope.yaml", "a", "b", "c", "d"))
    repo.set_var("none", get_in("a: 1\n", "b") == None)
`,
			vars: map[string]string{"label": "api", "nofile": "false", "none": "true"},
		},
	} {
		runLib(t, name, tc)
	}
}
