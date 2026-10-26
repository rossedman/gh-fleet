---
title: circleci-machine-image
---

# circleci-machine-image

Move CircleCI machine jobs off retired Ubuntu 20.04 images.

**Uses:** [`circleci.bump_image`](../starlark/circleci.md#bump_image), [`circleci.set_machine_image`](../starlark/circleci.md#set_machine_image)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from circleci-machine-image --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//circleci.star", "bump_image", "set_machine_image")

    def change(repo):
        set_machine_image(repo, env["MACHINE"], from_image = "ubuntu-2004:*")
        bump_image(repo, "cimg/base", "2022.06", "2024.01")
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: circleci-ubuntu-2404
    description: Move CircleCI machine jobs off retired Ubuntu 20.04 images

    targets:
      orgs: [acme]
      include:
        search: "ubuntu-2004 path:.circleci"

    change:
      steps:
        # Sets every machine executor on an ubuntu-2004 image, whatever its date
        # tag, to the Ubuntu 24.04 image, and moves Docker jobs from the cimg/base
        # 2022 images to 2024. Docker executor images are not machine images, so
        # the first call leaves them alone.
        - name: circleci-machine-image
          starlark: ./change.star
          timeout: 1m
      env:
        MACHINE: ubuntu-2404:2024.08.1

    pr:
      title: "ci: Ubuntu 24.04 machine images"
      body: |
        Moves {{ .Repo.Name }}'s machine jobs off the retired Ubuntu 20.04
        images.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/circleci-machine-image)
