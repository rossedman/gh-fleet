---
title: gradle.star
---

# gradle.star

Bump Gradle dependencies, version catalog entries and the Gradle wrapper.

Edits are line-based, so comments and formatting in build scripts and
libs.versions.toml stay as they were.

```python
load("@fleet//gradle.star", "build_files", "set_dependency", "wrapper_version", "set_wrapper_version")
```

| Function | What it does |
| --- | --- |
| [`build_files`](#build_files) | Lists the build scripts and version catalogs. |
| [`set_dependency`](#set_dependency) | Sets a library's version wherever the build declares it. |
| [`wrapper_version`](#wrapper_version) | Reads the Gradle version the wrapper downloads. |
| [`set_wrapper_version`](#set_wrapper_version) | Points the Gradle wrapper at a new version, never lowering it. |

## build_files

```python
build_files(repo)
```

Lists the build scripts and version catalogs.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The sorted paths of every build.gradle, build.gradle.kts,
settings.gradle(.kts) and *.versions.toml outside build directories.

**Example**

```python
build_files(repo)
# returns ["app/build.gradle", "build.gradle.kts", "gradle/libs.versions.toml"]
```

=== "Before"

    `build.gradle.kts`

    ```text
    plugins { java }
    ```

    `app/build.gradle`

    ```text
    apply plugin: 'java'
    ```

    `gradle/libs.versions.toml`

    ```toml
    [versions]
    ```

    `build/tmp/build.gradle`

    ```text
    generated
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/gradle.star#L12)

## set_dependency

```python
set_dependency(repo, module, version)
```

Sets a library's version wherever the build declares it.

Covers string coordinates in build scripts (`"group:name:1.0"`, in
Groovy or Kotlin), and in version catalogs both `alias = "group:name:1.0"`
and `{ module = "group:name", version = "1.0" }`, including a
`version.ref` that points into `[versions]`. Coordinates whose version is
a variable (`"group:name:$v"`) are left alone.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `module` | The library as `group:name`. |
| `version` | The new version. |

**Returns:** The list of paths that changed.

**Example**

```python
set_dependency(repo, "com.squareup.okhttp3:okhttp", "4.12.0")
set_dependency(repo, "com.fasterxml.jackson.core:jackson-databind", "2.17.2")
# returns ["gradle/libs.versions.toml"]
```

=== "Before"

    `build.gradle.kts`

    ```text
    dependencies {
        implementation("com.squareup.okhttp3:okhttp:4.11.0")
        testImplementation(libs.junit)
    }
    ```

    `gradle/libs.versions.toml`

    ```toml
    [versions]
    jackson = "2.15.2"

    [libraries]
    jackson-databind = { module = "com.fasterxml.jackson.core:jackson-databind", version.ref = "jackson" }
    ```

=== "After"

    `build.gradle.kts`

    ```text
    dependencies {
        implementation("com.squareup.okhttp3:okhttp:4.12.0")
        testImplementation(libs.junit)
    }
    ```

    `gradle/libs.versions.toml`

    ```toml
    [versions]
    jackson = "2.17.2"

    [libraries]
    jackson-databind = { module = "com.fasterxml.jackson.core:jackson-databind", version.ref = "jackson" }
    ```

**Used in:** [`jvm-dependency-bump`](https://github.com/rossedman/gh-fleet/tree/main/examples/jvm-dependency-bump)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/gradle.star#L34)

## wrapper_version

```python
wrapper_version(repo)
```

Reads the Gradle version the wrapper downloads.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The version, such as `8.5`, or None when the repo has no wrapper.

**Example**

```python
wrapper_version(repo)
# returns "8.5"
```

=== "Before"

    `gradle/wrapper/gradle-wrapper.properties`

    ```text
    distributionUrl=https\://services.gradle.org/distributions/gradle-8.5-bin.zip
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/gradle.star#L98)

## set_wrapper_version

```python
set_wrapper_version(repo, version, sha256 = None)
```

Points the Gradle wrapper at a new version, never lowering it.

Only gradle-wrapper.properties changes; Gradle downloads the new
distribution on the next run. When the file pins a checksum with
`distributionSha256Sum`, pass the new distribution's `sha256` from
https://gradle.org/release-checksums/, or the run fails rather than
leave a checksum that no longer matches.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `version` | The Gradle version, such as `8.10.2`. |
| `sha256` | The SHA-256 of the new distribution zip, for wrappers that verify it, or a dict of it by distribution type (`bin`, `all`), so one campaign covers both. |

**Returns:** True when the file changed.

**Example**

```python
set_wrapper_version(repo, "8.10.2")
# returns True
```

=== "Before"

    `gradle/wrapper/gradle-wrapper.properties`

    ```text
    distributionBase=GRADLE_USER_HOME
    distributionUrl=https\://services.gradle.org/distributions/gradle-8.5-bin.zip
    ```

=== "After"

    `gradle/wrapper/gradle-wrapper.properties`

    ```text
    distributionBase=GRADLE_USER_HOME
    distributionUrl=https\://services.gradle.org/distributions/gradle-8.10.2-bin.zip
    ```

**Used in:** [`gradle-wrapper-bump`](https://github.com/rossedman/gh-fleet/tree/main/examples/gradle-wrapper-bump)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/gradle.star#L114)
