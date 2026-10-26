---
title: kustomize-image
---

# kustomize-image

Roll out api 2.4.0 in every Kustomize overlay.

**Uses:** [`kustomize.set_image`](../starlark/kustomize.md#set_image)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from kustomize-image --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//kustomize.star", "set_image")

    def change(repo):
        changed = set_image(repo, env["IMAGE"], new_tag = env["TAG"])
        repo.set_var("overlays", ", ".join([p.removesuffix("/kustomization.yaml") for p in changed]))
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: api-2-4-0-kustomize
    description: Roll out api 2.4.0 in every Kustomize overlay

    targets:
      orgs: [acme]
      include:
        search: "ghcr.io/acme/api filename:kustomization.yaml"

    change:
      steps:
        # Sets newTag for ghcr.io/acme/api in every kustomization that lists the
        # image, keeping comments. Overlays for other images are left alone.
        - name: kustomize-image
          starlark: ./change.star
          timeout: 1m
      env:
        IMAGE: ghcr.io/acme/api
        TAG: "2.4.0"

    pr:
      title: "deploy: api 2.4.0"
      body: |
        Rolls ghcr.io/acme/api 2.4.0 out to {{ .Vars.overlays }}.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet, deploy]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/kustomize-image)
