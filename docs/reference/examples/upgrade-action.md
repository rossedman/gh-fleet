---
title: upgrade-action
---

# upgrade-action

Upgrade actions/checkout from v3 to v4.

**Uses:** [`gha.bump_action`](../starlark/gha.md#bump_action)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from upgrade-action --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//gha.star", "bump_action")

    def change(repo):
        bump_action(repo, env["ACTION"], env["TO"], from_ref = env["FROM"])
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: checkout-v4
    description: Upgrade actions/checkout from v3 to v4

    targets:
      orgs: [acme]
      include:
        files: [.github/workflows]
        search: "actions/checkout@v3 path:.github/workflows"

    change:
      steps:
        # Rewrites `uses: ACTION@FROM` to `ACTION@TO` in workflows and local
        # composite actions. Refs pinned to a SHA are left for pin-actions.
        - name: upgrade action
          starlark: ./change.star
          timeout: 1m
      env:
        ACTION: actions/checkout
        FROM: v3
        TO: v4

    pr:
      title: "ci: actions/checkout v4"
      body: |
        actions/checkout v3 runs on a deprecated Node runtime. This moves every
        workflow in {{ .Repo.Name }} to v4.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet, ci]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/upgrade-action)
