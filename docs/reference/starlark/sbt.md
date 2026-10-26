---
title: sbt.star
---

# sbt.star

Bump sbt, Scala, library dependencies and sbt plugins.

Edits are line-based, so comments and layout in build.sbt and the
project/ directory stay as they were.

```python
load("@fleet//sbt.star", "build_files", "set_sbt_version", "set_dependency", "set_val", "set_scala_version")
```

| Function | What it does |
| --- | --- |
| [`build_files`](#build_files) | Lists the sbt build files. |
| [`set_sbt_version`](#set_sbt_version) | Sets `sbt.version` in project/build.properties. |
| [`set_dependency`](#set_dependency) | Sets the version of a library or sbt plugin wherever the build declares it. |
| [`set_val`](#set_val) | Sets a string `val`, such as `val catsVersion = "2.9.0"`, wherever it is defined. |
| [`set_scala_version`](#set_scala_version) | Sets `scalaVersion`, and matching entries of `crossScalaVersions`. |

## build_files

```python
build_files(repo)
```

Lists the sbt build files.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The sorted paths of every *.sbt file, and every .scala and .sbt
file under a project directory, outside target directories.

**Example**

```python
build_files(repo)
# returns ["build.sbt", "project/Dependencies.scala", "project/plugins.sbt"]
```

=== "Before"

    `build.sbt`

    ```text
    name := "api"
    ```

    `project/plugins.sbt`

    ```text
    addSbtPlugin("x" % "y" % "1")
    ```

    `project/Dependencies.scala`

    ```text
    object Dependencies
    ```

**Used in:** [`sbt-scala-bump`](https://github.com/rossedman/gh-fleet/tree/main/examples/sbt-scala-bump)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/sbt.star#L10)

## set_sbt_version

```python
set_sbt_version(repo, version)
```

Sets `sbt.version` in project/build.properties.

A newer version already there is kept, so a campaign never lowers one.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `version` | The sbt version, such as `1.10.2`. |

**Returns:** True when the file changed.

**Example**

```python
set_sbt_version(repo, "1.10.2")
# returns True
```

=== "Before"

    `project/build.properties`

    ```text
    sbt.version=1.9.7
    ```

=== "After"

    `project/build.properties`

    ```text
    sbt.version=1.10.2
    ```

**Used in:** [`sbt-scala-bump`](https://github.com/rossedman/gh-fleet/tree/main/examples/sbt-scala-bump)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/sbt.star#L30)

## set_dependency

```python
set_dependency(repo, organization, name, version)
```

Sets the version of a library or sbt plugin wherever the build declares it.

A newer version already there is kept, so a campaign never lowers one.

Covers `"org" % "name" % "v"` and `"org" %% "name" % "v"` (with any
configuration such as `% Test` after it), and `addSbtPlugin(...)`.
Versions held in a `val` are left alone; set the `val` with
`set_val`.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `organization` | The group, such as `org.typelevel`. |
| `name` | The artifact name, without a Scala suffix. |
| `version` | The new version. |

**Returns:** The list of paths that changed.

**Example**

```python
set_dependency(repo, "org.typelevel", "cats-core", "2.12.0")
# returns ["build.sbt"]
```

=== "Before"

    `build.sbt`

    ```text
    libraryDependencies ++= Seq(
      "org.typelevel" %% "cats-core" % "2.9.0",
      "org.scalatest" %% "scalatest" % "3.2.15" % Test
    )
    ```

=== "After"

    `build.sbt`

    ```text
    libraryDependencies ++= Seq(
      "org.typelevel" %% "cats-core" % "2.12.0",
      "org.scalatest" %% "scalatest" % "3.2.15" % Test
    )
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/sbt.star#L59)

## set_val

```python
set_val(repo, name, value)
```

Sets a string `val`, such as `val catsVersion = "2.9.0"`, wherever it is defined.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `name` | The val's name. |
| `value` | The new string value. |

**Returns:** The list of paths that changed.

**Example**

```python
set_val(repo, "catsVersion", "2.12.0")
# returns ["project/Dependencies.scala"]
```

=== "Before"

    `project/Dependencies.scala`

    ```text
    object Dependencies {
      val catsVersion = "2.9.0"
    }
    ```

=== "After"

    `project/Dependencies.scala`

    ```text
    object Dependencies {
      val catsVersion = "2.12.0"
    }
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/sbt.star#L101)

## set_scala_version

```python
set_scala_version(repo, version, from_version = None)
```

Sets `scalaVersion`, and matching entries of `crossScalaVersions`.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `version` | The Scala version, such as `2.13.15`. |
| `from_version` | When set, only this version is replaced, so a cross-built project keeps its other Scala versions. |

**Returns:** The list of paths that changed.

**Example**

```python
set_scala_version(repo, "2.13.15", from_version = "2.13.12")
# returns ["build.sbt"]
```

=== "Before"

    `build.sbt`

    ```text
    ThisBuild / scalaVersion := "2.13.12"
    crossScalaVersions := Seq("2.13.12", "3.3.1")
    ```

=== "After"

    `build.sbt`

    ```text
    ThisBuild / scalaVersion := "2.13.15"
    crossScalaVersions := Seq("2.13.15", "3.3.1")
    ```

**Used in:** [`sbt-scala-bump`](https://github.com/rossedman/gh-fleet/tree/main/examples/sbt-scala-bump)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/sbt.star#L133)
