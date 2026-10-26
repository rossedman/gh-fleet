---
title: azure-ubuntu-24
---

# azure-ubuntu-24

Move Azure Pipelines off the retired Ubuntu 20.04 image.

**Uses:** [`azure_pipelines.bump_task`](../starlark/azure_pipelines.md#bump_task), [`azure_pipelines.set_vm_image`](../starlark/azure_pipelines.md#set_vm_image)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from azure-ubuntu-24 --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//azure_pipelines.star", "bump_task", "set_vm_image")

    def change(repo):
        set_vm_image(repo, env["TO"], from_image = env["FROM"])
        bump_task(repo, "Docker", 2, from_major = 1)
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: azure-ubuntu-24-04
    description: Move Azure Pipelines off the retired Ubuntu 20.04 image

    targets:
      orgs: [acme]
      include:
        files: [azure-pipelines.yml]

    change:
      steps:
        # Moves vmImage ubuntu-20.04 to ubuntu-24.04 (block and flow pool forms),
        # and Docker@1 tasks to Docker@2. Other images and tasks stay.
        - name: azure-ubuntu-24
          starlark: ./change.star
          timeout: 1m
      env:
        FROM: ubuntu-20.04
        TO: ubuntu-24.04

    pr:
      title: "ci: ubuntu-24.04 agents"
      body: |
        Moves {{ .Repo.Name }}'s pipelines from the retired ubuntu-20.04 hosted
        image to ubuntu-24.04.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/azure-ubuntu-24)
