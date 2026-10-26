---
title: codeowners.star
---

# codeowners.star

Find and edit CODEOWNERS files.

```python
load("@fleet//codeowners.star", "find", "owners", "set_owners")
```

| Function | What it does |
| --- | --- |
| [`find`](#find) | Finds the CODEOWNERS file GitHub uses. |
| [`owners`](#owners) | Reads the owners of a CODEOWNERS pattern. |
| [`set_owners`](#set_owners) | Makes a list of users or teams the owners of a pattern. |

## find

```python
find(repo)
```

Finds the CODEOWNERS file GitHub uses.

GitHub reads the first of .github/CODEOWNERS, CODEOWNERS and
docs/CODEOWNERS that exists.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** Its path, or None when the repo has none.

**Example**

```python
find(repo)
# returns "docs/CODEOWNERS"
```

=== "Before"

    `docs/CODEOWNERS`

    ```text
    * @acme/docs
    ```

**Used in:** [`add-codeowners`](../examples/add-codeowners.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/codeowners.star#L5)

## owners

```python
owners(repo, pattern = "*")
```

Reads the owners of a CODEOWNERS pattern.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `pattern` | The pattern exactly as written, such as `*` or `/docs/`. |

**Returns:** The owners of the last rule for the pattern, or an empty list.

**Example**

```python
owners(repo, "/docs/")
# returns ["@acme/writers"]
```

=== "Before"

    `.github/CODEOWNERS`

    ```text
    * @acme/platform
    /docs/ @acme/writers
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/codeowners.star#L31)

## set_owners

```python
set_owners(repo, pattern, owners, header = None)
```

Makes a list of users or teams the owners of a pattern.

The last existing rule for the pattern is replaced; otherwise a rule is
added at the end. With no CODEOWNERS file, one is created at
.github/CODEOWNERS.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `pattern` | The pattern, such as `*` or `/docs/`. |
| `owners` | A list or a space-separated string of users or teams; a leading `@` is added when missing. |
| `header` | A line, such as a comment, to start a new file with. |

**Returns:** True when the file changed.

**Example**

```python
set_owners(repo, "*", "acme/payments")
# returns True
```

=== "Before"

    `.github/CODEOWNERS`

    ```text
    * @acme/platform
    /docs/ @acme/writers
    ```

=== "After"

    `.github/CODEOWNERS`

    ```text
    * @acme/payments
    /docs/ @acme/writers
    ```

**Used in:** [`add-codeowners`](../examples/add-codeowners.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/codeowners.star#L57)
