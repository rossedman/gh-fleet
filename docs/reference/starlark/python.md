---
title: python.star
---

# python.star

Bump pinned Python requirements and the supported Python version.

Edits are line-based, so comments, ordering and markers in
requirements files and pyproject.toml stay as they were.

```python
load("@fleet//python.star", "requirements_files", "set_requirement", "requires_python", "set_requires_python")
```

| Function | What it does |
| --- | --- |
| [`requirements_files`](#requirements_files) | Lists the repo's requirements files. |
| [`set_requirement`](#set_requirement) | Sets a package's version specifier wherever it is already required. |
| [`requires_python`](#requires_python) | Reads the `requires-python` of a pyproject.toml. |
| [`set_requires_python`](#set_requires_python) | Sets `requires-python` in a pyproject.toml that already has it. |

## requirements_files

```python
requirements_files(repo)
```

Lists the repo's requirements files.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The sorted paths of every requirements*.txt, and every .txt in a
requirements directory, outside virtualenvs and node_modules.

**Example**

```python
requirements_files(repo)
# returns ["requirements.txt", "requirements/dev.txt"]
```

=== "Before"

    `requirements.txt`

    ```text
    flask
    ```

    `requirements/dev.txt`

    ```text
    pytest
    ```

    `.venv/requirements.txt`

    ```text
    flask
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/python.star#L11)

## set_requirement

```python
set_requirement(repo, name, spec)
```

Sets a package's version specifier wherever it is already required.

A newer version already there is kept, so a campaign never lowers one.

Changes requirements*.txt lines and the quoted entries of pyproject.toml
dependency lists, keeping extras such as `[socks]`, environment markers
and comments. Names match the way pip matches them: `Django`, `django`
and `DJANGO` are the same package. Files that do not mention the package
are left alone.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `name` | The package name. |
| `spec` | A full PEP 440 specifier, such as `==4.2.16` or `>=2.32,<3`. |

**Returns:** The list of paths that changed.

**Example**

```python
set_requirement(repo, "django", "==4.2.16")
# returns ["requirements.txt", "pyproject.toml"]
```

=== "Before"

    `requirements.txt`

    ```text
    Django==4.2.1  # web
    requests[socks]>=2.31
    ```

    `pyproject.toml`

    ```toml
    [project]
    dependencies = ["django>=4.0"]
    ```

=== "After"

    `requirements.txt`

    ```text
    Django==4.2.16  # web
    requests[socks]>=2.31
    ```

    `pyproject.toml`

    ```toml
    [project]
    dependencies = ["django==4.2.16"]
    ```

**Used in:** [`python-requirement-bump`](../examples/python-requirement-bump.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/python.star#L48)

## requires_python

```python
requires_python(repo, path = "pyproject.toml")
```

Reads the `requires-python` of a pyproject.toml.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `path` | The pyproject.toml to read. |

**Returns:** The specifier, such as `>=3.9`, or None.

**Example**

```python
requires_python(repo)
# returns ">=3.9"
```

=== "Before"

    `pyproject.toml`

    ```toml
    [project]
    requires-python = ">=3.9"
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/python.star#L94)

## set_requires_python

```python
set_requires_python(repo, spec, path = "pyproject.toml")
```

Sets `requires-python` in a pyproject.toml that already has it.

A newer version already there is kept, so a campaign never lowers one.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `spec` | The new specifier, such as `>=3.12`. |
| `path` | The pyproject.toml to edit. |

**Returns:** True when the file changed.

**Example**

```python
set_requires_python(repo, ">=3.12")
# returns True
```

=== "Before"

    `pyproject.toml`

    ```toml
    [project]
    requires-python = ">=3.9"
    ```

=== "After"

    `pyproject.toml`

    ```toml
    [project]
    requires-python = ">=3.12"
    ```

**Used in:** [`python-version`](../examples/python-version.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/python.star#L113)
