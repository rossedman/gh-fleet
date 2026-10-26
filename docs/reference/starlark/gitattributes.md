---
title: gitattributes.star
---

# gitattributes.star

Set attributes for path patterns in .gitattributes, such as line endings and Git LFS tracking.

```python
load("@fleet//gitattributes.star", "set", "track_lfs")
```

| Function | What it does |
| --- | --- |
| [`set`](#set) | Gives a pattern attributes, merging with any it already has. |
| [`track_lfs`](#track_lfs) | Tracks a pattern with Git LFS, as `git lfs track` would. |

## set

```python
set(repo, pattern, attributes)
```

Gives a pattern attributes, merging with any it already has.

An attribute already on the pattern's line is replaced by the new one
with the same name, so `eol=crlf` becomes `eol=lf` and `text` becomes
`-text`; other attributes stay. A pattern with no line gets one at the
end. The file is created if needed.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `pattern` | The path pattern, such as `*.sh` or `*`. |
| `attributes` | The attributes, as a space-separated string or a list, such as `"text eol=lf"`. |

**Returns:** True when the file changed.

**Example**

```python
set(repo, "*.sh", "eol=lf")
set(repo, "*.png", "binary")
# returns True
```

=== "Before"

    `.gitattributes`

    ```text
    * text=auto
    *.sh text eol=crlf
    ```

=== "After"

    `.gitattributes`

    ```text
    * text=auto
    *.sh text eol=lf
    *.png binary
    ```

**Used in:** [`editor-defaults`](../examples/editor-defaults.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/gitattributes.star#L8)

## track_lfs

```python
track_lfs(repo, pattern)
```

Tracks a pattern with Git LFS, as `git lfs track` would.

This only edits .gitattributes; files already committed stay in Git
until they are migrated.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `pattern` | The path pattern, such as `*.psd`. |

**Returns:** True when the file changed.

**Example**

```python
track_lfs(repo, "*.psd")
# returns True
```

=== "Before"

    `.gitattributes`

    ```text
    * text=auto
    ```

=== "After"

    `.gitattributes`

    ```text
    * text=auto
    *.psd filter=lfs diff=lfs merge=lfs -text
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/gitattributes.star#L58)
