---
title: cargo.star
---

# cargo.star

Bump Rust crate dependencies and rust-version in Cargo.toml files.

Edits are line-based, so comments, features and table layout stay as they
were. Cargo.lock is not touched: follow with a `run` step such as
`cargo update -p <crate>` when the lockfile is committed.

```python
load("@fleet//cargo.star", "manifests", "set_dependency", "rust_version", "set_rust_version")
```

| Function | What it does |
| --- | --- |
| [`manifests`](#manifests) | Lists the repo's Cargo manifests. |
| [`set_dependency`](#set_dependency) | Sets a crate's version wherever a Cargo.toml already depends on it. |
| [`rust_version`](#rust_version) | Reads the `rust-version` of a Cargo.toml. |
| [`set_rust_version`](#set_rust_version) | Raises `rust-version` in every Cargo.toml that sets it, never lowering it. |

## manifests

```python
manifests(repo)
```

Lists the repo's Cargo manifests.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The sorted paths of every Cargo.toml outside target directories.

**Example**

```python
manifests(repo)
# returns ["Cargo.toml", "crates/core/Cargo.toml"]
```

=== "Before"

    `Cargo.toml`

    ```toml
    [workspace]
    ```

    `crates/core/Cargo.toml`

    ```toml
    [package]
    ```

    `target/package/x/Cargo.toml`

    ```toml
    [package]
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/cargo.star#L8)

## set_dependency

```python
set_dependency(repo, name, version)
```

Sets a crate's version wherever a Cargo.toml already depends on it.

A newer version already there is kept, so a campaign never lowers one.

Covers `[dependencies]`, `[dev-dependencies]`, `[build-dependencies]`,
`[workspace.dependencies]` and their `[target.'...'.dependencies]` forms,
written as `name = "1.0"` or `name = { version = "1.0", ... }`.
Dependencies given only by `path` or `git` have no version and are left
alone.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `name` | The crate name. |
| `version` | The new version requirement, such as `1.0.210`. |

**Returns:** The list of paths that changed.

**Example**

```python
set_dependency(repo, "serde", "1.0.210")
# returns ["Cargo.toml"]
```

=== "Before"

    `Cargo.toml`

    ```toml
    [dependencies]
    serde = { version = "1.0.100", features = ["derive"] }
    tokio = "1.30"

    [dev-dependencies]
    serde = "1.0"
    ```

=== "After"

    `Cargo.toml`

    ```toml
    [dependencies]
    serde = { version = "1.0.210", features = ["derive"] }
    tokio = "1.30"

    [dev-dependencies]
    serde = "1.0.210"
    ```

**Used in:** [`cargo-dependency-bump`](https://github.com/rossedman/gh-fleet/tree/main/examples/cargo-dependency-bump)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/cargo.star#L30)

## rust_version

```python
rust_version(repo, path = "Cargo.toml")
```

Reads the `rust-version` of a Cargo.toml.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `path` | The Cargo.toml to read. |

**Returns:** The version, such as `1.70`, or None.

**Example**

```python
rust_version(repo)
# returns "1.70"
```

=== "Before"

    `Cargo.toml`

    ```toml
    [package]
    name = "api"
    rust-version = "1.70"
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/cargo.star#L91)

## set_rust_version

```python
set_rust_version(repo, version)
```

Raises `rust-version` in every Cargo.toml that sets it, never lowering it.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `version` | The minimum Rust version, such as `1.75`. |

**Returns:** The list of paths that changed.

**Example**

```python
set_rust_version(repo, "1.75")
# returns ["Cargo.toml"]
```

=== "Before"

    `Cargo.toml`

    ```toml
    [package]
    name = "api"
    rust-version = "1.70"
    ```

=== "After"

    `Cargo.toml`

    ```toml
    [package]
    name = "api"
    rust-version = "1.75"
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/cargo.star#L111)
