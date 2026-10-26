---
title: k8s.star
---

# k8s.star

Edit Kubernetes manifests: migrate removed API versions and set labels on every resource.

Works on multi-document YAML files, one document at a time. Edits are
line-based, so comments and layout stay as they were. To move images to a
new tag, use `docker.bump_image`; for Kustomize overlays, `kustomize`.

```python
load("@fleet//k8s.star", "manifests", "resources", "migrate_api_versions", "set_label")
```

| Function | What it does |
| --- | --- |
| [`manifests`](#manifests) | Lists the YAML files that hold Kubernetes resources. |
| [`resources`](#resources) | Lists every resource in the repo's manifests. |
| [`migrate_api_versions`](#migrate_api_versions) | Moves resources off API versions Kubernetes removed. |
| [`set_label`](#set_label) | Sets a label on every resource, and on the pods workloads create. |

## manifests

```python
manifests(repo)
```

Lists the YAML files that hold Kubernetes resources.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The sorted paths of YAML files with at least one document that has
top-level `apiVersion` and `kind` keys, outside node_modules and
Helm chart templates.

**Example**

```python
manifests(repo)
# returns ["k8s/api.yaml"]
```

=== "Before"

    `k8s/api.yaml`

    ```yaml
    apiVersion: apps/v1
    kind: Deployment
    ```

    `.github/workflows/ci.yml`

    ```yaml
    on: push
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/k8s.star#L67)

## resources

```python
resources(repo)
```

Lists every resource in the repo's manifests.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** A list of dicts with `path`, `api_version`, `kind` and `name`, in
file and document order.

**Example**

```python
["%s %s/%s" % (r["api_version"], r["kind"], r["name"]) for r in resources(repo)]
# returns ["apps/v1 Deployment/api", "v1 Service/api"]
```

=== "Before"

    `k8s/api.yaml`

    ```yaml
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: api
    ---
    apiVersion: v1
    kind: Service
    metadata:
      name: api
    ```

**Used in:** [`k8s-api-migration`](https://github.com/rossedman/gh-fleet/tree/main/examples/k8s-api-migration)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/k8s.star#L92)

## migrate_api_versions

```python
migrate_api_versions(repo, migrations = MIGRATIONS)
```

Moves resources off API versions Kubernetes removed.

The defaults cover moves where only `apiVersion` changes, such as
`policy/v1beta1` PodDisruptionBudgets to `policy/v1` and `batch/v1beta1`
CronJobs to `batch/v1`.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `migrations` | Maps `"<apiVersion> <Kind>"` to the new apiVersion. |

**Returns:** The list of paths that changed.

**Example**

```python
migrate_api_versions(repo)
# returns ["k8s/api.yaml"]
```

=== "Before"

    `k8s/api.yaml`

    ```yaml
    apiVersion: policy/v1beta1  # old
    kind: PodDisruptionBudget
    metadata:
      name: api
    ---
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: api
    ```

=== "After"

    `k8s/api.yaml`

    ```yaml
    apiVersion: policy/v1  # old
    kind: PodDisruptionBudget
    metadata:
      name: api
    ---
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: api
    ```

**Used in:** [`k8s-api-migration`](https://github.com/rossedman/gh-fleet/tree/main/examples/k8s-api-migration)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/k8s.star#L124)

## set_label

```python
set_label(repo, key, value, kinds = None, pod_templates = True)
```

Sets a label on every resource, and on the pods workloads create.

The label goes in `metadata.labels`, added if missing. With
`pod_templates`, Deployments, StatefulSets, DaemonSets, ReplicaSets,
Jobs and CronJobs get it in their pod template too, so the pods carry
it. Selectors are never touched.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `key` | The label key, such as `app.kubernetes.io/part-of`. |
| `value` | The label value. |
| `kinds` | The kinds to label, such as `["Deployment", "Service"]`; every kind when None. |
| `pod_templates` | Whether to label workloads' pod templates too. |

**Returns:** The list of paths that changed.

**Example**

```python
set_label(repo, "team", "payments")
# returns ["k8s/api.yaml"]
```

=== "Before"

    `k8s/api.yaml`

    ```yaml
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: api
    spec:
      template:
        spec:
          containers: []
    ```

=== "After"

    `k8s/api.yaml`

    ```yaml
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: api
      labels:
        team: payments
    spec:
      template:
        spec:
          containers: []
        metadata:
          labels:
            team: payments
    ```

**Used in:** [`k8s-labels`](https://github.com/rossedman/gh-fleet/tree/main/examples/k8s-labels)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/k8s.star#L177)
