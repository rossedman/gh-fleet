---
title: helm.star
---

# helm.star

Bump Helm chart versions, chart dependencies and image tags in values files.

Edits are line-based, so comments and layout in Chart.yaml and values
files stay as they were.

```python
load("@fleet//helm.star", "charts", "set_chart_version", "set_app_version", "set_dependency", "set_image_tag")
```

| Function | What it does |
| --- | --- |
| [`charts`](#charts) | Lists the repo's Helm charts. |
| [`set_chart_version`](#set_chart_version) | Sets a chart's `version` in its Chart.yaml. |
| [`set_app_version`](#set_app_version) | Sets a chart's `appVersion` in its Chart.yaml, keeping its quotes. |
| [`set_dependency`](#set_dependency) | Sets the version of a chart dependency in every Chart.yaml that lists it. |
| [`set_image_tag`](#set_image_tag) | Sets the `tag` next to `repository: <repository>` in values files. |

## charts

```python
charts(repo)
```

Lists the repo's Helm charts.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The sorted directories that hold a Chart.yaml; the repo root is `.`.

**Example**

```python
charts(repo)
# returns ["charts/api", "charts/worker"]
```

=== "Before"

    `charts/api/Chart.yaml`

    ```yaml
    name: api
    ```

    `charts/worker/Chart.yaml`

    ```yaml
    name: worker
    ```

**Used in:** [`helm-image-tag`](https://github.com/rossedman/gh-fleet/tree/main/examples/helm-image-tag)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/helm.star#L20)

## set_chart_version

```python
set_chart_version(repo, chart, version)
```

Sets a chart's `version` in its Chart.yaml.

A newer version already there is kept, so a campaign never lowers one.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `chart` | The chart directory, as `charts` lists it. |
| `version` | The new chart version. |

**Returns:** True when the file changed.

**Example**

```python
set_chart_version(repo, "charts/api", "1.5.0")
# returns True
```

=== "Before"

    `charts/api/Chart.yaml`

    ```yaml
    apiVersion: v2
    name: api
    version: 1.4.0
    appVersion: "2.3.1"
    ```

=== "After"

    `charts/api/Chart.yaml`

    ```yaml
    apiVersion: v2
    name: api
    version: 1.5.0
    appVersion: "2.3.1"
    ```

**Used in:** [`helm-image-tag`](https://github.com/rossedman/gh-fleet/tree/main/examples/helm-image-tag)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/helm.star#L37)

## set_app_version

```python
set_app_version(repo, chart, version)
```

Sets a chart's `appVersion` in its Chart.yaml, keeping its quotes.

A newer version already there is kept, so a campaign never lowers one.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `chart` | The chart directory, as `charts` lists it. |
| `version` | The new application version. |

**Returns:** True when the file changed.

**Example**

```python
set_app_version(repo, "charts/api", "2.4.0")
# returns True
```

=== "Before"

    `charts/api/Chart.yaml`

    ```yaml
    version: 1.4.0
    appVersion: "2.3.1"
    ```

=== "After"

    `charts/api/Chart.yaml`

    ```yaml
    version: 1.4.0
    appVersion: "2.4.0"
    ```

**Used in:** [`helm-image-tag`](https://github.com/rossedman/gh-fleet/tree/main/examples/helm-image-tag)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/helm.star#L65)

## set_dependency

```python
set_dependency(repo, name, version)
```

Sets the version of a chart dependency in every Chart.yaml that lists it.

A newer version already there is kept, so a campaign never lowers one.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `name` | The dependency's `name`. |
| `version` | The new version or range, such as `15.5.x`. |

**Returns:** The list of paths that changed. Remember to refresh Chart.lock with
`helm dependency update` in a `run` step if the repo commits it.

**Example**

```python
set_dependency(repo, "postgresql", "15.5.x")
# returns ["charts/api/Chart.yaml"]
```

=== "Before"

    `charts/api/Chart.yaml`

    ```yaml
    name: api
    dependencies:
      - name: postgresql
        version: 12.1.0  # bitnami
        repository: https://charts.bitnami.com/bitnami
      - name: redis
        version: 17.0.0
    ```

=== "After"

    `charts/api/Chart.yaml`

    ```yaml
    name: api
    dependencies:
      - name: postgresql
        version: 15.5.x  # bitnami
        repository: https://charts.bitnami.com/bitnami
      - name: redis
        version: 17.0.0
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/helm.star#L89)

## set_image_tag

```python
set_image_tag(
    repo,
    repository,
    tag,
    include = ["**/values*.yaml", "**/values*.yml"],
)
```

Sets the `tag` next to `repository: <repository>` in values files.

Matches the common layout where an image is a mapping with sibling
`repository` and `tag` keys, at any depth. Quotes around the old tag
are kept; images with no `tag` key are left alone.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `repository` | The image repository as written, such as `ghcr.io/acme/api`. |
| `tag` | The new tag. |
| `include` | Glob patterns of the values files to edit. |

**Returns:** The list of paths that changed.

**Example**

```python
set_image_tag(repo, "ghcr.io/acme/api", "2.4.0")
# returns ["charts/api/values.yaml"]
```

=== "Before"

    `charts/api/values.yaml`

    ```yaml
    image:
      repository: ghcr.io/acme/api
      tag: "2.3.1"
      pullPolicy: IfNotPresent
    sidecar:
      image:
        repository: envoyproxy/envoy
        tag: v1.29.0
    ```

=== "After"

    `charts/api/values.yaml`

    ```yaml
    image:
      repository: ghcr.io/acme/api
      tag: "2.4.0"
      pullPolicy: IfNotPresent
    sidecar:
      image:
        repository: envoyproxy/envoy
        tag: v1.29.0
    ```

**Used in:** [`helm-image-tag`](https://github.com/rossedman/gh-fleet/tree/main/examples/helm-image-tag)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/helm.star#L169)
