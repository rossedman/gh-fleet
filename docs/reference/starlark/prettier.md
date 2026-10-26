---
title: prettier.star
---

# prettier.star

Set Prettier options in whichever config a repo uses.

Supports .prettierrc (JSON or YAML), .prettierrc.json, .prettierrc.yaml,
.prettierrc.yml and the `prettier` key of package.json. JavaScript
configs (prettier.config.js and friends) can't be edited safely and are
not supported.

```python
load("@fleet//prettier.star", "find", "get", "set_option")
```

| Function | What it does |
| --- | --- |
| [`find`](#find) | Finds the repo's Prettier config. |
| [`get`](#get) | Reads an option from the Prettier config. |
| [`set_option`](#set_option) | Sets an option in the Prettier config, keeping its format. |

## find

```python
find(repo)
```

Finds the repo's Prettier config.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The path of the first of .prettierrc, .prettierrc.json,
.prettierrc.yaml and .prettierrc.yml that exists, `package.json`
when it has a `prettier` key, or None.

**Example**

```python
find(repo)
# returns "package.json"
```

=== "Before"

    `package.json`

    ```json
    {"name": "web", "prettier": {"semi": false}}
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/prettier.star#L14)

## get

```python
get(repo, option)
```

Reads an option from the Prettier config.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `option` | The option, such as `printWidth`. |

**Returns:** The value, or None when it isn't set or there is no config. YAML
values come back as strings.

**Example**

```python
get(repo, "printWidth")
# returns "100"
```

=== "Before"

    `.prettierrc`

    ```text
    singleQuote: true
    printWidth: 100
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/prettier.star#L43)

## set_option

```python
set_option(repo, option, value)
```

Sets an option in the Prettier config, keeping its format.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `option` | The option, such as `printWidth` or `trailingComma`. |
| `value` | A string, number or boolean. |

**Returns:** True when the file changed; False when the repo has no config.

**Example**

```python
set_option(repo, "trailingComma", "all")
# returns True
```

=== "Before"

    `.prettierrc.json`

    ```json
    {
      "semi": false,
      "trailingComma": "es5"
    }
    ```

=== "After"

    `.prettierrc.json`

    ```json
    {
      "semi": false,
      "trailingComma": "all"
    }
    ```

**Used in:** [`prettier-eslint`](../examples/prettier-eslint.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/prettier.star#L69)
