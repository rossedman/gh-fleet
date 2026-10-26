---
title: devcontainer-image
---

# devcontainer-image

Move every Go Dev Container to Go 1.23 and add the GitHub CLI.

**Uses:** [`devcontainer.add_feature`](../starlark/devcontainer.md#add_feature), [`devcontainer.bump_image`](../starlark/devcontainer.md#bump_image), [`devcontainer.files`](../starlark/devcontainer.md#files)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from devcontainer-image --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//devcontainer.star", "add_feature", "bump_image", "files")

    def change(repo):
        bump_image(repo, env["IMAGE"], env["FROM"], env["TO"])
        for path in files(repo):
            add_feature(repo, "ghcr.io/devcontainers/features/github-cli:1", path = path)
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: devcontainer-go-1-23
    description: Move every Go Dev Container to Go 1.23 and add the GitHub CLI

    targets:
      orgs: [acme]
      include:
        files: [.devcontainer]

    change:
      steps:
        # Moves the devcontainers/go image from 1.22 to 1.23 (keeping variants
        # such as -bookworm) and adds the github-cli feature where it is missing,
        # keeping comments in devcontainer.json.
        - name: devcontainer-image
          starlark: ./change.star
          timeout: 1m
      env:
        IMAGE: mcr.microsoft.com/devcontainers/go
        FROM: "1.22"
        TO: "1.23"

    pr:
      title: "chore: Go 1.23 Dev Container"
      body: |
        Moves {{ .Repo.Name }}'s Dev Container to Go 1.23 and adds the GitHub
        CLI to it.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/devcontainer-image)
