---
title: buf.star
---

# buf.star

Bump Buf remote plugins, module dependencies and the buf.yaml format.

Edits are line-based, so comments in buf.yaml and buf.gen.yaml stay as
they were.

```python
load("@fleet//buf.star", "set_plugin_version", "set_dep_ref")
```

| Function | What it does |
| --- | --- |
| [`set_plugin_version`](#set_plugin_version) | Moves a remote plugin to a new version in every buf.gen.yaml. |
| [`set_dep_ref`](#set_dep_ref) | Pins a module dependency in buf.yaml to a label or commit, such as `buf.build/googleapis/googleapis:<commit>`. |

## set_plugin_version

```python
set_plugin_version(repo, plugin, version)
```

Moves a remote plugin to a new version in every buf.gen.yaml.

Covers `remote:` (v1) and `plugin:` / `remote:` under `plugins` (v2),
written as `buf.build/<owner>/<plugin>:<version>`.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `plugin` | The plugin, such as `buf.build/protocolbuffers/go`. |
| `version` | The new version, such as `v1.35.1`. |

**Returns:** The list of paths that changed.

**Example**

```python
set_plugin_version(repo, "buf.build/protocolbuffers/go", "v1.35.1")
# returns ["buf.gen.yaml"]
```

=== "Before"

    `buf.gen.yaml`

    ```yaml
    version: v2
    plugins:
      - remote: buf.build/protocolbuffers/go:v1.31.0
        out: gen/go
      - remote: buf.build/grpc/go:v1.3.0
        out: gen/go
    ```

=== "After"

    `buf.gen.yaml`

    ```yaml
    version: v2
    plugins:
      - remote: buf.build/protocolbuffers/go:v1.35.1
        out: gen/go
      - remote: buf.build/grpc/go:v1.3.0
        out: gen/go
    ```

**Used in:** [`buf-plugins`](../examples/buf-plugins.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/buf.star#L14)

## set_dep_ref

```python
set_dep_ref(repo, module, ref)
```

Pins a module dependency in buf.yaml to a label or commit, such as `buf.build/googleapis/googleapis:<commit>`.

Dependencies listed without a ref get one. buf.lock isn't updated; run
`buf dep update` (or `buf mod update` for v1) in a `run` step.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `module` | The module, such as `buf.build/googleapis/googleapis`. |
| `ref` | The label or commit. |

**Returns:** The list of paths that changed.

**Example**

```python
set_dep_ref(repo, "buf.build/googleapis/googleapis", "e7f8d366f5264595bcc4cd4139af9973")
set_dep_ref(repo, "buf.build/bufbuild/protovalidate", "v0.8.0")
# returns ["buf.yaml"]
```

=== "Before"

    `buf.yaml`

    ```yaml
    version: v2
    deps:
      - buf.build/googleapis/googleapis
      - buf.build/bufbuild/protovalidate:v0.6.0
    ```

=== "After"

    `buf.yaml`

    ```yaml
    version: v2
    deps:
      - buf.build/googleapis/googleapis:e7f8d366f5264595bcc4cd4139af9973
      - buf.build/bufbuild/protovalidate:v0.8.0
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/buf.star#L47)
