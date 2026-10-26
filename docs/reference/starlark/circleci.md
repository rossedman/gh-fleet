---
title: circleci.star
---

# circleci.star

Edit CircleCI configuration: orb versions, executor images, machine images and resource classes.

Edits are line-based, so comments and anchors stay as they were.

```python
load("@fleet//circleci.star", "configs", "orbs", "bump_orb", "bump_image", "set_resource_class", "set_machine_image")
```

| Function | What it does |
| --- | --- |
| [`configs`](#configs) | Lists the repo's CircleCI configs. |
| [`orbs`](#orbs) | Lists the orbs configs use, with their versions. |
| [`bump_orb`](#bump_orb) | Moves an orb to a new version in every config. |
| [`bump_image`](#bump_image) | Moves a Docker executor image from one tag to another. |
| [`set_resource_class`](#set_resource_class) | Sets `resource_class` on every job and executor that sets one. |
| [`set_machine_image`](#set_machine_image) | Sets the `image` of every machine executor, such as when an Ubuntu image is retired. |

## configs

```python
configs(repo)
```

Lists the repo's CircleCI configs.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The sorted paths of every YAML file under .circleci, including
dynamic and continuation configs.

**Example**

```python
configs(repo)
# returns [".circleci/config.yml", ".circleci/continue/deploy.yml"]
```

=== "Before"

    `.circleci/config.yml`

    ```yaml
    version: 2.1
    ```

    `.circleci/continue/deploy.yml`

    ```yaml
    version: 2.1
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/circleci.star#L11)

## orbs

```python
orbs(repo)
```

Lists the orbs configs use, with their versions.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The sorted, de-duplicated `namespace/orb@version` strings.

**Example**

```python
orbs(repo)
# returns ["circleci/aws-cli@4.0", "circleci/node@5.1.0"]
```

=== "Before"

    `.circleci/config.yml`

    ```yaml
    version: 2.1
    orbs:
      node: circleci/node@5.1.0
      aws-cli: "circleci/aws-cli@4.0"
    ```

**Used in:** [`circleci-orbs`](https://github.com/rossedman/gh-fleet/tree/main/examples/circleci-orbs)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/circleci.star#L29)

## bump_orb

```python
bump_orb(repo, orb, version, from_ref = None)
```

Moves an orb to a new version in every config.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `orb` | The orb as `namespace/name`, such as `circleci/node`; `*` matches any run of name characters, as in `acme/*`. |
| `version` | The new version, such as `5.2.0`. |
| `from_ref` | When set, only this exact version is replaced. |

**Returns:** The list of paths that changed.

**Example**

```python
bump_orb(repo, "circleci/node", "5.2.0")
# returns [".circleci/config.yml"]
```

=== "Before"

    `.circleci/config.yml`

    ```yaml
    orbs:
      node: circleci/node@5.1.0  # pinned
      slack: circleci/slack@4.12.0
    ```

=== "After"

    `.circleci/config.yml`

    ```yaml
    orbs:
      node: circleci/node@5.2.0  # pinned
      slack: circleci/slack@4.12.0
    ```

**Used in:** [`circleci-orbs`](https://github.com/rossedman/gh-fleet/tree/main/examples/circleci-orbs), [`jenkins-shared-library`](https://github.com/rossedman/gh-fleet/tree/main/examples/jenkins-shared-library)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/circleci.star#L60)

## bump_image

```python
bump_image(repo, image, from_tag, to_tag)
```

Moves a Docker executor image from one tag to another.

A variant suffix is kept, so `cimg/node:18.20-browsers` becomes
`cimg/node:22.11-browsers`.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `image` | The image name, such as `cimg/node`. |
| `from_tag` | The tag to replace, such as `18.20`. |
| `to_tag` | The tag to put in its place. |

**Returns:** The list of paths that changed.

**Example**

```python
bump_image(repo, "cimg/node", "18.20", "22.11")
# returns [".circleci/config.yml"]
```

=== "Before"

    `.circleci/config.yml`

    ```yaml
    jobs:
      test:
        docker:
          - image: cimg/node:18.20-browsers
          - image: cimg/postgres:15.4
    ```

=== "After"

    `.circleci/config.yml`

    ```yaml
    jobs:
      test:
        docker:
          - image: cimg/node:22.11-browsers
          - image: cimg/postgres:15.4
    ```

**Used in:** [`circleci-machine-image`](https://github.com/rossedman/gh-fleet/tree/main/examples/circleci-machine-image)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/circleci.star#L86)

## set_resource_class

```python
set_resource_class(repo, resource_class, from_class = None)
```

Sets `resource_class` on every job and executor that sets one.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `resource_class` | The new class, such as `large` or `arm.medium`. |
| `from_class` | When set, only this class is replaced; `*` matches any run of characters. |

**Returns:** The list of paths that changed.

**Example**

```python
set_resource_class(repo, "large", from_class = "medium")
# returns [".circleci/config.yml"]
```

=== "Before"

    `.circleci/config.yml`

    ```yaml
    jobs:
      build:
        resource_class: medium
      e2e:
        resource_class: xlarge
    ```

=== "After"

    `.circleci/config.yml`

    ```yaml
    jobs:
      build:
        resource_class: large
      e2e:
        resource_class: xlarge
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/circleci.star#L149)

## set_machine_image

```python
set_machine_image(repo, image, from_image = None)
```

Sets the `image` of every machine executor, such as when an Ubuntu image is retired.

Only `image` keys under a `machine:` key change; Docker executor images
are left alone.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `image` | The new machine image, such as `ubuntu-2404:2024.08.1`. |
| `from_image` | When set, only this image is replaced; `*` matches any run of characters, so `ubuntu-2004:*` covers every tag. |

**Returns:** The list of paths that changed.

**Example**

```python
set_machine_image(repo, "ubuntu-2404:2024.08.1", from_image = "ubuntu-2004:*")
# returns [".circleci/config.yml"]
```

=== "Before"

    `.circleci/config.yml`

    ```yaml
    jobs:
      integration:
        machine:
          image: ubuntu-2004:202201-02
        docker_layer_caching: true
      unit:
        docker:
          - image: cimg/base:2024.01
    ```

=== "After"

    `.circleci/config.yml`

    ```yaml
    jobs:
      integration:
        machine:
          image: ubuntu-2404:2024.08.1
        docker_layer_caching: true
      unit:
        docker:
          - image: cimg/base:2024.01
    ```

**Used in:** [`circleci-machine-image`](https://github.com/rossedman/gh-fleet/tree/main/examples/circleci-machine-image)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/circleci.star#L178)
