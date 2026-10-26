---
title: runtime.star
---

# runtime.star

Set language versions in version files: .nvmrc, .python-version, .tool-versions and friends.

```python
load("@fleet//runtime.star", "version", "set_version")
```

| Function | What it does |
| --- | --- |
| [`version`](#version) | Reads the version a repo pins for a runtime. |
| [`set_version`](#set_version) | Sets a runtime's version in every version file the repo already has. |

## version

```python
version(repo, runtime)
```

Reads the version a repo pins for a runtime.

Dedicated files such as .nvmrc are read before .tool-versions.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `runtime` | One of `node`, `python`, `ruby`, `go`, `java`, `terraform`, `elixir`, `erlang` or `rust`. |

**Returns:** The version as written, or None when nothing pins it.

**Example**

```python
version(repo, "node")
# returns "20.11.0"
```

=== "Before"

    `.tool-versions`

    ```text
    nodejs 20.11.0
    python 3.12.1
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/runtime.star#L21)

## set_version

```python
set_version(repo, runtime, version, create = None)
```

Sets a runtime's version in every version file the repo already has.

Updates the dedicated files (.nvmrc and .node-version for node,
.python-version for python, and so on) and the runtime's line in
.tool-versions. A repo that pins the runtime nowhere is left alone unless
`create` is set.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `runtime` | One of `node`, `python`, `ruby`, `go`, `java`, `terraform`, `elixir`, `erlang` or `rust`. |
| `version` | The version to pin, such as `22` or `3.12.7`. |
| `create` | A file to write, such as `.nvmrc`, when nothing pins the runtime yet. |

**Returns:** The list of paths that changed.

**Example**

```python
set_version(repo, "node", "22")
# returns [".nvmrc", ".tool-versions"]
```

=== "Before"

    `.nvmrc`

    ```text
    20
    ```

    `.tool-versions`

    ```text
    nodejs 20.11.0
    python 3.12.1
    ```

=== "After"

    `.nvmrc`

    ```text
    22
    ```

    `.tool-versions`

    ```text
    nodejs 22
    python 3.12.1
    ```

**Used in:** [`elixir-phoenix`](https://github.com/rossedman/gh-fleet/tree/main/examples/elixir-phoenix), [`node-version`](https://github.com/rossedman/gh-fleet/tree/main/examples/node-version), [`python-version`](https://github.com/rossedman/gh-fleet/tree/main/examples/python-version)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/runtime.star#L51)
