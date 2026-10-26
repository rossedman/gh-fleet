---
title: devcontainer.star
---

# devcontainer.star

Edit Dev Container configs: images, feature versions and top-level settings.

devcontainer.json is JSON with comments, so edits are line-based and keep
comments, trailing commas and layout. Reading strips the comments first.

```python
load("@fleet//devcontainer.star", "files", "read", "bump_image", "features", "set_feature_version", "add_feature", "set")
```

| Function | What it does |
| --- | --- |
| [`files`](#files) | Lists the repo's Dev Container configs. |
| [`read`](#read) | Reads a Dev Container config, ignoring comments and trailing commas. |
| [`bump_image`](#bump_image) | Moves the config's `image` from one tag to another. |
| [`features`](#features) | Lists the features a config installs, with their versions. |
| [`set_feature_version`](#set_feature_version) | Moves a feature to a new major version, such as `docker-in-docker:1` to `:2`. |
| [`add_feature`](#add_feature) | Adds a feature to a config that doesn't have it yet, in any version. |
| [`set`](#set) | Sets a top-level setting, such as `postCreateCommand` or `remoteUser`. |

## files

```python
files(repo)
```

Lists the repo's Dev Container configs.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The sorted paths of .devcontainer/devcontainer.json,
.devcontainer.json and .devcontainer/<name>/devcontainer.json.

**Example**

```python
files(repo)
# returns [".devcontainer/devcontainer.json", ".devcontainer/python/devcontainer.json"]
```

=== "Before"

    `.devcontainer/devcontainer.json`

    ```json
    {}
    ```

    `.devcontainer/python/devcontainer.json`

    ```json
    {}
    ```

**Used in:** [`devcontainer-image`](../examples/devcontainer-image.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/devcontainer.star#L12)

## read

```python
read(repo, path = ".devcontainer/devcontainer.json")
```

Reads a Dev Container config, ignoring comments and trailing commas.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `path` | The config file. |

**Returns:** The decoded config, or None when the file does not exist.

**Example**

```python
read(repo)["image"]
# returns "mcr.microsoft.com/devcontainers/go:1.22"
```

=== "Before"

    `.devcontainer/devcontainer.json`

    ```json
    {
      // The Go image
      "image": "mcr.microsoft.com/devcontainers/go:1.22",
      "features": {
        "ghcr.io/devcontainers/features/node:1": {},
      },
    }
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/devcontainer.star#L65)

## bump_image

```python
bump_image(repo, image, from_tag, to_tag)
```

Moves the config's `image` from one tag to another.

A variant suffix is kept, so `go:1.22-bookworm` becomes
`go:1.23-bookworm`.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `image` | The image name, such as `mcr.microsoft.com/devcontainers/go`. |
| `from_tag` | The tag to replace. |
| `to_tag` | The tag to put in its place. |

**Returns:** The list of paths that changed.

**Example**

```python
bump_image(repo, "mcr.microsoft.com/devcontainers/go", "1.22", "1.23")
# returns [".devcontainer/devcontainer.json"]
```

=== "Before"

    `.devcontainer/devcontainer.json`

    ```json
    {
      "name": "api",
      "image": "mcr.microsoft.com/devcontainers/go:1.22-bookworm", // base
    }
    ```

=== "After"

    `.devcontainer/devcontainer.json`

    ```json
    {
      "name": "api",
      "image": "mcr.microsoft.com/devcontainers/go:1.23-bookworm", // base
    }
    ```

**Used in:** [`devcontainer-image`](../examples/devcontainer-image.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/devcontainer.star#L91)

## features

```python
features(repo, path = ".devcontainer/devcontainer.json")
```

Lists the features a config installs, with their versions.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `path` | The config file. |

**Returns:** A dict of feature ID, without its version, to the version (the part
after the last `:`), in file order.

**Example**

```python
features(repo)
# returns {"ghcr.io/devcontainers/features/node": "1", "ghcr.io/devcontainers/features/docker-in-docker": "2"}
```

=== "Before"

    `.devcontainer/devcontainer.json`

    ```json
    {
      "features": {
        "ghcr.io/devcontainers/features/node:1": {"version": "22"},
        "ghcr.io/devcontainers/features/docker-in-docker:2": {}
      }
    }
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/devcontainer.star#L129)

## set_feature_version

```python
set_feature_version(repo, feature, version)
```

Moves a feature to a new major version, such as `docker-in-docker:1` to `:2`.

A newer version already there is kept, so a campaign never lowers one.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `feature` | The feature ID without its version. |
| `version` | The new version, as written after `:`. |

**Returns:** The list of paths that changed.

**Example**

```python
set_feature_version(repo, "ghcr.io/devcontainers/features/docker-in-docker", "2")
# returns [".devcontainer/devcontainer.json"]
```

=== "Before"

    `.devcontainer/devcontainer.json`

    ```json
    {
      "features": {
        "ghcr.io/devcontainers/features/docker-in-docker:1": {} // dind
      }
    }
    ```

=== "After"

    `.devcontainer/devcontainer.json`

    ```json
    {
      "features": {
        "ghcr.io/devcontainers/features/docker-in-docker:2": {} // dind
      }
    }
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/devcontainer.star#L159)

## add_feature

```python
add_feature(
    repo,
    feature,
    options = {},
    path = ".devcontainer/devcontainer.json",
)
```

Adds a feature to a config that doesn't have it yet, in any version.

The feature goes first in `features`, which is created if missing.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `feature` | The feature ID with its version, such as `ghcr.io/devcontainers/features/github-cli:1`. |
| `options` | The feature's options. |
| `path` | The config file. |

**Returns:** True when the file changed.

**Example**

```python
add_feature(repo, "ghcr.io/devcontainers/features/github-cli:1")
# returns True
```

=== "Before"

    `.devcontainer/devcontainer.json`

    ```json
    {
      "image": "mcr.microsoft.com/devcontainers/base:ubuntu",
      "features": {
        "ghcr.io/devcontainers/features/node:1": {}
      }
    }
    ```

=== "After"

    `.devcontainer/devcontainer.json`

    ```json
    {
      "image": "mcr.microsoft.com/devcontainers/base:ubuntu",
      "features": {
        "ghcr.io/devcontainers/features/github-cli:1": {},
        "ghcr.io/devcontainers/features/node:1": {}
      }
    }
    ```

**Used in:** [`devcontainer-image`](../examples/devcontainer-image.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/devcontainer.star#L200)

## set

```python
set(repo, key, value, path = ".devcontainer/devcontainer.json")
```

Sets a top-level setting, such as `postCreateCommand` or `remoteUser`.

An existing single-line value is replaced, keeping its trailing comma
and comment; a missing key is added at the top. Values spanning
several lines can't be replaced and fail the run.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `key` | The top-level key. |
| `value` | A string, number, boolean, list or dict, written on one line. |
| `path` | The config file. |

**Returns:** True when the file changed.

**Example**

```python
set(repo, "postCreateCommand", "make setup")
set(repo, "remoteUser", "vscode")
# returns True
```

=== "Before"

    `.devcontainer/devcontainer.json`

    ```json
    {
      "image": "mcr.microsoft.com/devcontainers/go:1.23",
      "postCreateCommand": "go mod download" // cache modules
    }
    ```

=== "After"

    `.devcontainer/devcontainer.json`

    ```json
    {
      "remoteUser": "vscode",
      "image": "mcr.microsoft.com/devcontainers/go:1.23",
      "postCreateCommand": "make setup" // cache modules
    }
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/devcontainer.star#L275)
