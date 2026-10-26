---
title: helmfile.star
---

# helmfile.star

Bump chart versions of Helmfile releases.

Edits are line-based, so comments and templating in helmfile.yaml stay as
they were.

```python
load("@fleet//helmfile.star", "files", "set_chart_version")
```

| Function | What it does |
| --- | --- |
| [`files`](#files) | Lists the repo's Helmfiles. |
| [`set_chart_version`](#set_chart_version) | Sets the `version` of every release that installs a chart. |

## files

```python
files(repo)
```

Lists the repo's Helmfiles.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The sorted paths of helmfile.yaml, helmfile.yaml.gotmpl and files
under helmfile.d.

**Example**

```python
files(repo)
# returns ["helmfile.d/10-ingress.yaml", "helmfile.yaml"]
```

=== "Before"

    `helmfile.yaml`

    ```yaml
    releases: []
    ```

    `helmfile.d/10-ingress.yaml`

    ```yaml
    releases: []
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/helmfile.star#L9)

## set_chart_version

```python
set_chart_version(repo, chart, version)
```

Sets the `version` of every release that installs a chart.

A newer version already there is kept, so a campaign never lowers one.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `chart` | The chart as releases write it, such as `bitnami/redis`. |
| `version` | The new version or range. |

**Returns:** The list of paths that changed.

**Example**

```python
set_chart_version(repo, "bitnami/redis", "20.2.1")
# returns ["helmfile.yaml"]
```

=== "Before"

    `helmfile.yaml`

    ```yaml
    releases:
      - name: cache
        namespace: data
        chart: bitnami/redis
        version: 18.6.1
      - name: ingress
        chart: ingress-nginx/ingress-nginx
        version: 4.9.0
    ```

=== "After"

    `helmfile.yaml`

    ```yaml
    releases:
      - name: cache
        namespace: data
        chart: bitnami/redis
        version: 20.2.1
      - name: ingress
        chart: ingress-nginx/ingress-nginx
        version: 4.9.0
    ```

**Used in:** [`helmfile-chart`](../examples/helmfile-chart.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/helmfile.star#L27)
