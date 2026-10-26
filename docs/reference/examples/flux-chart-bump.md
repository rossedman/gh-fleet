---
title: flux-chart-bump
---

# flux-chart-bump

Move every Flux HelmRelease of ingress-nginx to chart 4.11.x.

**Uses:** [`flux.set_chart_version`](../starlark/flux.md#set_chart_version)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from flux-chart-bump --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//flux.star", "set_chart_version")

    def change(repo):
        set_chart_version(repo, env["CHART"], env["VERSION"])
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: ingress-nginx-4-11
    description: Move every Flux HelmRelease of ingress-nginx to chart 4.11.x

    targets:
      orgs: [acme]
      include:
        search: "ingress-nginx helm.toolkit.fluxcd.io"

    change:
      steps:
        # Sets the chart version of every HelmRelease that installs the chart,
        # keeping quotes and comments.
        - name: flux-chart-bump
          starlark: ./change.star
          timeout: 1m
      env:
        CHART: ingress-nginx
        VERSION: "4.11.x"

    pr:
      title: "deploy: ingress-nginx chart 4.11.x"
      body: |
        Moves ingress-nginx to chart 4.11.x in {{ .Repo.Name }}.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet, deploy]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/flux-chart-bump)
