---
title: bazel.star
---

# bazel.star

Bump Bazel itself, Bzlmod module dependencies and http_archive downloads.

Edits are line-based, so comments and formatting in MODULE.bazel,
WORKSPACE and .bzl files stay as they were. MODULE.bazel.lock isn't
updated; run `bazel mod deps --lockfile_mode=update` in a `run` step when
it is committed.

```python
load("@fleet//bazel.star", "version", "set_version", "set_dep", "set_archive")
```

| Function | What it does |
| --- | --- |
| [`version`](#version) | Reads the Bazel version pinned in .bazelversion. |
| [`set_version`](#set_version) | Sets the Bazel version in .bazelversion, when the repo pins one. |
| [`set_dep`](#set_dep) | Sets the version of a `bazel_dep` in every MODULE.bazel that declares it. |
| [`set_archive`](#set_archive) | Points an `http_archive` at a new download, with its checksum. |

## version

```python
version(repo)
```

Reads the Bazel version pinned in .bazelversion.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The version, or None when the repo has no .bazelversion.

**Example**

```python
version(repo)
# returns "7.1.1"
```

=== "Before"

    `.bazelversion`

    ```text
    7.1.1
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/bazel.star#L12)

## set_version

```python
set_version(repo, version)
```

Sets the Bazel version in .bazelversion, when the repo pins one.

A newer version already there is kept, so a campaign never lowers one.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `version` | The Bazel version, such as `7.4.1`. |

**Returns:** True when the file changed.

**Example**

```python
set_version(repo, "7.4.1")
# returns True
```

=== "Before"

    `.bazelversion`

    ```text
    6.4.0
    ```

=== "After"

    `.bazelversion`

    ```text
    7.4.1
    ```

**Used in:** [`bazel-module-bump`](https://github.com/rossedman/gh-fleet/tree/main/examples/bazel-module-bump)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/bazel.star#L28)

## set_dep

```python
set_dep(repo, name, version)
```

Sets the version of a `bazel_dep` in every MODULE.bazel that declares it.

A newer version already there is kept, so a campaign never lowers one.

The call may span lines and its arguments may come in any order.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `name` | The module name, such as `rules_go`. |
| `version` | The new version. |

**Returns:** The list of paths that changed.

**Example**

```python
set_dep(repo, "gazelle", "0.39.1")
set_dep(repo, "rules_go", "0.50.1")
# returns ["MODULE.bazel"]
```

=== "Before"

    `MODULE.bazel`

    ```text
    module(name = "app")

    bazel_dep(name = "rules_go", version = "0.46.0")
    bazel_dep(
        name = "gazelle",
        version = "0.35.0",
    )
    ```

=== "After"

    `MODULE.bazel`

    ```text
    module(name = "app")

    bazel_dep(name = "rules_go", version = "0.50.1")
    bazel_dep(
        name = "gazelle",
        version = "0.39.1",
    )
    ```

**Used in:** [`bazel-module-bump`](https://github.com/rossedman/gh-fleet/tree/main/examples/bazel-module-bump)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/bazel.star#L56)

## set_archive

```python
set_archive(repo, name, urls, sha256, strip_prefix = None)
```

Points an `http_archive` at a new download, with its checksum.

Sets `urls` (or `url`), `sha256` and, when given, `strip_prefix` of the
`http_archive` with this name in WORKSPACE files and .bzl files. Get
the checksum from the release page, or from `sha256sum` on the
download; the run fails rather than leave an archive without one.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `name` | The repository name, as in `http_archive(name = ...)`. |
| `urls` | The new download URL, or a list of mirrors. |
| `sha256` | The SHA-256 of the download. |
| `strip_prefix` | The new `strip_prefix`, or None to leave it. |

**Returns:** The list of paths that changed.

**Example**

```python
set_archive(repo, "rules_python", "https://github.com/bazelbuild/rules_python/releases/download/0.36.0/rules_python-0.36.0.tar.gz", "bbbb", strip_prefix = "rules_python-0.36.0")
# returns ["WORKSPACE"]
```

=== "Before"

    `WORKSPACE`

    ```text
    http_archive(
        name = "rules_python",
        sha256 = "aaaa",
        strip_prefix = "rules_python-0.27.0",
        url = "https://github.com/bazelbuild/rules_python/releases/download/0.27.0/rules_python-0.27.0.tar.gz",
    )
    ```

=== "After"

    `WORKSPACE`

    ```text
    http_archive(
        name = "rules_python",
        sha256 = "bbbb",
        strip_prefix = "rules_python-0.36.0",
        url = "https://github.com/bazelbuild/rules_python/releases/download/0.36.0/rules_python-0.36.0.tar.gz",
    )
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/bazel.star#L116)
