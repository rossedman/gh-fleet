---
title: android.star
---

# android.star

Raise Android SDK levels and the Android Gradle Plugin version.

Google Play requires a recent `targetSdk` every year, which makes this a
fleet-wide change for anyone with several apps. Edits are line-based and
work on Groovy and Kotlin DSL build files and version catalogs.

```python
load("@fleet//android.star", "sdk_levels", "set_sdk", "set_agp_version")
```

| Function | What it does |
| --- | --- |
| [`sdk_levels`](#sdk_levels) | Reads the SDK levels every module's build file sets. |
| [`set_sdk`](#set_sdk) | Raises the compile, target and min SDK levels, never lowering one. |
| [`set_agp_version`](#set_agp_version) | Moves the Android Gradle Plugin to a version. |

## sdk_levels

```python
sdk_levels(repo)
```

Reads the SDK levels every module's build file sets.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** A dict of build file path to a dict of `compile`, `target` and
`min` levels (as integers), for files that set any.

**Example**

```python
sdk_levels(repo)["app/build.gradle.kts"]["target"]
# returns 34
```

=== "Before"

    `app/build.gradle.kts`

    ```text
    android {
        compileSdk = 34
        defaultConfig {
            minSdk = 24
            targetSdk = 34
        }
    }
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/android.star#L23)

## set_sdk

```python
set_sdk(repo, compile = None, target = None, min = None)
```

Raises the compile, target and min SDK levels, never lowering one.

Covers `compileSdk 34`, `compileSdkVersion 33`, `targetSdk = 34` and
`minSdk(24)` forms in every module, and `compileSdk`, `targetSdk` and
`minSdk` entries in a version catalog's `[versions]`.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `compile` | The compileSdk level to raise to, or None. |
| `target` | The targetSdk level to raise to, or None. |
| `min` | The minSdk level to raise to, or None. |

**Returns:** The list of paths that changed.

**Example**

```python
set_sdk(repo, compile = 35, target = 35)
# returns ["app/build.gradle"]
```

=== "Before"

    `app/build.gradle`

    ```text
    android {
        compileSdkVersion 33
        defaultConfig {
            minSdkVersion 21
            targetSdkVersion 33
        }
    }
    ```

=== "After"

    `app/build.gradle`

    ```text
    android {
        compileSdkVersion 35
        defaultConfig {
            minSdkVersion 21
            targetSdkVersion 35
        }
    }
    ```

**Used in:** [`android-target-sdk`](https://github.com/rossedman/gh-fleet/tree/main/examples/android-target-sdk)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/android.star#L56)

## set_agp_version

```python
set_agp_version(repo, version)
```

Moves the Android Gradle Plugin to a version.

Sets the `com.android.*` plugins wherever they are declared (plugin
blocks or a version catalog) and the
`com.android.tools.build:gradle` classpath of older builds. Upgrading AGP
often needs a newer Gradle wrapper too; see
`gradle.set_wrapper_version`.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `version` | The AGP version, such as `8.7.2`. |

**Returns:** The list of paths that changed.

**Example**

```python
set_agp_version(repo, "8.7.2")
# returns ["build.gradle"]
```

=== "Before"

    `build.gradle`

    ```text
    buildscript {
        dependencies {
            classpath 'com.android.tools.build:gradle:7.4.2'
        }
    }
    ```

=== "After"

    `build.gradle`

    ```text
    buildscript {
        dependencies {
            classpath 'com.android.tools.build:gradle:8.7.2'
        }
    }
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/android.star#L114)
