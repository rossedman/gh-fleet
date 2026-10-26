---
title: renovate-automerge
---

# renovate-automerge

Automerge devDependency updates and move off config:base.

**Uses:** [`renovate.add_package_rule`](../starlark/renovate.md#add_package_rule), [`renovate.replace_preset`](../starlark/renovate.md#replace_preset)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from renovate-automerge --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//renovate.star", "add_package_rule", "replace_preset")

    def change(repo):
        replace_preset(repo, "config:base", "config:recommended")
        add_package_rule(repo, {
            "matchDepTypes": ["devDependencies"],
            "matchUpdateTypes": ["minor", "patch"],
            "automerge": True,
        })
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: renovate-automerge-dev
    description: Automerge devDependency updates and move off config:base

    targets:
      orgs: [acme]
      include:
        files: [renovate.json]

    change:
      steps:
        # Replaces the deprecated config:base preset with config:recommended and
        # adds a rule that automerges minor and patch devDependency updates,
        # unless the repo already has that exact rule.
        - name: renovate-automerge
          starlark: ./change.star
          timeout: 1m

    pr:
      title: "chore: automerge devDependency updates"
      body: |
        Lets Renovate automerge minor and patch devDependency updates in
        {{ .Repo.Name }}, and moves the config off the deprecated `config:base`.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet, dependencies]
    ```

## Test cases

`gh fleet test` runs the change on each case's `before/` files and checks the result against `after/`. See [Testing a campaign](../../guides/testing.md).

### inline-lists

=== "Before"

    `.github/renovate.json`

    ```json
    {
      "$schema": "https://docs.renovatebot.com/renovate-schema.json",
      "extends": ["config:base", ":pinDependencies"],
      "packageRules": [
        {
          "description": "Use build semantic type for some deps",
          "matchPackageNames": ["@vercel/ncc", "typescript"],
          "semanticCommitType": "build"
        }
      ]
    }
    ```

=== "After"

    `.github/renovate.json`

    ```json
    {
      "$schema": "https://docs.renovatebot.com/renovate-schema.json",
      "extends": ["config:recommended", ":pinDependencies"],
      "packageRules": [
        {
          "description": "Use build semantic type for some deps",
          "matchPackageNames": ["@vercel/ncc", "typescript"],
          "semanticCommitType": "build"
        },
        {
          "matchDepTypes": ["devDependencies"],
          "matchUpdateTypes": ["minor", "patch"],
          "automerge": true
        }
      ]
    }
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/renovate-automerge)
