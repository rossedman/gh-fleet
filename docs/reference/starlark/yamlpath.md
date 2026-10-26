---
title: yamlpath.star
---

# yamlpath.star

Read, set and delete keys in any YAML file by path, keeping comments and layout.

Keys are addressed with dots, such as `spec.replicas` or `jobs.test.runs-on`,
and `*` matches any one key, as in `jobs.*.runs-on`. For a key that itself
contains dots, such as the label `app.kubernetes.io/name`, pass the path as
a list: `["metadata", "labels", "app.kubernetes.io/name"]`. Edits change only the
lines they must, so comments, quoting, blank lines and key order survive,
which a decode and re-encode would lose. Paths go through mappings only:
keys inside list items can't be addressed. Only the first document of a
multi-document file is read.

```python
load("@fleet//yamlpath.star", "get", "values", "set", "get_in", "set_in", "set_sibling", "delete")
```

| Function | What it does |
| --- | --- |
| [`get`](#get) | Reads a scalar value by key path. |
| [`values`](#values) | Reads every scalar whose key path matches a pattern with `*` wildcards. |
| [`set`](#set) | Sets a key to a value, adding the key and any missing parents if needed. |
| [`get_in`](#get_in) | Reads a scalar value by key path from YAML text rather than a file. |
| [`set_in`](#set_in) | Sets a key in YAML text rather than a file, the same way `set` does. |
| [`set_sibling`](#set_sibling) | Sets `key` in every mapping where `match_key` has the value `match_value`, list items included. |
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

**Used in:** [`helm-image-tag`](../examples/helm-image-tag.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/yamlpath.star#L161)

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

**Used in:** [`runner-label`](../examples/runner-label.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/yamlpath.star#L189)

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

**Used in:** [`runner-label`](../examples/runner-label.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/yamlpath.star#L219)

## get_in

```python
get_in(text, key)
```

Reads a scalar value by key path from YAML text rather than a file.

Use it on text you already have, such as one document of a
multi-document file.

**Arguments**

| Name | Description |
| --- | --- |
| `text` | The YAML text. |
| `key` | The dotted key path. |

**Returns:** The value as a string, unquoted, or None when the key is missing or
holds a mapping or block scalar.

**Example**

```python
get_in("kind: Deployment\nmetadata:\n  name: api\n", "metadata.name")
# returns "api"
```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/yamlpath.star#L308)

## set_in

```python
set_in(text, key, value)
```

Sets a key in YAML text rather than a file, the same way `set` does.

**Arguments**

| Name | Description |
| --- | --- |
| `text` | The YAML text. |
| `key` | The dotted key path, where `*` matches any one key. |
| `value` | A string, number, boolean, None or list of those. |

**Returns:** The new text; unchanged when the key already had the value.

**Example**

```python
set_in("metadata:\n  name: api\n", "metadata.labels.team", "payments")
# returns "metadata:\n  name: api\n  labels:\n    team: payments\n"
```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/yamlpath.star#L331)

## set_sibling

```python
set_sibling(
    repo,
    path,
    match_key,
    match_value,
    key,
    value,
    never_lower = False,
)
```

Sets `key` in every mapping where `match_key` has the value `match_value`, list items included.

This is the common "set the version next to the name" edit: the `tag`
beside `repository: nginx`, the `targetRevision` beside a `repoURL`, the
`version` of the list item whose `name` is `community.general`. An
existing `key` is replaced, keeping its quotes and comment; a missing
one is added at the end of the mapping.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `path` | The YAML file, relative to the repo root. |
| `match_key` | The key that identifies the mapping. |
| `match_value` | The value `match_key` must have, compared unquoted. |
| `key` | The sibling key to set. |
| `value` | A string, number, boolean or None. |
| `never_lower` | Leave an existing value alone when it is a higher version than `value`. |

**Returns:** True when the file changed.

**Example**

```python
set_sibling(repo, "requirements.yml", "name", "community.general", "version", "9.4.0")
set_sibling(repo, "requirements.yml", "name", "ansible.posix", "version", "1.6.0")
# returns True
```

=== "Before"

    `requirements.yml`

    ```yaml
    collections:
      - name: community.general
        version: 7.0.0  # pinned
      - name: ansible.posix
    ```

=== "After"

    `requirements.yml`

    ```yaml
    collections:
      - name: community.general
        version: 9.4.0  # pinned
      - name: ansible.posix
        version: 1.6.0
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/yamlpath.star#L348)

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

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/yamlpath.star#L456)
