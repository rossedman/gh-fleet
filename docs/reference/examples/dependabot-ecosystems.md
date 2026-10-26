---
title: dependabot-ecosystems
---

# dependabot-ecosystems

Cover Docker and GitHub Actions in every Dependabot config.

**Uses:** [`dependabot.add_update`](../starlark/dependabot.md#add_update), [`dependabot.ecosystems`](../starlark/dependabot.md#ecosystems), [`dependabot.set_option`](../starlark/dependabot.md#set_option)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from dependabot-ecosystems --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//dependabot.star", "add_update", "ecosystems", "set_option")

    def change(repo):
        added = []
        for eco in ["docker", "github-actions"]:
            if eco in ecosystems(repo) and add_update(repo, eco):
                added.append(eco)
        repo.set_var("added", ", ".join(added) or "nothing new")
        set_option(repo, "open-pull-requests-limit", 5)
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: dependabot-coverage
    description: Cover Docker and GitHub Actions in every Dependabot config

    targets:
      orgs: [acme]
      include:
        files: [.github/dependabot.yml]

    change:
      steps:
        # Adds docker and github-actions entries to existing configs when the repo
        # has a Dockerfile or workflows, and caps open PRs per ecosystem at 5.
        - name: dependabot-ecosystems
          starlark: ./change.star
          timeout: 1m

    pr:
      title: "chore: Dependabot for Docker and Actions"
      body: |
        Adds missing Dependabot coverage to {{ .Repo.Name }}: {{ .Vars.added }}.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet, dependencies]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/dependabot-ecosystems)
