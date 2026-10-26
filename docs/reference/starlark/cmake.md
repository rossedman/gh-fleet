---
title: cmake.star
---

# cmake.star

Bump CMake's minimum version and FetchContent dependency tags.

Edits are line-based, so comments and formatting in CMakeLists.txt stay
as they were.

```python
load("@fleet//cmake.star", "files", "minimum_version", "set_minimum_version", "set_fetchcontent_tag")
```

| Function | What it does |
| --- | --- |
| [`files`](#files) | Lists the repo's CMake files. |
| [`minimum_version`](#minimum_version) | Reads the version from `cmake_minimum_required`. |
| [`set_minimum_version`](#set_minimum_version) | Raises `cmake_minimum_required` to a version, never lowering it. |
| [`set_fetchcontent_tag`](#set_fetchcontent_tag) | Sets the `GIT_TAG` of a `FetchContent_Declare` dependency. |

## files

```python
files(repo)
```

Lists the repo's CMake files.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The sorted paths of every CMakeLists.txt and *.cmake file outside
build directories.

**Example**

```python
files(repo)
# returns ["CMakeLists.txt", "cmake/deps.cmake"]
```

=== "Before"

    `CMakeLists.txt`

    ```text
    project(app)
    ```

    `cmake/deps.cmake`

    ```text
    include(FetchContent)
    ```

    `build/CMakeLists.txt`

    ```text
    generated
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/cmake.star#L10)

## minimum_version

```python
minimum_version(repo, path = "CMakeLists.txt")
```

Reads the version from `cmake_minimum_required`.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `path` | The CMake file to read. |

**Returns:** The minimum version, such as `3.16`, or None.

**Example**

```python
minimum_version(repo)
# returns "3.16"
```

=== "Before"

    `CMakeLists.txt`

    ```text
    cmake_minimum_required(VERSION 3.16...3.28)
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/cmake.star#L30)

## set_minimum_version

```python
set_minimum_version(repo, version)
```

Raises `cmake_minimum_required` to a version, never lowering it.

A `...max` policy range, such as `3.16...3.28`, keeps its upper end.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `version` | The new minimum, such as `3.20`. |

**Returns:** The list of paths that changed.

**Example**

```python
set_minimum_version(repo, "3.20")
# returns ["CMakeLists.txt"]
```

=== "Before"

    `CMakeLists.txt`

    ```text
    cmake_minimum_required(VERSION 3.10)
    project(app)
    ```

=== "After"

    `CMakeLists.txt`

    ```text
    cmake_minimum_required(VERSION 3.20)
    project(app)
    ```

**Used in:** [`cmake-minimum`](../examples/cmake-minimum.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/cmake.star#L48)

## set_fetchcontent_tag

```python
set_fetchcontent_tag(repo, name, tag)
```

Sets the `GIT_TAG` of a `FetchContent_Declare` dependency.

A newer version already there is kept, so a campaign never lowers one.

The declaration may span several lines. Declarations of other names,
and ones without a `GIT_TAG`, are left alone.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `name` | The dependency's name, the first argument of `FetchContent_Declare`. |
| `tag` | The new tag or commit. |

**Returns:** The list of paths that changed.

**Example**

```python
set_fetchcontent_tag(repo, "googletest", "v1.15.2")
# returns ["CMakeLists.txt"]
```

=== "Before"

    `CMakeLists.txt`

    ```text
    FetchContent_Declare(
      googletest
      GIT_REPOSITORY https://github.com/google/googletest.git
      GIT_TAG        v1.13.0  # pinned
    )
    ```

=== "After"

    `CMakeLists.txt`

    ```text
    FetchContent_Declare(
      googletest
      GIT_REPOSITORY https://github.com/google/googletest.git
      GIT_TAG        v1.15.2  # pinned
    )
    ```

**Used in:** [`cmake-minimum`](../examples/cmake-minimum.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/cmake.star#L80)
