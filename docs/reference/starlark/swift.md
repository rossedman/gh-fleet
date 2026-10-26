---
title: swift.star
---

# swift.star

Bump Swift Package Manager dependencies and the swift-tools-version.

Edits are line-based, so comments and layout in Package.swift stay as
they were. Package.resolved isn't updated; run `swift package update
<name>` in a `run` step when it is committed.

```python
load("@fleet//swift.star", "manifests", "set_package", "set_tools_version")
```

| Function | What it does |
| --- | --- |
| [`manifests`](#manifests) | Lists the repo's Package.swift files. |
| [`set_package`](#set_package) | Sets the version of a package dependency wherever a manifest declares it. |
| [`set_tools_version`](#set_tools_version) | Raises the `// swift-tools-version:` line, never lowering it. |

## manifests

```python
manifests(repo)
```

Lists the repo's Package.swift files.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The sorted paths of every Package.swift outside .build directories.

**Example**

```python
manifests(repo)
# returns ["Package.swift", "Tools/Package.swift"]
```

=== "Before"

    `Package.swift`

    ```text
    // swift-tools-version:5.9
    ```

    `Tools/Package.swift`

    ```text
    // swift-tools-version:5.9
    ```

    `.build/checkouts/x/Package.swift`

    ```text
    // swift-tools-version:5.5
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/swift.star#L11)

## set_package

```python
set_package(repo, url, version)
```

Sets the version of a package dependency wherever a manifest declares it.

A newer version already there is kept, so a campaign never lowers one.

Covers `from:`, `exact:`, `.upToNextMajor(from:)` and
`.upToNextMinor(from:)` requirements; the kind of requirement stays.
Packages pinned to a branch or revision are left alone. The URL matches
with or without a trailing `.git`.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `url` | The package URL, such as `https://github.com/apple/swift-log`. |
| `version` | The new version. |

**Returns:** The list of paths that changed.

**Example**

```python
set_package(repo, "https://github.com/apple/swift-log", "1.6.1")
# returns ["Package.swift"]
```

=== "Before"

    `Package.swift`

    ```text
    dependencies: [
        .package(url: "https://github.com/apple/swift-log.git", from: "1.5.3"),
        .package(url: "https://github.com/vapor/vapor", .upToNextMinor(from: "4.89.0")),
    ]
    ```

=== "After"

    `Package.swift`

    ```text
    dependencies: [
        .package(url: "https://github.com/apple/swift-log.git", from: "1.6.1"),
        .package(url: "https://github.com/vapor/vapor", .upToNextMinor(from: "4.89.0")),
    ]
    ```

**Used in:** [`swift-package-bump`](https://github.com/rossedman/gh-fleet/tree/main/examples/swift-package-bump)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/swift.star#L30)

## set_tools_version

```python
set_tools_version(repo, version)
```

Raises the `// swift-tools-version:` line, never lowering it.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `version` | The Swift tools version, such as `5.10`. |

**Returns:** The list of paths that changed.

**Example**

```python
set_tools_version(repo, "5.10")
# returns ["Package.swift"]
```

=== "Before"

    `Package.swift`

    ```text
    // swift-tools-version:5.7
    import PackageDescription
    ```

=== "After"

    `Package.swift`

    ```text
    // swift-tools-version:5.10
    import PackageDescription
    ```

**Used in:** [`swift-package-bump`](https://github.com/rossedman/gh-fleet/tree/main/examples/swift-package-bump)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/swift.star#L72)
