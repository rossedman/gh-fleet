---
title: helmfile-chart
---

# helmfile-chart

Move every Helmfile release of bitnami/redis to chart 20.x.

**Uses:** [`helmfile.set_chart_version`](../starlark/helmfile.md#set_chart_version)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from helmfile-chart --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//helmfile.star", "set_chart_version")

    def change(repo):
        set_chart_version(repo, env["CHART"], env["VERSION"])
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: helmfile-redis-20
    description: Move every Helmfile release of bitnami/redis to chart 20.x

    targets:
      orgs: [acme]
      include:
        search: "bitnami/redis filename:helmfile"

    change:
      steps:
        # Sets the version of every release that installs bitnami/redis, in
        # helmfile.yaml and helmfile.d.
        - name: helmfile-chart
          starlark: ./change.star
          timeout: 1m
      env:
        CHART: bitnami/redis
        VERSION: "20.2.1"

    pr:
      title: "deploy: bitnami/redis chart 20.2.1"
      body: |
        Moves {{ .Repo.Name }}'s Redis releases to chart 20.2.1. Check the
        chart's upgrade notes before merging.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet, deploy]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/helmfile-chart)
