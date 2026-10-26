---
title: changelog-entry
---

# changelog-entry

Patch lodash and record it in each repo's changelog.

**Uses:** [`changelog.add_entry`](../starlark/changelog.md#add_entry), [`npm.set_dependency`](../starlark/npm.md#set_dependency)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from changelog-entry --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//changelog.star", "add_entry")
    load("@fleet//npm.star", "set_dependency")

    def change(repo):
        if set_dependency(repo, env["PACKAGE"], env["RANGE"]):
            add_entry(repo, "Update %s to %s." % (env["PACKAGE"], env["RANGE"]), section = "Security")
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: bump-lodash-with-changelog
    description: Patch lodash and record it in each repo's changelog

    targets:
      orgs: [acme]
      include:
        search: "lodash filename:package.json"

    change:
      steps:
        # Bumps lodash where it is listed, then adds an entry under Unreleased >
        # Security in CHANGELOG.md, creating the Unreleased section if needed.
        # Repos without a changelog just get the bump.
        - name: changelog-entry
          starlark: ./change.star
          timeout: 1m
      env:
        PACKAGE: lodash
        RANGE: "^4.17.21"

    pr:
      title: "build(deps): lodash 4.17.21"
      body: |
        Updates lodash in {{ .Repo.Name }} to fix a prototype pollution issue,
        and records it in the changelog.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet, dependencies, security]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/changelog-entry)
