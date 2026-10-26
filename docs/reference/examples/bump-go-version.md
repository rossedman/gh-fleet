---
title: bump-go-version
---

# bump-go-version

Move every Go module to Go 1.27.

**Uses:** [`gha.set_input`](../starlark/gha.md#set_input), [`gomod.set_go_version`](../starlark/gomod.md#set_go_version)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from bump-go-version --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//gha.star", "set_input")
    load("@fleet//gomod.star", "set_go_version")

    def change(repo):
        version = env["GO_VERSION"]
        set_go_version(repo, version)
        # Only numeric pins: leave `go-version: stable` and matrix expressions alone.
        set_input(repo, "go-version", version, match = r"^\d")
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: bump-go-1-27
    description: Move every Go module to Go 1.27

    targets:
      orgs: [acme]
      include:
        languages: [Go]
        files: [go.mod]
      exclude:
        repos: [acme/legacy-api]

    change:
      steps:
        # Raises the go directive (never lowers it), drops a toolchain line the
        # new version covers, and bumps go-version in workflows that pin one.
        - name: go version
          starlark: ./change.star
          timeout: 2m
      env:
        GO_VERSION: "1.27.3"

    pr:
      title: "build: go 1.27"
      body: |
        Updates {{ .Repo.Name }} to Go 1.27: the `go` directive in `go.mod`, and
        any `go-version:` pinned in GitHub Actions workflows.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      draft: true
      labels: [fleet]

    limits:
      concurrency: 16
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/bump-go-version)
