---
title: gomod.star
---

# gomod.star

Read and raise the go directive in go.mod files.

```python
load("@fleet//gomod.star", "modules", "go_version", "set_go_version", "requires", "outdated")
```

| Function | What it does |
| --- | --- |
| [`modules`](#modules) | Lists the repo's Go modules. |
| [`go_version`](#go_version) | Reads the version in a go.mod's `go` directive. |
| [`set_go_version`](#set_go_version) | Raises the `go` directive in every go.mod, never lowering it. |
| [`requires`](#requires) | Reads every module a go.mod requires, in single-line and block form. |
| [`outdated`](#outdated) | Finds the modules that require a dependency below a version. |

## modules

```python
modules(repo)
```

Lists the repo's Go modules.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The sorted paths of every go.mod outside vendor and testdata
directories.

**Example**

```python
modules(repo)
# returns ["go.mod", "tools/go.mod"]
```

=== "Before"

    `go.mod`

    ```text
    module acme/api
    ```

    `tools/go.mod`

    ```text
    module acme/tools
    ```

    `vendor/x/go.mod`

    ```text
    module x
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/gomod.star#L3)

## go_version

```python
go_version(repo, path = "go.mod")
```

Reads the version in a go.mod's `go` directive.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `path` | The go.mod to read. |

**Returns:** The version, such as `1.22.1`, or None when the file has no `go`
directive or does not exist.

**Example**

```python
go_version(repo)
# returns "1.22.1"
```

=== "Before"

    `go.mod`

    ```text
    module acme/api

    go 1.22.1
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/gomod.star#L23)

## set_go_version

```python
set_go_version(repo, version, paths = None)
```

Raises the `go` directive in every go.mod, never lowering it.

A go.mod without a directive gets one after its `module` line. A
`toolchain` line the new version covers is removed; a newer one is kept.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `version` | The Go version, such as `1.27.3`. |
| `paths` | The go.mod files to change; every module by default. |

**Returns:** The list of paths that changed.

**Example**

```python
set_go_version(repo, "1.27.3")
# returns ["go.mod"]
```

=== "Before"

    `go.mod`

    ```text
    module acme/api

    go 1.22.1

    toolchain go1.22.5
    ```

    `tools/go.mod`

    ```text
    module acme/tools

    go 1.28
    ```

=== "After"

    `go.mod`

    ```text
    module acme/api

    go 1.27.3
    ```

**Used in:** [`bump-go-version`](https://github.com/rossedman/gh-fleet/tree/main/examples/bump-go-version), [`make-go-version`](https://github.com/rossedman/gh-fleet/tree/main/examples/make-go-version)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/gomod.star#L44)

## requires

```python
requires(repo, path = "go.mod")
```

Reads every module a go.mod requires, in single-line and block form.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `path` | The go.mod to read. |

**Returns:** A dict of module path to version, in file order; empty when the file
does not exist.

**Example**

```python
requires(repo)
# returns {"golang.org/x/net": "v0.20.0", "github.com/spf13/cobra": "v1.8.0"}
```

=== "Before"

    `go.mod`

    ```text
    module acme/api

    require (
        golang.org/x/net v0.20.0 // indirect
        github.com/spf13/cobra v1.8.0
    )
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/gomod.star#L95)

## outdated

```python
outdated(repo, module, version)
```

Finds the modules that require a dependency below a version.

Use it to pick where a `run` step should run `go get module@version`.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `module` | The dependency's module path. |
| `version` | The version it should be at least, such as `v0.33.0`. |

**Returns:** The directories of the matching go.mod files; the repo root is `.`.

**Example**

```python
outdated(repo, "golang.org/x/net", "v0.33.0")
# returns ["."]
```

=== "Before"

    `go.mod`

    ```text
    module acme/api

    require golang.org/x/net v0.20.0
    ```

    `tools/go.mod`

    ```text
    module acme/tools

    require golang.org/x/net v0.40.0
    ```

**Used in:** [`go-module-bump`](https://github.com/rossedman/gh-fleet/tree/main/examples/go-module-bump)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/gomod.star#L133)
