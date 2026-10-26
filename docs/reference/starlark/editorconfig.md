---
title: editorconfig.star
---

# editorconfig.star

Read and set properties in .editorconfig, line by line.

Sections are named as in the file, such as `*` or `*.{js,ts}`; use `""`
for the preamble before the first section, where `root = true` goes.

```python
load("@fleet//editorconfig.star", "get", "set")
```

| Function | What it does |
| --- | --- |
| [`get`](#get) | Reads a property. |
| [`set`](#set) | Sets a property, adding it, or its section, when missing. |

## get

```python
get(repo, section, key)
```

Reads a property.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `section` | The section name, such as `*` or `*.go`, or `""` for the preamble. |
| `key` | The property name. |

**Returns:** The value as written, or None.

**Example**

```python
get(repo, "*.go", "indent_style")
# returns "tab"
```

=== "Before"

    `.editorconfig`

    ```text
    root = true

    [*]
    indent_style = space
    indent_size = 2

    [*.go]
    indent_style = tab
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/editorconfig.star#L21)

## set

```python
set(repo, section, key, value)
```

Sets a property, adding it, or its section, when missing.

A new property goes after the section's last property; a new section
goes at the end of the file. The file is created if needed.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `section` | The section name, or `""` for the preamble. |
| `key` | The property name. |
| `value` | The value, such as `lf` or `4`; booleans become `true` and `false`. |

**Returns:** True when the file changed.

**Example**

```python
set(repo, "*", "end_of_line", "lf")
set(repo, "Makefile", "indent_style", "tab")
# returns True
```

=== "Before"

    `.editorconfig`

    ```text
    root = true

    [*]
    indent_style = space
    ```

=== "After"

    `.editorconfig`

    ```text
    root = true

    [*]
    indent_style = space
    end_of_line = lf

    [Makefile]
    indent_style = tab
    ```

**Used in:** [`editor-defaults`](../examples/editor-defaults.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/editorconfig.star#L55)
