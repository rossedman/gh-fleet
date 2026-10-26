---
title: eslint.star
---

# eslint.star

Set rules and presets in ESLint's JSON and YAML configs.

Supports the legacy (eslintrc) format: .eslintrc.json, .eslintrc,
.eslintrc.yaml, .eslintrc.yml and the `eslintConfig` key of package.json.
Flat configs (eslint.config.js) are JavaScript and can't be edited
safely; `find` reports them so a campaign can skip or flag those repos.

```python
load("@fleet//eslint.star", "find", "set_rule", "add_extends")
```

| Function | What it does |
| --- | --- |
| [`find`](#find) | Finds the repo's ESLint config. |
| [`set_rule`](#set_rule) | Sets a rule's level, and optionally its options. |
| [`add_extends`](#add_extends) | Adds a preset to `extends` unless it is already there. |

## find

```python
find(repo)
```

Finds the repo's ESLint config.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The path of the first JSON or YAML config that exists, `package.json`
when it has an `eslintConfig` key, the path of a flat config when the
repo uses one, or None.

**Example**

```python
find(repo)
# returns "eslint.config.mjs"
```

=== "Before"

    `eslint.config.mjs`

    ```text
    export default []
    ```

**Used in:** [`prettier-eslint`](https://github.com/rossedman/gh-fleet/tree/main/examples/prettier-eslint)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/eslint.star#L15)

## set_rule

```python
set_rule(repo, rule, level, options = None)
```

Sets a rule's level, and optionally its options.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `rule` | The rule, such as `no-console` or `@typescript-eslint/no-unused-vars`. |
| `level` | `off`, `warn` or `error` (or 0, 1, 2). |
| `options` | The rule's options, written after the level as in `["error", {"allow": ["warn"]}]`; JSON configs only. |

**Returns:** True when the file changed; False when the repo has no editable
config.

**Example**

```python
set_rule(repo, "no-console", "error", {"allow": ["warn", "error"]})
# returns True
```

=== "Before"

    `.eslintrc.json`

    ```json
    {
      "extends": ["eslint:recommended"],
      "rules": {
        "no-console": "warn"
      }
    }
    ```

=== "After"

    `.eslintrc.json`

    ```json
    {
      "extends": [
        "eslint:recommended"
      ],
      "rules": {
        "no-console": [
          "error",
          {
            "allow": [
              "warn",
              "error"
            ]
          }
        ]
      }
    }
    ```

**Used in:** [`prettier-eslint`](https://github.com/rossedman/gh-fleet/tree/main/examples/prettier-eslint)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/eslint.star#L50)

## add_extends

```python
add_extends(repo, preset)
```

Adds a preset to `extends` unless it is already there.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `preset` | The preset, such as `plugin:security/recommended`. |

**Returns:** True when the file changed; False when the repo has no editable
JSON config.

**Example**

```python
add_extends(repo, "plugin:security/recommended")
# returns True
```

=== "Before"

    `.eslintrc.json`

    ```json
    {"extends": "eslint:recommended"}
    ```

=== "After"

    `.eslintrc.json`

    ```json
    {
      "extends": [
        "eslint:recommended",
        "plugin:security/recommended"
      ]
    }
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/eslint.star#L102)
