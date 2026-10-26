---
title: docker-replace-image
---

# docker-replace-image

Move Dockerfiles from Docker Hub's python image to the internal registry's.

**Uses:** [`docker.replace_base_image`](../starlark/docker.md#replace_base_image)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from docker-replace-image --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//docker.star", "replace_base_image")

    def change(repo):
        changed = replace_base_image(repo, env["FROM_IMAGE"], env["TO_IMAGE"])
        repo.set_var("dockerfiles", ", ".join(changed))
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: internal-python-image
    description: Move Dockerfiles from Docker Hub's python image to the internal registry's

    targets:
      orgs: [acme]
      include:
        files: [Dockerfile]
        search: "FROM python"

    change:
      steps:
        # Replaces the whole image reference in FROM lines: python at any tag or
        # digest becomes TO_IMAGE, keeping --platform flags and stage names.
        # Set FROM_IMAGE to an exact reference, such as python:3.12-slim, to
        # replace only that one.
        - name: base image
          starlark: ./change.star
          timeout: 1m
      env:
        FROM_IMAGE: python
        TO_IMAGE: registry.acme.io/library/python/3.14-slim:main-latest

    pr:
      title: "build: use the internal python base image"
      body: |
        Moves every Dockerfile in {{ .Repo.Name }} from Docker Hub's `python`
        image to `registry.acme.io/library/python/3.14-slim:main-latest`, which
        is scanned and patched by the platform team.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet, security]
    ```

## Test cases

`gh fleet test` runs the change on each case's `before/` files and checks the result against `after/`. See [Testing a campaign](../../guides/testing.md).

### already-moved

=== "Before"

    `Dockerfile`

    ```dockerfile
    FROM registry.acme.io/library/python/3.14-slim:main-latest
    ```

=== "After"

    No changes: the campaign leaves this repo alone.

### hub-python

=== "Before"

    `Dockerfile`

    ```dockerfile
    FROM python:3.12-slim
    WORKDIR /app
    COPY . .
    RUN pip install -r requirements.txt
    CMD ["python", "app.py"]
    ```

=== "After"

    `Dockerfile`

    ```dockerfile
    FROM registry.acme.io/library/python/3.14-slim:main-latest
    WORKDIR /app
    COPY . .
    RUN pip install -r requirements.txt
    CMD ["python", "app.py"]
    ```

### stages

=== "Before"

    `Dockerfile`

    ```dockerfile
    FROM --platform=$BUILDPLATFORM python:3.11 AS build
    RUN pip wheel -w /wheels .

    FROM python@sha256:4f1a5b7c AS runtime
    COPY --from=build /wheels /wheels
    ```

    `tools.Dockerfile`

    ```dockerfile
    FROM python-dev:1
    ```

=== "After"

    `Dockerfile`

    ```dockerfile
    FROM --platform=$BUILDPLATFORM registry.acme.io/library/python/3.14-slim:main-latest AS build
    RUN pip wheel -w /wheels .

    FROM registry.acme.io/library/python/3.14-slim:main-latest AS runtime
    COPY --from=build /wheels /wheels
    ```

    `tools.Dockerfile`

    ```dockerfile
    FROM python-dev:1
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/docker-replace-image)
