---
title: trivy-ignore-prune
---

# trivy-ignore-prune

Remove expired entries from .trivyignore so teams see which findings are back.

**Uses:** [`scanignore.prune_expired`](../starlark/scanignore.md#prune_expired)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from trivy-ignore-prune --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//scanignore.star", "prune_expired")

    def change(repo):
        repo.set_var("expired", ", ".join(prune_expired(repo, repo.vars["today"])))
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: trivy-ignore-prune
    description: Remove expired entries from .trivyignore so teams see which findings are back

    targets:
      orgs: [acme]
      include:
        files: [.trivyignore]

    change:
      steps:
        # 1. Today's date, from the shell.
        - name: today
          run: echo "today=$(date -u +%F)" >> "$FLEET_OUTPUT/vars"
          timeout: 1m
        # 2. Starlark removes entries whose exp: date has passed, with comments
        #    that only described them.
        - name: prune
          starlark: ./change.star
          timeout: 1m

    pr:
      title: "chore: remove expired .trivyignore entries"
      body: |
        These `.trivyignore` entries in {{ .Repo.Name }} expired, so Trivy
        reports them again: {{ .Vars.expired }}.

        Fix them, or add them back with a new `exp:` date and a reason.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet, security]
    ```

## Test cases

`gh fleet test` runs the change on each case's `before/` files and checks the result against `after/`. See [Testing a campaign](../../guides/testing.md).

### expired

=== "Before"

    `.trivyignore`

    ```text
    # Waiting on the base image.
    CVE-2024-1234 exp:2020-01-31

    CVE-2025-0001 exp:2999-12-31
    ```

=== "After"

    `.trivyignore`

    ```text
    CVE-2025-0001 exp:2999-12-31
    ```

### nothing-expired

=== "Before"

    `.trivyignore`

    ```text
    CVE-2025-0001 exp:2999-12-31
    ```

=== "After"

    No changes: the campaign leaves this repo alone.

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/trivy-ignore-prune)
