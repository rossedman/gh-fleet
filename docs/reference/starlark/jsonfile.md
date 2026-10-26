---
title: jsonfile.star
---

# jsonfile.star

Read and edit any JSON file, keeping key order and indentation.

For files such as renovate.json, tsconfig.json without comments,
.eslintrc.json and composer.json. Keys are addressed with dots, such as
`compilerOptions.strict`. Files with comments (JSONC) can't be decoded and
fail the run.

```python
load("@fleet//jsonfile.star", "read", "write", "get", "set", "delete")
```

| Function | What it does |
| --- | --- |
| [`read`](#read) | Reads a JSON file into a value whose dict keys keep the file's order. |
| [`write`](#write) | Writes a value as JSON, using the indentation the file already has. |
| [`get`](#get) | Reads a value by dotted key path. |
| [`set`](#set) | Sets a value by dotted key path, adding missing objects on the way. |
| [`delete`](#delete) | Removes a key by dotted key path. |

## read

```python
read(repo, path, default = None)
```

Reads a JSON file into a value whose dict keys keep the file's order.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `path` | The JSON file, relative to the repo root. |
| `default` | What to return when the file does not exist; without it, a missing file fails the run. |

**Returns:** The decoded value.

**Example**

```python
read(repo, "renovate.json")["extends"]
# returns ["config:base"]
```

=== "Before"

    `renovate.json`

    ```json
    {"extends": ["config:base"], "schedule": ["weekly"]}
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/jsonfile.star#L13)

## write

```python
write(repo, path, data)
```

Writes a value as JSON, using the indentation the file already has.

A new file gets two spaces. Output ends with a newline.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `path` | The JSON file, relative to the repo root. |
| `data` | The value to write, usually from `read`. |

**Returns:** True when the file changed.

**Example**

```python
config = read(repo, ".eslintrc.json")
config["extends"] = ["eslint:recommended"]
write(repo, ".eslintrc.json", config)
# returns True
```

=== "Before"

    `.eslintrc.json`

    ```json
    {
        "root": true
    }
    ```

=== "After"

    `.eslintrc.json`

    ```json
    {
        "root": true,
        "extends": [
            "eslint:recommended"
        ]
    }
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/jsonfile.star#L37)

## get

```python
get(repo, path, key, default = None)
```

Reads a value by dotted key path.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `path` | The JSON file, relative to the repo root. |
| `key` | The dotted key path, such as `compilerOptions.strict`. |
| `default` | What to return when the file or key is missing. |

**Returns:** The value, or `default`.

**Example**

```python
get(repo, "tsconfig.json", "compilerOptions.target")
# returns "ES2020"
```

=== "Before"

    `tsconfig.json`

    ```json
    {"compilerOptions": {"strict": false, "target": "ES2020"}}
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/jsonfile.star#L73)

## set

```python
set(repo, path, key, value)
```

Sets a value by dotted key path, adding missing objects on the way.

The file is created if needed. Existing keys keep their place; new keys
go at the end of their object.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `path` | The JSON file, relative to the repo root. |
| `key` | The dotted key path, such as `compilerOptions.strict`. |
| `value` | Any JSON value: string, number, boolean, None, list or dict. |

**Returns:** True when the file changed.

**Example**

```python
set(repo, "tsconfig.json", "compilerOptions.strict", True)
# returns True
```

=== "Before"

    `tsconfig.json`

    ```json
    {
      "compilerOptions": {
        "strict": false
      }
    }
    ```

=== "After"

    `tsconfig.json`

    ```json
    {
      "compilerOptions": {
        "strict": true
      }
    }
    ```

**Used in:** [`renovate-preset`](https://github.com/rossedman/gh-fleet/tree/main/examples/renovate-preset)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/jsonfile.star#L97)

## delete

```python
delete(repo, path, key)
```

Removes a key by dotted key path.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `path` | The JSON file, relative to the repo root. |
| `key` | The dotted key path. |

**Returns:** True when the file changed.

**Example**

```python
delete(repo, "package.json", "scripts.postinstall")
# returns True
```

=== "Before"

    `package.json`

    ```json
    {
      "name": "web",
      "scripts": {
        "test": "jest",
        "postinstall": "husky install"
      }
    }
    ```

=== "After"

    `package.json`

    ```json
    {
      "name": "web",
      "scripts": {
        "test": "jest"
      }
    }
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/jsonfile.star#L143)
