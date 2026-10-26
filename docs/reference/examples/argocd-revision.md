---
title: argocd-revision
---

# argocd-revision

Point every Argo CD Application at the platform manifests v3.0.0.

**Uses:** [`argocd.set_target_revision`](../starlark/argocd.md#set_target_revision)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from argocd-revision --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//argocd.star", "set_target_revision")

    def change(repo):
        set_target_revision(repo, env["REPO_URL"], env["REVISION"])
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: platform-apps-v3
    description: Point every Argo CD Application at the platform manifests v3.0.0

    targets:
      orgs: [acme]
      include:
        search: "github.com/acme/platform-manifests argoproj.io"

    change:
      steps:
        # Sets targetRevision for every source whose repoURL is the platform
        # manifests repo, in `source` and in each entry of `sources`.
        - name: argocd-revision
          starlark: ./change.star
          timeout: 1m
      env:
        REPO_URL: https://github.com/acme/platform-manifests.git
        REVISION: v3.0.0

    pr:
      title: "deploy: platform manifests v3.0.0"
      body: |
        Moves {{ .Repo.Name }}'s Argo CD Applications to platform-manifests v3.0.0.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet, deploy]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/argocd-revision)
