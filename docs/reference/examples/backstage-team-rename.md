---
title: backstage-team-rename
---

# backstage-team-rename

Move catalog entities from the payments team to billing after the reorg.

**Uses:** [`backstage.entities`](../starlark/backstage.md#entities), [`backstage.set_owner`](../starlark/backstage.md#set_owner)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from backstage-team-rename --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//backstage.star", "entities", "set_owner")

    def change(repo):
        before = {(e["path"], e["name"]): e["owner"] for e in entities(repo)}
        set_owner(repo, env["TO"], from_owner = env["FROM"])
        moved = [e["name"] for e in entities(repo) if before.get((e["path"], e["name"])) != e["owner"]]
        repo.set_var("entities", ", ".join(moved))
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: backstage-team-rename
    description: Move catalog entities from the payments team to billing after the reorg

    targets:
      orgs: [acme]
      include:
        search: "payments filename:catalog-info"

    change:
      steps:
        # Matches `payments`, `group:payments` and `group:default/payments`,
        # and leaves entities other teams own alone.
        - name: rename
          starlark: ./change.star
          timeout: 1m
      env:
        FROM: payments
        TO: group:billing

    pr:
      title: "chore: billing owns what payments owned"
      body: |
        The payments team is now billing. This moves the owner of
        {{ .Vars.entities }} in {{ .Repo.Name }}'s Backstage catalog.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/backstage-team-rename)
