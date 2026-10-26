---
title: drone-golang
---

# drone-golang

Move Drone and Woodpecker Go pipelines to Go 1.23.

**Uses:** [`drone.bump_image`](../starlark/drone.md#bump_image)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from drone-golang --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//drone.star", "bump_image")

    def change(repo):
        bump_image(repo, "golang", env["FROM"], env["TO"])
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: drone-go-1-23
    description: Move Drone and Woodpecker Go pipelines to Go 1.23

    targets:
      orgs: [acme]
      include:
        languages: [Go]

    change:
      steps:
        # Moves golang:1.22 step images to 1.23 (keeping -alpine and similar
        # variants) in .drone.yml and every Woodpecker pipeline.
        - name: drone-golang
          starlark: ./change.star
          timeout: 1m
      env:
        FROM: "1.22"
        TO: "1.23"

    pr:
      title: "ci: go 1.23 in CI"
      body: |
        Moves {{ .Repo.Name }}'s CI steps to Go 1.23.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/drone-golang)
