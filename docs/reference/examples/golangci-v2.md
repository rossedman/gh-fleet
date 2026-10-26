---
title: golangci-v2
---

# golangci-v2

Move golangci-lint configs to the v2 format and enable gosec.

**Uses:** [`golangci.config_version`](../starlark/golangci.md#config_version), [`golangci.enable_linter`](../starlark/golangci.md#enable_linter)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from golangci-v2 --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//golangci.star", "config_version")

    def change(repo):
        repo.set_var("format", config_version(repo) or "none")
    ```

=== "enable.star"

    ```python
    load("@fleet//golangci.star", "enable_linter")

    def change(repo):
        enable_linter(repo, "gosec")
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: golangci-lint-v2
    description: Move golangci-lint configs to the v2 format and enable gosec

    targets:
      orgs: [acme]
      include:
        languages: [Go]
        files: [.golangci.yml]

    change:
      steps:
        # 1. Starlark checks which config format the repo uses.
        - name: check
          starlark: ./change.star
          timeout: 1m
        # 2. The official migrate command rewrites a v1 config, including
        #    renamed and merged linters, and the backup it leaves is removed.
        - name: migrate
          run: |
            if [ "$FLEET_VAR_FORMAT" = 1 ]; then
              go run "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$GOLANGCI" migrate
              rm -f .golangci.bck.*
            fi
          timeout: 5m
        # 3. Starlark enables gosec in the v2 config.
        - name: enable gosec
          starlark: ./enable.star
          timeout: 1m
      env:
        GOLANGCI: v2.15.0

    pr:
      title: "ci: golangci-lint v2 config"
      body: |
        Migrates {{ .Repo.Name }}'s golangci-lint config to the v2 format (it was
        v{{ .Vars.format }}) and enables gosec.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/golangci-v2)
