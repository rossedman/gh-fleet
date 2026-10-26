---
title: toml.star
---

# toml.star

Read, set and delete keys in TOML files, such as pyproject.toml and Cargo.toml.

Edits are line-based, so comments, ordering and formatting stay as they
were. A key is addressed by its table, such as `tool.ruff` (`""` for the
top level), and its name. Array-of-tables (`[[...]]`) entries and dotted
keys can't be addressed.

```python
load("@fleet//toml.star", "get", "set", "delete")
```

| Function | What it does |
| --- | --- |
| [`get`](#get) | Reads a key's value. |
| [`set`](#set) | Sets a key, adding it, and its table, when missing. |
| [`delete`](#delete) | Removes a key from a table. |

## get

```python
get(repo, path, table, key)
```

Reads a key's value.

Strings, integers, floats and booleans are decoded; arrays, inline
tables and dates are returned as written.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `path` | The TOML file, relative to the repo root. |
| `table` | The table name, such as `tool.ruff`, or `""` for the top level. |
| `key` | The key name. |

**Returns:** The value, or None when the file, table or key is missing.

**Example**

```python
get(repo, "pyproject.toml", "tool.ruff", "line-length")
# returns 88
```

=== "Before"

    `pyproject.toml`

    ```toml
    [tool.ruff]
    line-length = 88  # black's default
    target-version = "py39"
    ```

**Used in:** [`ruff-config`](https://github.com/rossedman/gh-fleet/tree/main/examples/ruff-config)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/toml.star#L123)

## set

```python
set(repo, path, table, key, value)
```

Sets a key, adding it, and its table, when missing.

An existing value is replaced, keeping a trailing comment and the
string quote style. A new key goes after the last key of its table; a
new table goes at the end of the file.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `path` | The TOML file, relative to the repo root; created if needed. |
| `table` | The table name, such as `tool.ruff`, or `""` for the top level. |
| `key` | The key name. |
| `value` | A string, number, boolean, or a list or dict of those. |

**Returns:** True when the file changed.

**Example**

```python
set(repo, "pyproject.toml", "tool.ruff", "line-length", 100)
set(repo, "pyproject.toml", "tool.ruff.lint", "select", ["E", "F", "I"])
# returns True
```

=== "Before"

    `pyproject.toml`

    ```toml
    [project]
    name = "api"

    [tool.ruff]
    line-length = 88  # black's default
    ```

=== "After"

    `pyproject.toml`

    ```toml
    [project]
    name = "api"

    [tool.ruff]
    line-length = 100  # black's default

    [tool.ruff.lint]
    select = ["E", "F", "I"]
    ```

**Used in:** [`ruff-config`](https://github.com/rossedman/gh-fleet/tree/main/examples/ruff-config)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/toml.star#L156)

## delete

```python
delete(repo, path, table, key)
```

Removes a key from a table.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `path` | The TOML file, relative to the repo root. |
| `table` | The table name, or `""` for the top level. |
| `key` | The key name. |

**Returns:** True when the file changed.

**Example**

```python
delete(repo, "pyproject.toml", "tool.black", "experimental-string-processing")
# returns True
```

=== "Before"

    `pyproject.toml`

    ```toml
    [tool.black]
    line-length = 100
    experimental-string-processing = true
    ```

=== "After"

    `pyproject.toml`

    ```toml
    [tool.black]
    line-length = 100
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/toml.star#L232)
