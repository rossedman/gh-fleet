---
title: docker.star
---

# docker.star

Find Dockerfiles, replace base images, and move images to a new tag in Dockerfiles, Compose files and Kubernetes manifests.

```python
load("@fleet//docker.star", "dockerfiles", "bump_base_image", "replace_base_image", "bump_image")
```

| Function | What it does |
| --- | --- |
| [`dockerfiles`](#dockerfiles) | Lists the repo's Dockerfiles. |
| [`bump_base_image`](#bump_base_image) | Moves `FROM` lines of an image from one tag to another in every Dockerfile. |
| [`replace_base_image`](#replace_base_image) | Replaces a whole image reference in the `FROM` lines of every Dockerfile. |
| [`bump_image`](#bump_image) | Moves `image:` lines from one tag to another in YAML files. |

## dockerfiles

```python
dockerfiles(repo)
```

Lists the repo's Dockerfiles.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The sorted paths of every `Dockerfile`, `Dockerfile.*`, `*.Dockerfile`
and `*.dockerfile` outside node_modules.

**Example**

```python
dockerfiles(repo)
# returns ["Dockerfile", "web/app.Dockerfile"]
```

=== "Before"

    `Dockerfile`

    ```dockerfile
    FROM scratch
    ```

    `web/app.Dockerfile`

    ```dockerfile
    FROM node:22
    ```

    `node_modules/x/Dockerfile`

    ```dockerfile
    FROM node:18
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/docker.star#L6)

## bump_base_image

```python
bump_base_image(repo, image, from_tag, to_tag)
```

Moves `FROM` lines of an image from one tag to another in every Dockerfile.

A variant suffix is kept, so with tags `18` and `22`, `node:18-alpine`
becomes `node:22-alpine`, while `node:18.2` is not touched. `--platform`
flags and stage names are kept, and images pinned only by digest
(`node@sha256:...`) never match.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `image` | The image name as written, including any registry. |
| `from_tag` | The tag to replace. |
| `to_tag` | The tag to put in its place. |

**Returns:** The list of paths that changed.

**Example**

```python
bump_base_image(repo, "node", "18", "22")
# returns ["Dockerfile"]
```

=== "Before"

    `Dockerfile`

    ```dockerfile
    FROM node:18-alpine AS build
    RUN npm ci
    FROM node:18.2
    ```

=== "After"

    `Dockerfile`

    ```dockerfile
    FROM node:22-alpine AS build
    RUN npm ci
    FROM node:18.2
    ```

**Used in:** [`compose-image-bump`](../examples/compose-image-bump.md), [`docker-base-image`](../examples/docker-base-image.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/docker.star#L32)

## replace_base_image

```python
replace_base_image(repo, from_image, to_image)
```

Replaces a whole image reference in the `FROM` lines of every Dockerfile.

Use it to move to another image entirely, such as from Docker Hub to
your own registry, where `bump_base_image` only changes the tag.

`from_image` is matched as written. A bare name, such as `python` or
`ghcr.io/acme/base`, matches that image at any tag or digest, so
`python:3.12-slim` and `python@sha256:...` are both replaced; it doesn't
match `python-dev` or `python/tools`. A name with a tag or digest, such
as `python:3.12-slim`, matches only that exact reference. The whole
reference is replaced by `to_image`, and `--platform` flags and stage
names are kept.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `from_image` | The image to replace: a name, or a name with a tag or digest. |
| `to_image` | The full reference to put in its place, such as `registry.acme.io/python:3.14-slim`. |

**Returns:** The list of paths that changed.

**Example**

```python
replace_base_image(repo, "python", "registry.acme.io/library/python/3.14-slim:main-latest")
# returns ["Dockerfile"]
```

=== "Before"

    `Dockerfile`

    ```dockerfile
    FROM --platform=linux/amd64 python:3.12-slim AS build
    RUN pip install -r requirements.txt
    FROM python:3.12-slim
    FROM python-dev:1
    ```

=== "After"

    `Dockerfile`

    ```dockerfile
    FROM --platform=linux/amd64 registry.acme.io/library/python/3.14-slim:main-latest AS build
    RUN pip install -r requirements.txt
    FROM registry.acme.io/library/python/3.14-slim:main-latest
    FROM python-dev:1
    ```

**Used in:** [`docker-replace-image`](../examples/docker-replace-image.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/docker.star#L71)

## bump_image

```python
bump_image(
    repo,
    image,
    from_tag,
    to_tag,
    include = ["**/*.yml", "**/*.yaml"],
    exclude = ["**/node_modules/**"],
)
```

Moves `image:` lines from one tag to another in YAML files.

Covers docker-compose.yml, Kubernetes manifests and anything else that
writes `image: name:tag`. Tags match like `bump_base_image`: a variant
suffix is kept, and `redis:7.2` does not match tag `7`. Quotes are kept.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `image` | The full image name as written, such as `ghcr.io/acme/api`. |
| `from_tag` | The tag to replace. |
| `to_tag` | The tag to put in its place. |
| `include` | Glob patterns of the files to edit. |
| `exclude` | Glob patterns of files to skip. |

**Returns:** The list of paths that changed.

**Example**

```python
bump_image(repo, "redis", "7", "8")
# returns ["compose.yaml"]
```

=== "Before"

    `compose.yaml`

    ```yaml
    services:
      cache:
        image: redis:7-alpine
      pinned:
        image: redis:7.2.4
    ```

=== "After"

    `compose.yaml`

    ```yaml
    services:
      cache:
        image: redis:8-alpine
      pinned:
        image: redis:7.2.4
    ```

**Used in:** [`compose-image-bump`](../examples/compose-image-bump.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/docker.star#L122)
