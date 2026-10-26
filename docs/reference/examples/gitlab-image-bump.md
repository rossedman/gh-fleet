---
title: gitlab-image-bump
---

# gitlab-image-bump

Move every GitLab CI job and service from Node 18 to Node 22.

**Uses:** [`gitlab.bump_image`](../starlark/gitlab.md#bump_image)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from gitlab-image-bump --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//gitlab.star", "bump_image")

    def change(repo):
        bump_image(repo, env["IMAGE"], env["FROM_TAG"], env["TO_TAG"])
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: gitlab-node-22
    description: Move every GitLab CI job and service from Node 18 to Node 22

    targets:
      orgs: [acme]
      include:
        files: [.gitlab-ci.yml]

    change:
      steps:
        # Rewrites node:18 images in job `image:` keys (plain or `name:` form)
        # and `services`, keeping variants such as -alpine, in .gitlab-ci.yml
        # and the files it includes from .gitlab/.
        - name: gitlab-image-bump
          starlark: ./change.star
          timeout: 1m
      env:
        IMAGE: node
        FROM_TAG: "18"
        TO_TAG: "22"

    pr:
      title: "ci: node 22 in GitLab CI"
      body: |
        Moves {{ .Repo.Name }}'s CI jobs from Node 18, which is out of support,
        to Node 22.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/gitlab-image-bump)
