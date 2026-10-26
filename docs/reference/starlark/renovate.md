---
title: renovate.star
---

# renovate.star

Edit Renovate configs: presets, package rules, schedules and other settings.

Works on JSON configs (renovate.json, .github/renovate.json, .renovaterc
and .renovaterc.json), keeping key order and indentation. JSON5 configs
can't be edited this way and are skipped by `find`.

```python
load("@fleet//renovate.star", "find", "add_preset", "remove_preset", "replace_preset", "add_package_rule", "set_option")
```

| Function | What it does |
| --- | --- |
| [`find`](#find) | Finds the repo's Renovate config. |
| [`add_preset`](#add_preset) | Adds a preset to `extends` unless it is already there. |
| [`remove_preset`](#remove_preset) | Removes a preset from `extends`, such as a retired `config:base`. |
| [`replace_preset`](#replace_preset) | Replaces a preset in `extends` where it is, such as `config:base` with `config:recommended`. |
| [`add_package_rule`](#add_package_rule) | Appends a rule to `packageRules` unless an identical rule is already there. |
| [`set_option`](#set_option) | Sets a top-level or nested option, such as `schedule` or `lockFileMaintenance.enabled`. |

## find

```python
find(repo)
```

Finds the repo's Renovate config.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The path of the first JSON config that exists, or None.

**Example**

```python
find(repo)
# returns ".github/renovate.json"
```

=== "Before"

    `.github/renovate.json`

    ```json
    {}
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/renovate.star#L12)

## add_preset

```python
add_preset(repo, preset)
```

Adds a preset to `extends` unless it is already there.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `preset` | The preset, such as `config:recommended` or `github>acme/renovate-config`. |

**Returns:** True when the file changed.

**Example**

```python
add_preset(repo, "github>acme/renovate-config")
# returns True
```

=== "Before"

    `renovate.json`

    ```json
    {
      "extends": ["config:recommended"]
    }
    ```

=== "After"

    `renovate.json`

    ```json
    {
      "extends": ["config:recommended", "github>acme/renovate-config"]
    }
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/renovate.star#L30)

## remove_preset

```python
remove_preset(repo, preset)
```

Removes a preset from `extends`, such as a retired `config:base`.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `preset` | The preset to remove. |

**Returns:** True when the file changed.

**Example**

```python
remove_preset(repo, "config:base")
# returns True
```

=== "Before"

    `renovate.json`

    ```json
    {"extends": ["config:base", ":semanticCommits"]}
    ```

=== "After"

    `renovate.json`

    ```json
    {"extends": [":semanticCommits"]}
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/renovate.star#L60)

## replace_preset

```python
replace_preset(repo, old, new)
```

Replaces a preset in `extends` where it is, such as `config:base` with `config:recommended`.

When `new` is already there too, `old` is just removed.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `old` | The preset to replace. |
| `new` | The preset to put in its place. |

**Returns:** True when the file changed.

**Example**

```python
replace_preset(repo, "config:base", "config:recommended")
# returns True
```

=== "Before"

    `renovate.json`

    ```json
    {
      "extends": ["config:base", ":semanticCommits"]
    }
    ```

=== "After"

    `renovate.json`

    ```json
    {
      "extends": ["config:recommended", ":semanticCommits"]
    }
    ```

**Used in:** [`renovate-automerge`](../examples/renovate-automerge.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/renovate.star#L85)

## add_package_rule

```python
add_package_rule(repo, rule)
```

Appends a rule to `packageRules` unless an identical rule is already there.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `rule` | The rule, a dict such as `{"matchUpdateTypes": ["patch"], "automerge": True}`. |

**Returns:** True when the file changed.

**Example**

```python
add_package_rule(repo, {"matchDepTypes": ["devDependencies"], "automerge": True})
# returns True
```

=== "Before"

    `renovate.json`

    ```json
    {
      "extends": ["config:recommended"]
    }
    ```

=== "After"

    `renovate.json`

    ```json
    {
      "extends": ["config:recommended"],
      "packageRules": [
        {
          "matchDepTypes": ["devDependencies"],
          "automerge": true
        }
      ]
    }
    ```

**Used in:** [`renovate-automerge`](../examples/renovate-automerge.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/renovate.star#L122)

## set_option

```python
set_option(repo, key, value)
```

Sets a top-level or nested option, such as `schedule` or `lockFileMaintenance.enabled`.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `key` | The dotted option path. |
| `value` | Any JSON value. |

**Returns:** True when the file changed; False when the repo has no config.

**Example**

```python
set_option(repo, "schedule", ["before 6am on monday"])
# returns True
```

=== "Before"

    `renovate.json`

    ```json
    {
      "extends": ["config:recommended"],
      "schedule": ["at any time"]
    }
    ```

=== "After"

    `renovate.json`

    ```json
    {
      "extends": ["config:recommended"],
      "schedule": ["before 6am on monday"]
    }
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/renovate.star#L158)
