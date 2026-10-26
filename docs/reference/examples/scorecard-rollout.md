---
title: scorecard-rollout
---

# scorecard-rollout

Run OpenSSF Scorecard on open source repos and show the badge.

**Uses:** [`scorecard.add_badge`](../starlark/scorecard.md#add_badge), [`scorecard.add_workflow`](../starlark/scorecard.md#add_workflow)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from scorecard-rollout --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//scorecard.star", "add_badge", "add_workflow")

    def change(repo):
        add_workflow(repo, publish = True)
        add_badge(repo)
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: openssf-scorecard
    description: Run OpenSSF Scorecard on open source repos and show the badge

    targets:
      orgs: [acme]
      # Published results, which the badge needs, only work on public repos.
      include:
        topics: [open-source]
      exclude:
        topics: [no-fleet]

    change:
      steps:
        # Adds the Scorecard workflow (publishing results, which the badge
        # needs) and the badge under the README's title.
        - name: scorecard
          starlark: ./change.star
          timeout: 1m

    pr:
      title: "ci: OpenSSF Scorecard"
      body: |
        Adds the [OpenSSF Scorecard](https://scorecard.dev) workflow, which
        checks {{ .Repo.Name }} against supply-chain security practices every
        week and on pushes to the default branch. Findings show up under
        Security → Code scanning, and the README gets the Scorecard badge.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet, security]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/scorecard-rollout)
