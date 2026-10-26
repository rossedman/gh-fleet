---
title: npm.star
---

# npm.star

Find package.json files and edit them as data, keeping key order and indentation.

These helpers edit package.json only. To refresh package-lock.json, follow
the Starlark step with a `run` step such as
`npm install --package-lock-only --ignore-scripts`.

```python
load("@fleet//npm.star", "packages", "read_package", "write_package", "set_dependency")
```

| Function | What it does |
| --- | --- |
| [`packages`](#packages) | Lists the repo's package.json files. |
| [`read_package`](#read_package) | Reads a package.json as a dict whose keys keep the file's order. |
| [`write_package`](#write_package) | Writes a dict as a package.json, keeping the file's indentation. |
| [`set_dependency`](#set_dependency) | Sets a dependency's version range in every package.json that already lists it. |

## packages

```python
packages(repo)
```

Lists the repo's package.json files.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The sorted paths of every package.json outside node_modules.

**Example**

```python
packages(repo)
# returns ["package.json", "packages/ui/package.json"]
```

=== "Before"

    `package.json`

    ```json
    {"name": "web"}
    ```

    `packages/ui/package.json`

    ```json
    {"name": "ui"}
    ```

    `node_modules/x/package.json`

    ```json
    {"name": "x"}
    ```

**Used in:** [`node-engines`](https://github.com/rossedman/gh-fleet/tree/main/examples/node-engines)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/npm.star#L12)

## read_package

```python
read_package(repo, path = "package.json")
```

Reads a package.json as a dict whose keys keep the file's order.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `path` | The package.json to read. |

**Returns:** The decoded package.json.

**Example**

```python
read_package(repo)["name"]
# returns "web"
```

=== "Before"

    `package.json`

    ```json
    {"name": "web", "private": true}
    ```

**Used in:** [`node-engines`](https://github.com/rossedman/gh-fleet/tree/main/examples/node-engines)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/npm.star#L31)

## write_package

```python
write_package(repo, path, pkg)
```

Writes a dict as a package.json, keeping the file's indentation.

The indentation the file already uses is detected; a new file gets two
spaces. Key order is kept, so only what you changed shows in the diff.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `path` | The package.json to write. |
| `pkg` | The package, usually from `read_package`. |

**Returns:** True when the file changed.

**Example**

```python
pkg = read_package(repo)
pkg["private"] = True
write_package(repo, "package.json", pkg)
# returns True
```

=== "Before"

    `package.json`

    ```json
    {
        "name": "web"
    }
    ```

=== "After"

    `package.json`

    ```json
    {
        "name": "web",
        "private": true
    }
    ```

**Used in:** [`node-engines`](https://github.com/rossedman/gh-fleet/tree/main/examples/node-engines)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/npm.star#L48)

## set_dependency

```python
set_dependency(repo, name, version, sections = SECTIONS)
```

Sets a dependency's version range in every package.json that already lists it.

Packages that do not depend on `name` are left alone, so this never adds
a dependency. package-lock.json is not touched; follow with a `run` step
such as `npm install --package-lock-only --ignore-scripts`.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `name` | The package name. |
| `version` | The new range, such as `^1.7.9`. |
| `sections` | The dependency sections to look in. |

**Returns:** The list of paths that changed.

**Example**

```python
set_dependency(repo, "axios", "^1.7.9")
# returns ["package.json"]
```

=== "Before"

    `package.json`

    ```json
    {
      "dependencies": {
        "axios": "^1.6.0"
      }
    }
    ```

    `web/package.json`

    ```json
    {"name": "web"}
    ```

=== "After"

    `package.json`

    ```json
    {
      "dependencies": {
        "axios": "^1.7.9"
      }
    }
    ```

**Used in:** [`npm-package-bump`](https://github.com/rossedman/gh-fleet/tree/main/examples/npm-package-bump)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/npm.star#L78)
