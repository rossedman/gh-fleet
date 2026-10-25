---
title: precommit.star
---

# precommit.star

Read and bump hook revisions in .pre-commit-config.yaml.

Edits are line-based, so comments and layout stay as they were.

```python
load("@fleet//precommit.star", "revs", "set_rev")
```

| Function | What it does |
| --- | --- |
| [`revs`](#revs) | Reads the revision each hook repository is pinned to. |
| [`set_rev`](#set_rev) | Pins a hook repository to a new revision, keeping quotes and comments. |

## revs

```python
revs(repo, path = CONFIG)
```

Reads the revision each hook repository is pinned to.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `path` | The pre-commit config to read. |

**Returns:** A dict of repository URL to `rev`, in file order.

**Example**

```python
revs(repo)
# returns {"https://github.com/pre-commit/pre-commit-hooks": "v4.6.0", "https://github.com/psf/black": "24.4.0"}
```

=== "Before"

    `.pre-commit-config.yaml`

    ```yaml
    repos:
      - repo: https://github.com/pre-commit/pre-commit-hooks
        rev: v4.6.0
        hooks:
          - id: check-yaml
      - repo: https://github.com/psf/black
        rev: "24.4.0"
        hooks:
          - id: black
    ```

**Used in:** [`pre-commit-rev`](https://github.com/rossedman/gh-fleet/tree/main/examples/pre-commit-rev)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/precommit.star#L15)

## set_rev

```python
set_rev(repo, url, rev, path = CONFIG)
```

Pins a hook repository to a new revision, keeping quotes and comments.

The URL matches with or without a trailing `.git`. A config that does
not use the repository is left alone.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `url` | The hook repository URL. |
| `rev` | The new revision, such as `v5.0.0`. |
| `path` | The pre-commit config to edit. |

**Returns:** True when the file changed.

**Example**

```python
set_rev(repo, "https://github.com/psf/black", "24.10.0")
# returns True
```

=== "Before"

    `.pre-commit-config.yaml`

    ```yaml
    repos:
      - repo: https://github.com/psf/black
        rev: "24.4.0"  # formatter
        hooks:
          - id: black
    ```

=== "After"

    `.pre-commit-config.yaml`

    ```yaml
    repos:
      - repo: https://github.com/psf/black
        rev: "24.10.0"  # formatter
        hooks:
          - id: black
    ```

**Used in:** [`pre-commit-rev`](https://github.com/rossedman/gh-fleet/tree/main/examples/pre-commit-rev)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/precommit.star#L51)
