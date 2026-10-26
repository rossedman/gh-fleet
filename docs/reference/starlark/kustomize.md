---
title: kustomize.star
---

# kustomize.star

Set image tags and names in kustomization files, the usual way to roll out a release with Kustomize.

Edits are line-based, so comments and layout stay as they were.

```python
load("@fleet//kustomize.star", "files", "set_image")
```

| Function | What it does |
| --- | --- |
| [`files`](#files) | Lists the repo's kustomization files. |
| [`set_image`](#set_image) | Sets `newTag`, `newName` or `digest` for an image in every kustomization's `images` list. |

## files

```python
files(repo)
```

Lists the repo's kustomization files.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The sorted paths of every kustomization.yaml, kustomization.yml
and Kustomization file.

**Example**

```python
files(repo)
# returns ["deploy/base/kustomization.yaml", "deploy/prod/kustomization.yml"]
```

=== "Before"

    `deploy/base/kustomization.yaml`

    ```yaml
    resources: []
    ```

    `deploy/prod/kustomization.yml`

    ```yaml
    resources: []
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/kustomize.star#L8)

## set_image

```python
set_image(repo, name, new_tag = None, new_name = None, digest = None)
```

Sets `newTag`, `newName` or `digest` for an image in every kustomization's `images` list.

Only kustomizations that already list the image change; an entry is
found by its `name`. A missing `newTag` or `newName` key is added to the
entry.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `name` | The image name as the manifests use it, such as `ghcr.io/acme/api`. |
| `new_tag` | The tag to set, or None to leave it. |
| `new_name` | The replacement image name to set, or None to leave it. |
| `digest` | The digest to set, such as `sha256:...`, or None to leave it. |

**Returns:** The list of paths that changed.

**Example**

```python
set_image(repo, "ghcr.io/acme/api", new_tag = "2.4.0")
# returns ["deploy/prod/kustomization.yaml"]
```

=== "Before"

    `deploy/prod/kustomization.yaml`

    ```yaml
    resources:
      - ../base
    images:
      - name: ghcr.io/acme/api
        newTag: "2.3.1"  # release
      - name: redis
        newTag: "7"
    ```

=== "After"

    `deploy/prod/kustomization.yaml`

    ```yaml
    resources:
      - ../base
    images:
      - name: ghcr.io/acme/api
        newTag: "2.4.0"  # release
      - name: redis
        newTag: "7"
    ```

**Used in:** [`kustomize-image`](https://github.com/rossedman/gh-fleet/tree/main/examples/kustomize-image)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/kustomize.star#L26)
