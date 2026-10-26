---
title: flux.star
---

# flux.star

Edit Flux HelmReleases and sources: chart versions and Git refs.

Edits are line-based, so comments and layout stay as they were.

```python
load("@fleet//flux.star", "helm_releases", "set_chart_version", "set_git_tag")
```

| Function | What it does |
| --- | --- |
| [`helm_releases`](#helm_releases) | Lists the YAML files that define Flux HelmReleases. |
| [`set_chart_version`](#set_chart_version) | Sets the chart `version` of every HelmRelease that installs a chart. |
| [`set_git_tag`](#set_git_tag) | Sets `ref.tag` of every GitRepository that points at a URL. |

## helm_releases

```python
helm_releases(repo)
```

Lists the YAML files that define Flux HelmReleases.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The sorted paths of YAML files with a `helm.toolkit.fluxcd.io`
apiVersion and `kind: HelmRelease`.

**Example**

```python
helm_releases(repo)
# returns ["clusters/prod/redis.yaml"]
```

=== "Before"

    `clusters/prod/redis.yaml`

    ```yaml
    apiVersion: helm.toolkit.fluxcd.io/v2
    kind: HelmRelease
    ```

    `clusters/prod/ns.yaml`

    ```yaml
    apiVersion: v1
    kind: Namespace
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/flux.star#L16)

## set_chart_version

```python
set_chart_version(repo, chart, version)
```

Sets the chart `version` of every HelmRelease that installs a chart.

A newer version already there is kept, so a campaign never lowers one.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `chart` | The chart name, as in `spec.chart.spec.chart`. |
| `version` | The new version or range, such as `19.x`. |

**Returns:** The list of paths that changed.

**Example**

```python
set_chart_version(repo, "redis", "19.x")
# returns ["clusters/prod/redis.yaml"]
```

=== "Before"

    `clusters/prod/redis.yaml`

    ```yaml
    apiVersion: helm.toolkit.fluxcd.io/v2
    kind: HelmRelease
    spec:
      chart:
        spec:
          chart: redis
          version: "18.x"
          sourceRef:
            kind: HelmRepository
            name: bitnami
    ```

=== "After"

    `clusters/prod/redis.yaml`

    ```yaml
    apiVersion: helm.toolkit.fluxcd.io/v2
    kind: HelmRelease
    spec:
      chart:
        spec:
          chart: redis
          version: "19.x"
          sourceRef:
            kind: HelmRepository
            name: bitnami
    ```

**Used in:** [`flux-chart-bump`](../examples/flux-chart-bump.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/flux.star#L36)

## set_git_tag

```python
set_git_tag(repo, url, tag)
```

Sets `ref.tag` of every GitRepository that points at a URL.

A newer version already there is kept, so a campaign never lowers one.

Only sources already pinned by `tag` change; sources following a
branch, semver range or commit are left alone.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `url` | The repository URL, as in `spec.url`. |
| `tag` | The new tag. |

**Returns:** The list of paths that changed.

**Example**

```python
set_git_tag(repo, "https://github.com/acme/deploy", "v1.5.0")
# returns ["clusters/prod/source.yaml"]
```

=== "Before"

    `clusters/prod/source.yaml`

    ```yaml
    apiVersion: source.toolkit.fluxcd.io/v1
    kind: GitRepository
    spec:
      url: https://github.com/acme/deploy
      ref:
        tag: v1.4.0
    ```

=== "After"

    `clusters/prod/source.yaml`

    ```yaml
    apiVersion: source.toolkit.fluxcd.io/v1
    kind: GitRepository
    spec:
      url: https://github.com/acme/deploy
      ref:
        tag: v1.5.0
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/flux.star#L76)
