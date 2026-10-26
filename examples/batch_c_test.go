package examples_test

import "testing"

func TestBatchCExamples(t *testing.T) {
	t.Parallel()
	dir := scratch(t, map[string]string{".prettierrc.json": "{\n  \"semi\": false\n}\n", ".eslintrc.json": "{\n  \"rules\": {}\n}\n"})
	vars := runTwice(t, "prettier-eslint", dir)
	if vars["eslint"] != "" {
		t.Errorf("eslint = %v", vars["eslint"])
	}
	contains(t, dir, ".prettierrc.json", `"printWidth": 100`, `"trailingComma": "all"`)
	contains(t, dir, ".eslintrc.json", `"eqeqeq": "error"`, `"no-console": "error"`)
	flat := scratch(t, map[string]string{"package.json": "{\"prettier\": {}}", "eslint.config.js": "export default []\n"})
	vars = runTwice(t, "prettier-eslint", flat)
	if vars["eslint"] != "ESLint uses a flat config (eslint.config.js); apply the rules there by hand." {
		t.Errorf("eslint = %v", vars["eslint"])
	}

	dir = scratch(t, map[string]string{".markdownlint.yaml": "default: true\n", ".yamllint": "rules:\n  line-length:\n    max: 80\n", "Dockerfile": "FROM debian\n"})
	runTwice(t, "lint-configs", dir)
	contains(t, dir, ".markdownlint.yaml", "MD013: false")
	contains(t, dir, ".yamllint", "max: 120")
	contains(t, dir, ".hadolint.yaml", "  - DL3008\n  - DL3013\n")

	dir = scratch(t, map[string]string{"azure-pipelines.yml": "pool:\n  vmImage: ubuntu-20.04\nsteps:\n  - task: Docker@1\n"})
	runTwice(t, "azure-ubuntu-24", dir)
	contains(t, dir, "azure-pipelines.yml", "vmImage: ubuntu-24.04", "task: Docker@2")

	dir = scratch(t, map[string]string{"bitbucket-pipelines.yml": "image: node:18\npipelines:\n  default:\n    - step:\n        script:\n          - pipe: atlassian/aws-s3-deploy:1.1.0\n"})
	vars = runTwice(t, "bitbucket-pipes", dir)
	contains(t, dir, "bitbucket-pipelines.yml", "image: node:22", "aws-s3-deploy:1.6.1")
	if vars["previous"] != "1.1.0" {
		t.Errorf("previous = %v", vars["previous"])
	}

	dir = scratch(t, map[string]string{".woodpecker/test.yml": "steps:\n  test:\n    image: golang:1.22-alpine\n"})
	runTwice(t, "drone-golang", dir)
	contains(t, dir, ".woodpecker/test.yml", "image: golang:1.23-alpine")

	dir = scratch(t, map[string]string{"helmfile.d/cache.yaml": "releases:\n  - name: cache\n    chart: bitnami/redis\n    version: 18.6.1\n"})
	runTwice(t, "helmfile-chart", dir)
	contains(t, dir, "helmfile.d/cache.yaml", "version: 20.2.1")

	dir = scratch(t, map[string]string{"serverless.yml": "provider:\n  runtime: nodejs16.x\nfunctions:\n  py:\n    runtime: python3.12\n"})
	runTwice(t, "serverless-runtime", dir)
	contains(t, dir, "serverless.yml", "runtime: nodejs20.x", "runtime: python3.12")

	dir = scratch(t, map[string]string{"cdk.json": "{\n  \"app\": \"npx ts-node bin/app.ts\"\n}\n"})
	runTwice(t, "cdk-feature-flags", dir)
	contains(t, dir, "cdk.json", `"@aws-cdk/aws-s3:createDefaultLoggingPolicy": true`)

	dir = scratch(t, map[string]string{"build.pkr.hcl": "packer {\n  required_plugins {\n    amazon = {\n      version = \">= 1.0.0\"\n      source  = \"github.com/hashicorp/amazon\"\n    }\n  }\n}\n"})
	runTwice(t, "packer-plugins", dir)
	contains(t, dir, "build.pkr.hcl", `version = ">= 1.3.3"`)
}
