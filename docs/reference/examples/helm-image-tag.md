---
title: helm-image-tag
---

# helm-image-tag

Roll out api 2.4.0 to every chart that deploys it.

**Uses:** [`helm.charts`](../starlark/helm.md#charts), [`helm.set_app_version`](../starlark/helm.md#set_app_version), [`helm.set_chart_version`](../starlark/helm.md#set_chart_version), [`helm.set_image_tag`](../starlark/helm.md#set_image_tag), [`yamlpath.get`](../starlark/yamlpath.md#get)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from helm-image-tag --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//helm.star", "charts", "set_app_version", "set_chart_version", "set_image_tag")
    load("@fleet//yamlpath.star", "get")

    def change(repo):
        changed = set_image_tag(repo, env["IMAGE"], env["TAG"])
        for chart in charts(repo):
            if not [p for p in changed if p.startswith(chart + "/") or chart == "."]:
                continue
            set_app_version(repo, chart, env["TAG"])
            major, minor, patch = get(repo, chart + "/Chart.yaml", "version").split(".")
            set_chart_version(repo, chart, "%s.%s.%d" % (major, minor, int(patch) + 1))
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: api-2-4-0
    description: Roll out api 2.4.0 to every chart that deploys it

    targets:
      orgs: [acme]
      include:
        files: [charts]

    change:
      steps:
        # Sets the image tag next to `repository: ghcr.io/acme/api` in every
        # values file, and for each chart that changed, its appVersion and a
        # patch bump of its chart version.
        - name: helm
          starlark: ./change.star
          timeout: 1m
      env:
        IMAGE: ghcr.io/acme/api
        TAG: "2.4.0"

    pr:
      title: "deploy: api 2.4.0"
      body: |
        Deploys ghcr.io/acme/api 2.4.0 from {{ .Repo.Name }}'s charts.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet, deploy]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/helm-image-tag)
