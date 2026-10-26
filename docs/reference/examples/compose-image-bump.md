---
title: compose-image-bump
---

# compose-image-bump

Move Redis to 8 in Compose files and Kubernetes manifests.

**Uses:** [`docker.bump_base_image`](../starlark/docker.md#bump_base_image), [`docker.bump_image`](../starlark/docker.md#bump_image)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from compose-image-bump --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//docker.star", "bump_base_image", "bump_image")

    def change(repo):
        changed = bump_image(repo, env["IMAGE"], env["FROM_TAG"], env["TO_TAG"])
        changed += bump_base_image(repo, env["IMAGE"], env["FROM_TAG"], env["TO_TAG"])
        repo.set_var("files", ", ".join(changed))
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: redis-8
    description: Move Redis to 8 in Compose files and Kubernetes manifests

    targets:
      orgs: [acme]
      include:
        search: "image: redis:7"

    change:
      steps:
        # Rewrites `image: redis:7...` in every YAML file, keeping variants
        # such as -alpine. redis:7.2 and other exact tags are left alone.
        - name: image
          starlark: ./change.star
          timeout: 1m
      env:
        IMAGE: redis
        FROM_TAG: "7"
        TO_TAG: "8"

    pr:
      title: "build: redis 8"
      body: |
        Moves Redis from 7 to 8 in {{ .Vars.files }}.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      draft: true
      labels: [fleet]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/compose-image-bump)
