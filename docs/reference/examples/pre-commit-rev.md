---
title: pre-commit-rev
---

# pre-commit-rev

Move pre-commit-hooks to v5 everywhere.

**Uses:** [`precommit.revs`](../starlark/precommit.md#revs), [`precommit.set_rev`](../starlark/precommit.md#set_rev)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from pre-commit-rev --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//precommit.star", "revs", "set_rev")

    def change(repo):
        repo.set_var("previous", revs(repo).get(env["HOOKS"], "unknown"))
        set_rev(repo, env["HOOKS"], env["REV"])
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: bump-pre-commit-hooks
    description: Move pre-commit-hooks to v5 everywhere

    targets:
      orgs: [acme]
      include:
        files: [.pre-commit-config.yaml]

    change:
      steps:
        # Sets the rev of one hook repository in .pre-commit-config.yaml,
        # keeping quotes and comments. Repos that don't use it are no-ops.
        - name: rev
          starlark: ./change.star
          timeout: 1m
      env:
        HOOKS: https://github.com/pre-commit/pre-commit-hooks
        REV: v5.0.0

    pr:
      title: "ci: pre-commit-hooks v5.0.0"
      body: |
        Updates pre-commit-hooks in {{ .Repo.Name }} from {{ .Vars.previous }}
        to v5.0.0.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/pre-commit-rev)
