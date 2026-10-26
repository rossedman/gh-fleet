package examples_test

import (
	"strings"
	"testing"
)

func TestGitLabExamples(t *testing.T) {
	t.Parallel()
	dir := scratch(t, map[string]string{
		".gitlab-ci.yml":      "include:\n  - project: acme/ci-templates\n    ref: v1.9.0\n    file: /node.yml\n  - component: gitlab.com/acme/components/test@1.4.0\ndefault:\n  image: node:18-alpine\ntest:\n  services:\n    - name: node:18\n  script: npm test\n",
		".gitlab/ci/lint.yml": "lint:\n  image:\n    name: node:18\n",
	})
	runTwice(t, "gitlab-image-bump", dir)
	contains(t, dir, ".gitlab-ci.yml", "image: node:22-alpine", "- name: node:22")
	contains(t, dir, ".gitlab/ci/lint.yml", "name: node:22")
	runTwice(t, "gitlab-templates", dir)
	contains(t, dir, ".gitlab-ci.yml", "ref: v2.0.0", "components/test@2.0.0", "default:\n  image: node:22-alpine\n  interruptible: true\n")
}

func TestCircleCIExamples(t *testing.T) {
	t.Parallel()
	config := "version: 2.1\norbs:\n  node: circleci/node@5.0.3\njobs:\n  it:\n    machine:\n      image: ubuntu-2004:202201-02\n  unit:\n    docker:\n      - image: cimg/base:2022.06\n"
	dir := scratch(t, map[string]string{".circleci/config.yml": config})
	vars := runTwice(t, "circleci-orbs", dir)
	contains(t, dir, ".circleci/config.yml", "circleci/node@5.2.0")
	if vars["previous"] != "5.0.3" {
		t.Errorf("previous = %v", vars["previous"])
	}
	runTwice(t, "circleci-machine-image", dir)
	contains(t, dir, ".circleci/config.yml", "image: ubuntu-2404:2024.08.1", "image: cimg/base:2024.01")
}

func TestJenkinsExamples(t *testing.T) {
	t.Parallel()
	jf := "@Library('ci-lib@v2.4') _\npipeline {\n  agent { docker { image 'maven:3.8-eclipse-temurin-17' } }\n  tools {\n    jdk 'jdk17'\n  }\n}\n"
	dir := scratch(t, map[string]string{"Jenkinsfile": jf})
	runTwice(t, "jenkins-agent-image", dir)
	contains(t, dir, "Jenkinsfile", "image 'maven:3.9-eclipse-temurin-17'", "    jdk 'jdk21'\n")
	if got := read(t, dir+"/Jenkinsfile"); strings.Contains(got, "3.8") {
		t.Errorf("Jenkinsfile still has 3.8:\n%s", got)
	}
}
