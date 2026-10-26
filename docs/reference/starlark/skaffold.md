---
title: skaffold.star
---

# skaffold.star

Bump Helm chart versions deployed by Skaffold.

Edits are line-based, so comments and profiles in skaffold.yaml stay as
they were. For the `apiVersion` of the config itself, run `skaffold fix`
in a `run` step: schema upgrades rename fields, which it knows how to do.

```python
load("@fleet//skaffold.star", "files", "set_helm_release_version")
```

| Function | What it does |
| --- | --- |
| [`files`](#files) | Lists the repo's Skaffold configs. |
| [`set_helm_release_version`](#set_helm_release_version) | Sets the `version` of every Helm release that installs a remote chart. |

## files

```python
files(repo)
```

Lists the repo's Skaffold configs.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The sorted paths of every skaffold.yaml and skaffold.yml.

**Example**

```python
files(repo)
# returns ["skaffold.yaml"]
```

=== "Before"

    `skaffold.yaml`

    ```yaml
    apiVersion: skaffold/v4beta11
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/skaffold.star#L10)

## set_helm_release_version

```python
set_helm_release_version(repo, chart, version)
```

Sets the `version` of every Helm release that installs a remote chart.

A newer version already there is kept, so a campaign never lowers one.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `chart` | The release's `remoteChart`, such as `bitnami/postgresql`. |
| `version` | The new chart version. |

**Returns:** The list of paths that changed.

**Example**

```python
set_helm_release_version(repo, "bitnami/postgresql", "16.0.3")
# returns ["skaffold.yaml"]
```

=== "Before"

    `skaffold.yaml`

    ```yaml
    deploy:
      helm:
        releases:
          - name: db
            remoteChart: bitnami/postgresql
            version: 13.2.0
    ```

=== "After"

    `skaffold.yaml`

    ```yaml
    deploy:
      helm:
        releases:
          - name: db
            remoteChart: bitnami/postgresql
            version: 16.0.3
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/skaffold.star#L25)
