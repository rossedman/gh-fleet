---
title: make-go-version
---

# make-go-version

Keep the Go version in Makefiles in step with go.mod.

**Uses:** [`gomod.set_go_version`](../starlark/gomod.md#set_go_version), [`make.set_var`](../starlark/make.md#set_var)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from make-go-version --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//gomod.star", "set_go_version")
    load("@fleet//make.star", "set_var")

    def change(repo):
        set_var(repo, "GO_VERSION", env["GO_VERSION"])
        set_go_version(repo, env["GO_VERSION"])
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: makefile-go-1-27
    description: Keep the Go version in Makefiles in step with go.mod

    targets:
      orgs: [acme]
      include:
        languages: [Go]
        files: [Makefile]

    change:
      steps:
        # Sets GO_VERSION wherever a Makefile assigns it, keeping the operator
        # (?=, :=, =) and comments, and raises go.mod to match.
        - name: make-go-version
          starlark: ./change.star
          timeout: 1m
      env:
        GO_VERSION: "1.27.3"

    pr:
      title: "build: go 1.27.3 in Makefile"
      body: |
        Sets `GO_VERSION` to 1.27.3 in {{ .Repo.Name }}'s Makefiles and go.mod.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/make-go-version)
