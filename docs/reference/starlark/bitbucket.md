---
title: bitbucket.star
---

# bitbucket.star

Edit Bitbucket Pipelines: step images and pipe versions.

Edits are line-based, so comments and anchors in bitbucket-pipelines.yml
stay as they were.

```python
load("@fleet//bitbucket.star", "pipes", "bump_pipe", "bump_image")
```

| Function | What it does |
| --- | --- |
| [`pipes`](#pipes) | Lists the pipes the pipeline uses, with their versions. |
| [`bump_pipe`](#bump_pipe) | Moves a pipe to a new version. |
| [`bump_image`](#bump_image) | Moves the default or a step's `image` from one tag to another, keeping variants. |

## pipes

```python
pipes(repo)
```

Lists the pipes the pipeline uses, with their versions.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The sorted, de-duplicated `name:version` strings.

**Example**

```python
pipes(repo)
# returns ["atlassian/aws-s3-deploy:1.1.0", "atlassian/slack-notify:2.0.0"]
```

=== "Before"

    `bitbucket-pipelines.yml`

    ```yaml
    pipelines:
      default:
        - step:
            script:
              - pipe: atlassian/aws-s3-deploy:1.1.0
              - pipe: atlassian/slack-notify:2.0.0
    ```

**Used in:** [`bitbucket-pipes`](../examples/bitbucket-pipes.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/bitbucket.star#L12)

## bump_pipe

```python
bump_pipe(repo, pipe, version, from_ref = None)
```

Moves a pipe to a new version.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `pipe` | The pipe, such as `atlassian/aws-s3-deploy`. |
| `version` | The new version, such as `1.6.1`. |
| `from_ref` | When set, only this exact version is replaced. |

**Returns:** The list of paths that changed.

**Example**

```python
bump_pipe(repo, "atlassian/aws-s3-deploy", "1.6.1")
# returns ["bitbucket-pipelines.yml"]
```

=== "Before"

    `bitbucket-pipelines.yml`

    ```yaml
    script:
      - pipe: atlassian/aws-s3-deploy:1.1.0
        variables:
          S3_BUCKET: acme-web
    ```

=== "After"

    `bitbucket-pipelines.yml`

    ```yaml
    script:
      - pipe: atlassian/aws-s3-deploy:1.6.1
        variables:
          S3_BUCKET: acme-web
    ```

**Used in:** [`bitbucket-pipes`](../examples/bitbucket-pipes.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/bitbucket.star#L37)

## bump_image

```python
bump_image(repo, image, from_tag, to_tag)
```

Moves the default or a step's `image` from one tag to another, keeping variants.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `image` | The image name, such as `node`. |
| `from_tag` | The tag to replace. |
| `to_tag` | The tag to put in its place. |

**Returns:** The list of paths that changed.

**Example**

```python
bump_image(repo, "node", "18", "22")
# returns ["bitbucket-pipelines.yml"]
```

=== "Before"

    `bitbucket-pipelines.yml`

    ```yaml
    image: node:18-bullseye
    pipelines:
      default:
        - step:
            image: python:3.10
    ```

=== "After"

    `bitbucket-pipelines.yml`

    ```yaml
    image: node:22-bullseye
    pipelines:
      default:
        - step:
            image: python:3.10
    ```

**Used in:** [`bitbucket-pipes`](../examples/bitbucket-pipes.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/bitbucket.star#L64)
