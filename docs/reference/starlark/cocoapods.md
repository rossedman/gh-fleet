---
title: cocoapods.star
---

# cocoapods.star

Bump CocoaPods pod versions and the platform version in Podfiles.

Edits are line-based, so comments and layout stay as they were.
Podfile.lock isn't updated; run `pod update <name>` in a `run` step on a
macOS runner when it is committed.

```python
load("@fleet//cocoapods.star", "podfiles", "set_pod", "set_platform")
```

| Function | What it does |
| --- | --- |
| [`podfiles`](#podfiles) | Lists the repo's Podfiles. |
| [`set_pod`](#set_pod) | Sets a pod's version requirement wherever a Podfile lists it with one. |
| [`set_platform`](#set_platform) | Raises the deployment target in `platform :ios, '13.0'`, never lowering it. |

## podfiles

```python
podfiles(repo)
```

Lists the repo's Podfiles.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The sorted paths of every Podfile outside Pods directories.

**Example**

```python
podfiles(repo)
# returns ["ios/Podfile"]
```

=== "Before"

    `ios/Podfile`

    ```text
    platform :ios, '13.0'
    ```

    `ios/Pods/x/Podfile`

    ```text
    x
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/cocoapods.star#L11)

## set_pod

```python
set_pod(repo, name, requirement)
```

Sets a pod's version requirement wherever a Podfile lists it with one.

Pods without a version, or pinned to a `:git` source, are left alone.
Subspecs match by their full name, such as `Firebase/Analytics`.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `name` | The pod name. |
| `requirement` | The requirement, such as `~> 5.9`. |

**Returns:** The list of paths that changed.

**Example**

```python
set_pod(repo, "Alamofire", "~> 5.9")
# returns ["ios/Podfile"]
```

=== "Before"

    `ios/Podfile`

    ```text
    target 'App' do
      pod 'Alamofire', '~> 5.6'
      pod 'SnapKit'
    end
    ```

=== "After"

    `ios/Podfile`

    ```text
    target 'App' do
      pod 'Alamofire', '~> 5.9'
      pod 'SnapKit'
    end
    ```

**Used in:** [`cocoapods-ios-15`](https://github.com/rossedman/gh-fleet/tree/main/examples/cocoapods-ios-15)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/cocoapods.star#L28)

## set_platform

```python
set_platform(repo, platform, version)
```

Raises the deployment target in `platform :ios, '13.0'`, never lowering it.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `platform` | The platform, such as `ios`, `osx` or `tvos`. |
| `version` | The minimum version, such as `15.0`. |

**Returns:** The list of paths that changed.

**Example**

```python
set_platform(repo, "ios", "15.0")
# returns ["ios/Podfile"]
```

=== "Before"

    `ios/Podfile`

    ```text
    platform :ios, '12.0'
    use_frameworks!
    ```

=== "After"

    `ios/Podfile`

    ```text
    platform :ios, '15.0'
    use_frameworks!
    ```

**Used in:** [`cocoapods-ios-15`](https://github.com/rossedman/gh-fleet/tree/main/examples/cocoapods-ios-15)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/cocoapods.star#L65)
