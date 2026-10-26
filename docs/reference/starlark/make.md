---
title: make.star
---

# make.star

Read and set variables in Makefiles, such as `GO_VERSION ?= 1.22`.

Edits are line-based and keep the assignment operator (`=`, `:=`, `::=`,
`?=`, `+=`), any `export` or `override` prefix, and trailing comments.

```python
load("@fleet//make.star", "makefiles", "get_var", "set_var")
```

| Function | What it does |
| --- | --- |
| [`makefiles`](#makefiles) | Lists the repo's Makefiles. |
| [`get_var`](#get_var) | Reads the value of the first assignment to a variable. |
| [`set_var`](#set_var) | Sets every assignment to a variable, in every Makefile that has one. |

## makefiles

```python
makefiles(repo)
```

Lists the repo's Makefiles.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The sorted paths of every Makefile, GNUmakefile, makefile and *.mk
file outside vendor and node_modules.

**Example**

```python
makefiles(repo)
# returns ["Makefile", "build/common.mk"]
```

=== "Before"

    `Makefile`

    ```text
    all:
    ```

    `build/common.mk`

    ```text
    X = 1
    ```

    `vendor/x/Makefile`

    ```text
    all:
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/make.star#L7)

## get_var

```python
get_var(repo, path, name)
```

Reads the value of the first assignment to a variable.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `path` | The Makefile to read. |
| `name` | The variable name. |

**Returns:** The value as written, or None when the file doesn't assign it.

**Example**

```python
get_var(repo, "Makefile", "GO_VERSION")
# returns "1.22.1"
```

=== "Before"

    `Makefile`

    ```text
    GO_VERSION ?= 1.22.1  # keep in sync with go.mod
    BIN := bin/app
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/make.star#L30)

## set_var

```python
set_var(repo, name, value, files = None)
```

Sets every assignment to a variable, in every Makefile that has one.

Makefiles that don't assign the variable are left alone. Appends
(`+=`) are left alone too, since they add to a value rather than set it.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `name` | The variable name. |
| `value` | The new value, as it should appear after the operator. |
| `files` | The Makefiles to edit; every Makefile by default. |

**Returns:** The list of paths that changed.

**Example**

```python
set_var(repo, "GO_VERSION", "1.27.3")
# returns ["Makefile"]
```

=== "Before"

    `Makefile`

    ```text
    GO_VERSION ?= 1.22.1  # keep in sync with go.mod
    export GOFLAGS = -mod=mod
    ```

=== "After"

    `Makefile`

    ```text
    GO_VERSION ?= 1.27.3  # keep in sync with go.mod
    export GOFLAGS = -mod=mod
    ```

**Used in:** [`make-go-version`](../examples/make-go-version.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/make.star#L53)
