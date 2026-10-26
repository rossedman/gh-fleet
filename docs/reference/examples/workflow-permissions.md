---
title: workflow-permissions
---

# workflow-permissions

Default every workflow's token to read-only.

**Uses:** [`gha.ensure_permissions`](../starlark/gha.md#ensure_permissions)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from workflow-permissions --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//gha.star", "ensure_permissions")

    def change(repo):
        ensure_permissions(repo, {"contents": "read"})
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: workflow-permissions
    description: Default every workflow's token to read-only

    targets:
      orgs: [acme]
      include:
        files: [.github/workflows]

    change:
      steps:
        # Adds a top-level `permissions: contents: read` to each workflow that
        # has no top-level permissions block. Jobs that need more must ask.
        - name: permissions
          starlark: ./change.star
          timeout: 1m

    pr:
      title: "ci: read-only GITHUB_TOKEN by default"
      body: |
        Workflows in {{ .Repo.Name }} without a `permissions:` block get the
        repository's default token permissions, which can include write access.
        This sets them to `contents: read`. If a job needs more (pushing tags,
        commenting on PRs), give that job its own `permissions:`.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet, security, ci]
      draft: true
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/workflow-permissions)
