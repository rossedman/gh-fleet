---
title: golangci.star
---

# golangci.star

Edit golangci-lint configs: enable and disable linters and set linter settings.

Works on YAML configs (.golangci.yml and .golangci.yaml), in both the v1
and v2 formats; settings go where each version expects them. Edits are
line-based, so comments stay. To move a config from v1 to v2, run the
official `golangci-lint migrate` in a `run` step: it knows every renamed
and merged linter, which a text edit can't.

```python
load("@fleet//golangci.star", "find", "config_version", "enable_linter", "disable_linter", "set_setting")
```

| Function | What it does |
| --- | --- |
| [`find`](#find) | Finds the repo's golangci-lint config. |
| [`config_version`](#config_version) | Reads the config format version. |
| [`enable_linter`](#enable_linter) | Enables a linter, removing it from `disable` if it was there. |
| [`disable_linter`](#disable_linter) | Disables a linter, removing it from `enable` if it was there. |
| [`set_setting`](#set_setting) | Sets a linter's setting, in `linters-settings` (v1) or `linters.settings` (v2). |

## find

```python
find(repo)
```

Finds the repo's golangci-lint config.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The path of .golangci.yml or .golangci.yaml, or None.

**Example**

```python
find(repo)
# returns ".golangci.yaml"
```

=== "Before"

    `.golangci.yaml`

    ```yaml
    linters: {}
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/golangci.star#L14)

## config_version

```python
config_version(repo)
```

Reads the config format version.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** `"2"` for a v2 config, `"1"` for a config without a `version` key, or
None when the repo has no config.

**Example**

```python
config_version(repo)
# returns "2"
```

=== "Before"

    `.golangci.yml`

    ```yaml
    version: "2"
    linters:
      default: standard
    ```

**Used in:** [`golangci-v2`](https://github.com/rossedman/gh-fleet/tree/main/examples/golangci-v2)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/golangci.star#L32)

## enable_linter

```python
enable_linter(repo, name)
```

Enables a linter, removing it from `disable` if it was there.

The linter is added to `linters.enable`, as a block or flow list
(whichever the file uses), which is created if missing.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `name` | The linter, such as `gosec`. |

**Returns:** True when the file changed.

**Example**

```python
enable_linter(repo, "gosec")
# returns True
```

=== "Before"

    `.golangci.yml`

    ```yaml
    linters:
      enable:
        - errcheck  # always
        - govet
      disable:
        - gosec
    ```

=== "After"

    `.golangci.yml`

    ```yaml
    linters:
      enable:
        - errcheck  # always
        - govet
        - gosec
    ```

**Used in:** [`golangci-v2`](https://github.com/rossedman/gh-fleet/tree/main/examples/golangci-v2)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/golangci.star#L128)

## disable_linter

```python
disable_linter(repo, name)
```

Disables a linter, removing it from `enable` if it was there.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `name` | The linter, such as `lll`. |

**Returns:** True when the file changed.

**Example**

```python
disable_linter(repo, "lll")
# returns True
```

=== "Before"

    `.golangci.yml`

    ```yaml
    linters:
      enable: [errcheck, lll, govet]
    ```

=== "After"

    `.golangci.yml`

    ```yaml
    linters:
      enable: [errcheck, govet]
      disable: [lll]
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/golangci.star#L159)

## set_setting

```python
set_setting(repo, linter, key, value)
```

Sets a linter's setting, in `linters-settings` (v1) or `linters.settings` (v2).

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `linter` | The linter, such as `govet`. |
| `key` | The setting, dotted for nested keys, such as `enable-all`. |
| `value` | A string, number, boolean or list. |

**Returns:** True when the file changed.

**Example**

```python
set_setting(repo, "lll", "line-length", 140)
# returns True
```

=== "Before"

    `.golangci.yml`

    ```yaml
    version: "2"
    linters:
      settings:
        lll:
          line-length: 120
    ```

=== "After"

    `.golangci.yml`

    ```yaml
    version: "2"
    linters:
      settings:
        lll:
          line-length: 140
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/golangci.star#L197)
