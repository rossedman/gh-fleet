---
title: yamlpath.star
---

# yamlpath.star

Read, set and delete keys in any YAML file by path, keeping comments and layout.

Keys are addressed with dots, such as `spec.replicas` or `jobs.test.runs-on`,
and `*` matches any one key, as in `jobs.*.runs-on`. Edits change only the
lines they must, so comments, quoting, blank lines and key order survive,
which a decode and re-encode would lose. Paths go through mappings only:
keys inside list items can't be addressed. Only the first document of a
multi-document file is read.

```python
load("@fleet//yamlpath.star", "get", "values", "set", "delete")
```

| Function | What it does |
| --- | --- |
| [`get`](#get) | Reads a scalar value by key path. |
| [`values`](#values) | Reads every scalar whose key path matches a pattern with `*` wildcards. |
| [`set`](#set) | Sets a key to a value, adding the key and any missing parents if needed. |
| [`delete`](#delete) | Removes a key and everything nested under it. |

## get

```python
get(repo, path, key)
```

Reads a scalar value by key path.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `path` | The YAML file, relative to the repo root. |
| `key` | The dotted key path, such as `spec.replicas`. |

**Returns:** The value as a string, unquoted, or None when the file or key is
missing or the key holds a mapping or block scalar.

**Example**

```python
get(repo, "deploy.yaml", "spec.template.metadata.labels.app")
# returns "api"
```

=== "Before"

    `deploy.yaml`

    ```yaml
    spec:
      replicas: 2  # scaled by HPA
      template:
        metadata:
          labels:
            app: "api"
    ```

**Used in:** [`helm-image-tag`](https://github.com/rossedman/gh-fleet/tree/main/examples/helm-image-tag)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/yamlpath.star#L146)

## values

```python
values(repo, path, key)
```

Reads every scalar whose key path matches a pattern with `*` wildcards.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `path` | The YAML file, relative to the repo root. |
| `key` | The dotted key path, where `*` matches any one key, such as `jobs.*.runs-on`. |

**Returns:** A dict of each matching key path to its value, in file order.

**Example**

```python
values(repo, ".github/workflows/ci.yml", "jobs.*.runs-on")
# returns {"jobs.test.runs-on": "ubuntu-latest", "jobs.lint.runs-on": "ubuntu-22.04"}
```

=== "Before"

    `.github/workflows/ci.yml`

    ```yaml
    jobs:
      test:
        runs-on: ubuntu-latest
      lint:
        runs-on: ubuntu-22.04
    ```

**Used in:** [`runner-label`](https://github.com/rossedman/gh-fleet/tree/main/examples/runner-label)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/yamlpath.star#L174)

## set

```python
set(repo, path, key, value)
```

Sets a key to a value, adding the key and any missing parents if needed.

An existing value is replaced on its own line, keeping its quote style
and trailing comment. A missing key is added as the last child of its
deepest existing parent, at the indentation the file already uses. With
`*` in the path, every matching existing key is set and nothing is
added. Strings are written plain when YAML would read them back as the
same string, and double-quoted otherwise.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `path` | The YAML file, relative to the repo root; created if needed. |
| `key` | The dotted key path, where `*` matches any one key. |
| `value` | A string, number, boolean, None or list of those. |

**Returns:** True when the file changed.

**Example**

```python
set(repo, "deploy.yaml", "spec.replicas", 3)
set(repo, "deploy.yaml", "spec.revisionHistoryLimit", 5)
# returns True
```

=== "Before"

    `deploy.yaml`

    ```yaml
    spec:
      replicas: 2  # scaled by HPA
      template:
        spec:
          containers: []
    ```

=== "After"

    `deploy.yaml`

    ```yaml
    spec:
      replicas: 3  # scaled by HPA
      template:
        spec:
          containers: []
      revisionHistoryLimit: 5
    ```

**Used in:** [`runner-label`](https://github.com/rossedman/gh-fleet/tree/main/examples/runner-label)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/yamlpath.star#L204)

## delete

```python
delete(repo, path, key)
```

Removes a key and everything nested under it.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `path` | The YAML file, relative to the repo root. |
| `key` | The dotted key path, where `*` matches any one key. |

**Returns:** True when the file changed.

**Example**

```python
delete(repo, ".github/workflows/ci.yml", "jobs.test.continue-on-error")
# returns True
```

=== "Before"

    `.github/workflows/ci.yml`

    ```yaml
    jobs:
      test:
        runs-on: ubuntu-latest
        continue-on-error: true
        steps: []
    ```

=== "After"

    `.github/workflows/ci.yml`

    ```yaml
    jobs:
      test:
        runs-on: ubuntu-latest
        steps: []
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/yamlpath.star#L288)
