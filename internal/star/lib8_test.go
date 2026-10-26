package star

import (
	"strings"
	"testing"
)

func TestLibBatchC(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]libCase{
		"prettier and eslint yaml": {
			repo: map[string]string{
				".prettierrc":   "# shared\nsemi: false\n",
				".eslintrc.yml": "extends: eslint:recommended\nrules:\n  no-console: warn\n",
			},
			script: `load("@fleet//prettier.star", "get", "set_option")
load("@fleet//eslint.star", "add_extends", "set_rule")
def change(repo):
    set_option(repo, "printWidth", 100)
    repo.set_var("semi", get(repo, "semi"))
    set_rule(repo, "no-console", "error")
    repo.set_var("ext", add_extends(repo, "x"))
`,
			want: map[string]string{".prettierrc": "# shared\nsemi: false\nprintWidth: 100\n", ".eslintrc.yml": "extends: eslint:recommended\nrules:\n  no-console: error\n"},
			vars: map[string]string{"semi": "false", "ext": "false"},
		},
		"package.json configs": {
			repo: map[string]string{"package.json": "{\n  \"prettier\": {\"semi\": true},\n  \"eslintConfig\": {\"extends\": \"react-app\"}\n}\n"},
			script: `load("@fleet//prettier.star", "set_option")
load("@fleet//eslint.star", "add_extends", "set_rule")
def change(repo):
    set_option(repo, "semi", False)
    add_extends(repo, "prettier")
    set_rule(repo, "eqeqeq", "error")
`,
			want: map[string]string{"package.json": "{\n  \"prettier\": {\n    \"semi\": false\n  },\n  \"eslintConfig\": {\n    \"extends\": [\n      \"react-app\",\n      \"prettier\"\n    ],\n    \"rules\": {\n      \"eqeqeq\": \"error\"\n    }\n  }\n}\n"},
		},
		"no configs": {
			repo: map[string]string{"eslint.config.js": "export default []\n"},
			script: `load("@fleet//prettier.star", "set_option")
load("@fleet//eslint.star", "set_rule")
load("@fleet//lintconfig.star", "markdownlint_rule", "yamllint_rule", "hadolint_ignore")
def change(repo):
    repo.set_var("p", set_option(repo, "semi", False))
    repo.set_var("e", set_rule(repo, "eqeqeq", "error"))
    repo.set_var("m", markdownlint_rule(repo, "MD013", False))
    repo.set_var("y", yamllint_rule(repo, "line-length", "disable"))
    hadolint_ignore(repo, ["DL3008"])
`,
			want: map[string]string{".hadolint.yaml": "ignored:\n  - DL3008\n"},
			vars: map[string]string{"p": "false", "e": "false", "m": "false", "y": "false"},
		},
		"ci and deploy": {
			repo: map[string]string{
				"ci/azure-pipelines-release.yml": "pool: { vmImage: 'ubuntu-20.04' }\nsteps:\n  - task: NodeTool@0\n",
				"bitbucket-pipelines.yml":        "pipelines:\n  default:\n    - step:\n        image: node:18\n",
				"serverless.yml":                 "provider:\n  runtime: python3.8\nfunctions:\n  a:\n    runtime: nodejs16.x\n",
				"cdk.json":                       "{\"context\": {}}",
				"img.pkr.hcl":                    "packer {\n  required_plugins {\n    docker = { version = \"~> 1.0\", source = \"github.com/hashicorp/docker\" }\n  }\n}\n",
			},
			script: `load("@fleet//azure_pipelines.star", "bump_task", "set_vm_image")
load("@fleet//bitbucket.star", "bump_image")
load("@fleet//serverless.star", "set_runtime")
load("@fleet//cdk.star", "set_context")
load("@fleet//packer.star", "set_plugin_version")
def change(repo):
    set_vm_image(repo, "ubuntu-24.04")
    bump_task(repo, "NodeTool", 1, from_major = 0)
    bump_image(repo, "node", "18", "22")
    set_runtime(repo, "nodejs20.x")
    repo.set_var("cdk", set_context(repo, "x", True, only_existing = True))
    set_plugin_version(repo, "github.com/hashicorp/docker", "~> 1.1")
`,
			want: map[string]string{
				"ci/azure-pipelines-release.yml": "pool: { vmImage: 'ubuntu-24.04' }\nsteps:\n  - task: NodeTool@1\n",
				"bitbucket-pipelines.yml":        "pipelines:\n  default:\n    - step:\n        image: node:22\n",
				"serverless.yml":                 "provider:\n  runtime: nodejs20.x\nfunctions:\n  a:\n    runtime: nodejs20.x\n",
				"cdk.json":                       "{\"context\": {}}",
				"img.pkr.hcl":                    "packer {\n  required_plugins {\n    docker = { version = \"~> 1.1\", source = \"github.com/hashicorp/docker\" }\n  }\n}\n",
			},
			vars: map[string]string{"cdk": "[]"},
		},
	} {
		runLib(t, name, tc)
	}
	_, _, _, err := run(t, "load('@fleet//eslint.star', 'set_rule')\ndef change(repo):\n  set_rule(repo, 'x', 'error', {'a': 1})\n", map[string]string{".eslintrc.yaml": "rules: {}\n"}, RepoInfo{})
	if err == nil || !strings.Contains(err.Error(), "rule options need a JSON config") {
		t.Errorf("yaml options: %v", err)
	}
}
