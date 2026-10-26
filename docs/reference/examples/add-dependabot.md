---
title: add-dependabot
---

# add-dependabot

Give every service a Dependabot config covering its ecosystems.

**Uses:** [`dependabot.write_if_missing`](../starlark/dependabot.md#write_if_missing)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from add-dependabot --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//dependabot.star", "write_if_missing")

    def change(repo):
        found = write_if_missing(repo, interval = "weekly")
        repo.set_var("ecosystems", ", ".join(found))
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: add-dependabot
    description: Give every service a Dependabot config covering its ecosystems

    targets:
      orgs: [acme]
      include:
        topics: [service]

    change:
      steps:
        # Writes .github/dependabot.yml for the ecosystems it detects. Repos that
        # already have a config are left alone, so they come out as no-ops.
        - name: dependabot config
          starlark: ./change.star
          timeout: 1m

    pr:
      title: "build: add Dependabot config"
      body: |
        Adds `.github/dependabot.yml` to {{ .Repo.FullName }} with weekly,
        grouped updates for every package ecosystem found in the repo.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet, dependencies]
      reviewers: [acme/platform]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/add-dependabot)
