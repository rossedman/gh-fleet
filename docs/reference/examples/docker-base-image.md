---
title: docker-base-image
---

# docker-base-image

Move Docker images from Node 18 to Node 22.

**Uses:** [`docker.bump_base_image`](../starlark/docker.md#bump_base_image)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from docker-base-image --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//docker.star", "bump_base_image")

    def change(repo):
        changed = bump_base_image(repo, env["IMAGE"], env["FROM_TAG"], env["TO_TAG"])
        repo.set_var("dockerfiles", ", ".join(changed))
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: node-22-images
    description: Move Docker images from Node 18 to Node 22

    targets:
      orgs: [acme]
      include:
        files: [Dockerfile]
        search: "FROM node:18"

    change:
      steps:
        # Rewrites FROM lines for IMAGE:FROM_TAG to IMAGE:TO_TAG in every
        # Dockerfile, keeping variants (node:18-alpine -> node:22-alpine), stage
        # names and --platform flags.
        - name: base image
          starlark: ./change.star
          timeout: 1m
      env:
        IMAGE: node
        FROM_TAG: "18"
        TO_TAG: "22"

    pr:
      title: "build: Node 22 base images"
      body: |
        Node 18 is end of life. This moves every `FROM node:18` image in
        {{ .Repo.Name }} to Node 22, keeping the same variant. Check the build
        and any native modules.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet, dependencies]
      draft: true
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/docker-base-image)
