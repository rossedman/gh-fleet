package star

import "testing"

func TestLibCIServices(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]libCase{
		"gitlab": {
			repo: map[string]string{
				".gitlab-ci.yml":   "include:\n  - project: acme/ci-templates\n    file: /go.yml\n  - component: gitlab.com/acme/c/lint@1.0.0\nbuild:\n  image:\n    name: golang:1.22-bookworm\n    entrypoint: [\"\"]\n  services:\n    - name: \"postgres:15\"\n      alias: db\n  script: go build\n",
				".gitlab/ci/x.yml": "job:\n  image: golang:1.22\n",
			},
			script: `load("@fleet//gitlab.star", "bump_component", "bump_image", "set_default", "set_include_ref")
def change(repo):
    repo.set_var("img", bump_image(repo, "golang", "1.22", "1.23"))
    bump_image(repo, "postgres", "15", "17")
    set_include_ref(repo, "acme/ci-templates", "v2")
    bump_component(repo, "gitlab.com/acme/c/*", "2.0.0", from_ref = "1.0.0")
    set_default(repo, "timeout", "30m")
`,
			want: map[string]string{
				".gitlab-ci.yml":   "include:\n  - project: acme/ci-templates\n    file: /go.yml\n    ref: v2\n  - component: gitlab.com/acme/c/lint@2.0.0\nbuild:\n  image:\n    name: golang:1.23-bookworm\n    entrypoint: [\"\"]\n  services:\n    - name: \"postgres:17\"\n      alias: db\n  script: go build\ndefault:\n  timeout: 30m\n",
				".gitlab/ci/x.yml": "job:\n  image: golang:1.23\n",
			},
			vars: map[string]string{"img": "[.gitlab-ci.yml .gitlab/ci/x.yml]"},
		},
		"gitlab none": {
			script: `load("@fleet//gitlab.star", "components", "set_default")
def change(repo):
    repo.set_var("d", set_default(repo, "interruptible", True))
    repo.set_var("c", components(repo))
`,
			vars: map[string]string{"d": "false", "c": "[]"},
		},
		"circleci": {
			repo: map[string]string{".circleci/config.yml": "version: 2.1\norbs:\n  node: circleci/node@5.1.0\nexecutors:\n  vm:\n    machine:\n      image: ubuntu-2004:2023.04.2\n    resource_class: \"medium\"\njobs:\n  test:\n    executor: node/default\n    docker:\n      - image: cimg/node:18.20\n  vm:\n    machine: true\n    resource_class: medium  # default\nworkflows:\n  main:\n    jobs: [node/test]\n"},
			script: `load("@fleet//circleci.star", "bump_orb", "orbs", "set_machine_image", "set_resource_class")
def change(repo):
    repo.set_var("orbs", orbs(repo))
    repo.set_var("skip", bump_orb(repo, "circleci/node", "6.0.0", from_ref = "5.0.0"))
    set_machine_image(repo, "ubuntu-2404:2024.08.1", from_image = "ubuntu-2004:2023.04.2")
    set_resource_class(repo, "large")
`,
			want: map[string]string{".circleci/config.yml": "version: 2.1\norbs:\n  node: circleci/node@5.1.0\nexecutors:\n  vm:\n    machine:\n      image: ubuntu-2404:2024.08.1\n    resource_class: \"large\"\njobs:\n  test:\n    executor: node/default\n    docker:\n      - image: cimg/node:18.20\n  vm:\n    machine: true\n    resource_class: large  # default\nworkflows:\n  main:\n    jobs: [node/test]\n"},
			vars: map[string]string{"orbs": "[circleci/node@5.1.0]", "skip": "[]"},
		},
		"jenkins": {
			repo: map[string]string{
				"Jenkinsfile":           "library identifier: 'ci-lib@v2', retriever: modernSCM([$class: 'GitSCMSource', remote: 'https://x/ci-lib.git'])\n@Library('notify') _\npipeline {\n  agent { docker 'registry.acme.dev/build/maven:3.8' }\n  tools { jdk 'jdk11' }\n  stages { stage('x') { steps { sh \"echo jdk 'jdk11'\" } } }\n}\n",
				"ci/deploy.jenkinsfile": "@Library(\"ci-lib@v2\") _\n",
			},
			script: `load("@fleet//jenkins.star", "bump_image", "bump_library", "libraries", "set_tool")
def change(repo):
    repo.set_var("libs", libraries(repo))
    repo.set_var("bumped", bump_library(repo, "ci-lib", "v3"))
    bump_library(repo, "notify", "v9")
    bump_image(repo, "registry.acme.dev/build/maven", "3.8", "3.9")
    set_tool(repo, "jdk", "jdk21", from_name = "jdk11")
`,
			want: map[string]string{
				"Jenkinsfile":           "library identifier: 'ci-lib@v3', retriever: modernSCM([$class: 'GitSCMSource', remote: 'https://x/ci-lib.git'])\n@Library('notify') _\npipeline {\n  agent { docker 'registry.acme.dev/build/maven:3.9' }\n  tools { jdk 'jdk21' }\n  stages { stage('x') { steps { sh \"echo jdk 'jdk11'\" } } }\n}\n",
				"ci/deploy.jenkinsfile": "@Library(\"ci-lib@v3\") _\n",
			},
			vars: map[string]string{"libs": "[ci-lib@v2 notify@]", "bumped": "[Jenkinsfile ci/deploy.jenkinsfile]"},
		},
	} {
		runLib(t, name, tc)
	}
}
