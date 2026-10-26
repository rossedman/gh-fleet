---
title: argocd.star
---

# argocd.star

Edit Argo CD Application manifests: move an application to a new revision or chart version.

Edits are line-based, so comments and layout stay as they were.

```python
load("@fleet//argocd.star", "applications", "set_target_revision")
```

| Function | What it does |
| --- | --- |
| [`applications`](#applications) | Lists the YAML files that define Argo CD Applications or ApplicationSets. |
| [`set_target_revision`](#set_target_revision) | Sets `targetRevision` for every source that points at a repository. |

## applications

```python
applications(repo)
```

Lists the YAML files that define Argo CD Applications or ApplicationSets.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The sorted paths of YAML files with `apiVersion: argoproj.io/...` and
`kind: Application` or `kind: ApplicationSet`.

**Example**

```python
applications(repo)
# returns ["apps/api.yaml"]
```

=== "Before"

    `apps/api.yaml`

    ```yaml
    apiVersion: argoproj.io/v1alpha1
    kind: Application
    ```

    `apps/svc.yaml`

    ```yaml
    apiVersion: v1
    kind: Service
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/argocd.star#L8)

## set_target_revision

```python
set_target_revision(repo, repo_url, revision)
```

Sets `targetRevision` for every source that points at a repository.

A newer version already there is kept, so a campaign never lowers one.

Works for `source` and each entry of `sources`. A Helm chart from a chart
repository is matched by its `repoURL` too, and its `targetRevision` is
the chart version.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `repo_url` | The source's `repoURL`, as written. |
| `revision` | The new branch, tag, commit or chart version. |

**Returns:** The list of paths that changed.

**Example**

```python
set_target_revision(repo, "https://github.com/acme/deploy.git", "v1.5.0")
# returns ["apps/api.yaml"]
```

=== "Before"

    `apps/api.yaml`

    ```yaml
    apiVersion: argoproj.io/v1alpha1
    kind: Application
    spec:
      source:
        repoURL: https://github.com/acme/deploy.git
        path: apps/api
        targetRevision: v1.4.0
    ```

=== "After"

    `apps/api.yaml`

    ```yaml
    apiVersion: argoproj.io/v1alpha1
    kind: Application
    spec:
      source:
        repoURL: https://github.com/acme/deploy.git
        path: apps/api
        targetRevision: v1.5.0
    ```

**Used in:** [`argocd-revision`](../examples/argocd-revision.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/argocd.star#L33)
