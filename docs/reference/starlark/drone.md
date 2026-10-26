---
title: drone.star
---

# drone.star

Move step and plugin images in Drone and Woodpecker CI pipelines.

Both tools write steps with an `image:` key, and plugins are images too,
so one helper covers moving a build image and upgrading a plugin.

```python
load("@fleet//drone.star", "pipelines", "bump_image")
```

| Function | What it does |
| --- | --- |
| [`pipelines`](#pipelines) | Lists the repo's Drone and Woodpecker pipeline files. |
| [`bump_image`](#bump_image) | Moves a step or plugin image from one tag to another, keeping variants. |

## pipelines

```python
pipelines(repo)
```

Lists the repo's Drone and Woodpecker pipeline files.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The sorted paths of .drone.yml, .woodpecker.yml and the files under
.woodpecker.

**Example**

```python
pipelines(repo)
# returns [".drone.yml", ".woodpecker/test.yml"]
```

=== "Before"

    `.drone.yml`

    ```yaml
    kind: pipeline
    ```

    `.woodpecker/test.yml`

    ```yaml
    steps: {}
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/drone.star#L11)

## bump_image

```python
bump_image(repo, image, from_tag, to_tag)
```

Moves a step or plugin image from one tag to another, keeping variants.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `image` | The image, such as `golang` or `plugins/docker`. |
| `from_tag` | The tag to replace. |
| `to_tag` | The tag to put in its place. |

**Returns:** The list of paths that changed.

**Example**

```python
bump_image(repo, "golang", "1.22", "1.23")
bump_image(repo, "plugins/docker", "20", "21")
# returns [".drone.yml"]
```

=== "Before"

    `.drone.yml`

    ```yaml
    kind: pipeline
    steps:
      - name: test
        image: golang:1.22-alpine
      - name: publish
        image: plugins/docker:20
    ```

=== "After"

    `.drone.yml`

    ```yaml
    kind: pipeline
    steps:
      - name: test
        image: golang:1.23-alpine
      - name: publish
        image: plugins/docker:21
    ```

**Used in:** [`drone-golang`](https://github.com/rossedman/gh-fleet/tree/main/examples/drone-golang)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/drone.star#L29)
