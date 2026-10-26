---
title: backstage.star
---

# backstage.star

Edit Backstage catalog-info.yaml files: owners, annotations, tags and other entity fields.

A catalog file can hold several entities as separate YAML documents, so
every edit can be narrowed to entities of one `kind` or one `name`. Edits
are line-based, through `yamlpath`, so comments and layout survive.

```python
load("@fleet//backstage.star", "files", "entities", "set_owner", "set_field", "set_annotation", "add_tag", "ensure")
```

| Function | What it does |
| --- | --- |
| [`files`](#files) | Lists the repo's Backstage catalog files. |
| [`entities`](#entities) | Lists the entities in every catalog file. |
| [`set_owner`](#set_owner) | Sets `spec.owner`, such as after a team is renamed or a service changes hands. |
| [`set_field`](#set_field) | Sets any entity field by key path, such as `spec.lifecycle` or `spec.system`. |
| [`set_annotation`](#set_annotation) | Sets a `metadata.annotations` entry, such as `backstage.io/techdocs-ref`. |
| [`add_tag`](#add_tag) | Adds a tag to `metadata.tags`, keeping the list's flow or block style. |
| [`ensure`](#ensure) | Writes a Component catalog file, unless the repo already has one. |

## files

```python
files(repo)
```

Lists the repo's Backstage catalog files.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The sorted paths of every catalog-info.yaml and catalog-info.yml,
outside node_modules.

**Example**

```python
files(repo)
# returns ["catalog-info.yaml", "services/billing/catalog-info.yml"]
```

=== "Before"

    `catalog-info.yaml`

    ```yaml
    kind: Component
    ```

    `services/billing/catalog-info.yml`

    ```yaml
    kind: Component
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/backstage.star#L10)

## entities

```python
entities(repo)
```

Lists the entities in every catalog file.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** One dict per entity, in file order, with `path`, `kind`, `name`,
`type`, `owner`, `lifecycle` and `system`; a field the entity
doesn't set is None.

**Example**

```python
[(e["kind"], e["name"], e["owner"], e["lifecycle"]) for e in entities(repo)]
# returns [("Component", "billing-api", "group:payments", "production"), ("API", "billing", "payments", None)]
```

=== "Before"

    `catalog-info.yaml`

    ```yaml
    apiVersion: backstage.io/v1alpha1
    kind: Component
    metadata:
      name: billing-api
    spec:
      type: service
      owner: group:payments
      lifecycle: production
    ---
    apiVersion: backstage.io/v1alpha1
    kind: API
    metadata:
      name: billing
    spec:
      type: openapi
      owner: payments
    ```

**Used in:** [`backstage-team-rename`](https://github.com/rossedman/gh-fleet/tree/main/examples/backstage-team-rename)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/backstage.star#L87)

## set_owner

```python
set_owner(repo, owner, from_owner = None, kind = None, name = None)
```

Sets `spec.owner`, such as after a team is renamed or a service changes hands.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `owner` | The new owner, written as given: `payments`, `group:payments` or `user:jdoe`. |
| `from_owner` | When set, only entities owned by this group change. `payments`, `group:payments` and `group:default/payments` all match each other. |
| `kind` | Only change entities of this kind, such as `Component`. |
| `name` | Only change the entity with this `metadata.name`. |

**Returns:** The list of paths that changed.

**Example**

```python
set_owner(repo, "group:billing", from_owner = "payments")
# returns ["catalog-info.yaml"]
```

=== "Before"

    `catalog-info.yaml`

    ```yaml
    kind: Component
    metadata:
      name: billing-api
    spec:
      owner: group:default/payments  # since 2023
    ---
    kind: Component
    metadata:
      name: search
    spec:
      owner: discovery
    ```

=== "After"

    `catalog-info.yaml`

    ```yaml
    kind: Component
    metadata:
      name: billing-api
    spec:
      owner: group:billing  # since 2023
    ---
    kind: Component
    metadata:
      name: search
    spec:
      owner: discovery
    ```

**Used in:** [`backstage-team-rename`](https://github.com/rossedman/gh-fleet/tree/main/examples/backstage-team-rename)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/backstage.star#L139)

## set_field

```python
set_field(
    repo,
    key,
    value,
    kind = None,
    name = None,
    only_missing = False,
)
```

Sets any entity field by key path, such as `spec.lifecycle` or `spec.system`.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `key` | The dotted key path, or a list for keys containing dots. |
| `value` | A string, number, boolean, None or list of those. |
| `kind` | Only change entities of this kind. |
| `name` | Only change the entity with this `metadata.name`. |
| `only_missing` | Leave entities that already set the key alone. |

**Returns:** The list of paths that changed.

**Example**

```python
set_field(repo, "spec.lifecycle", "production", kind = "Component")
set_field(repo, "spec.system", "billing", only_missing = True)
# returns ["catalog-info.yaml"]
```

=== "Before"

    `catalog-info.yaml`

    ```yaml
    kind: Component
    metadata:
      name: billing-api
    spec:
      type: service
      lifecycle: experimental
    ---
    kind: API
    metadata:
      name: billing
    spec:
      type: openapi
    ```

=== "After"

    `catalog-info.yaml`

    ```yaml
    kind: Component
    metadata:
      name: billing-api
    spec:
      type: service
      lifecycle: production
      system: billing
    ---
    kind: API
    metadata:
      name: billing
    spec:
      type: openapi
      system: billing
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/backstage.star#L192)

## set_annotation

```python
set_annotation(
    repo,
    key,
    value,
    kind = None,
    name = None,
    only_missing = False,
)
```

Sets a `metadata.annotations` entry, such as `backstage.io/techdocs-ref`.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `key` | The annotation, such as `pagerduty.com/service-id`. |
| `value` | Its value. |
| `kind` | Only change entities of this kind. |
| `name` | Only change the entity with this `metadata.name`. |
| `only_missing` | Leave entities that already have the annotation alone. |

**Returns:** The list of paths that changed.

**Example**

```python
set_annotation(repo, "backstage.io/techdocs-ref", "dir:.")
# returns ["catalog-info.yaml"]
```

=== "Before"

    `catalog-info.yaml`

    ```yaml
    kind: Component
    metadata:
      name: billing-api
      annotations:
        github.com/project-slug: acme/billing-api
    ```

=== "After"

    `catalog-info.yaml`

    ```yaml
    kind: Component
    metadata:
      name: billing-api
      annotations:
        github.com/project-slug: acme/billing-api
        backstage.io/techdocs-ref: dir:.
    ```

**Used in:** [`backstage-onboarding`](https://github.com/rossedman/gh-fleet/tree/main/examples/backstage-onboarding)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/backstage.star#L246)

## add_tag

```python
add_tag(repo, tag, kind = None, name = None)
```

Adds a tag to `metadata.tags`, keeping the list's flow or block style.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `tag` | The tag, such as `go` or `pci`. Backstage tags are lowercase letters, digits, `-`, `+` and `#`. |
| `kind` | Only change entities of this kind. |
| `name` | Only change the entity with this `metadata.name`. |

**Returns:** The list of paths that changed.

**Example**

```python
add_tag(repo, "pci")
# returns ["catalog-info.yaml"]
```

=== "Before"

    `catalog-info.yaml`

    ```yaml
    kind: Component
    metadata:
      name: billing-api
      tags:
        - java
    ---
    kind: API
    metadata:
      name: billing
      tags: [rest]
    ---
    kind: Resource
    metadata:
      name: billing-db
    ```

=== "After"

    `catalog-info.yaml`

    ```yaml
    kind: Component
    metadata:
      name: billing-api
      tags:
        - java
        - pci
    ---
    kind: API
    metadata:
      name: billing
      tags: [rest, pci]
    ---
    kind: Resource
    metadata:
      name: billing-db
      tags: [pci]
    ```

**Used in:** [`backstage-onboarding`](https://github.com/rossedman/gh-fleet/tree/main/examples/backstage-onboarding)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/backstage.star#L314)

## ensure

```python
ensure(
    repo,
    owner,
    type = "service",
    lifecycle = "production",
    name = None,
    system = None,
    description = None,
    path = "catalog-info.yaml",
)
```

Writes a Component catalog file, unless the repo already has one.

The component gets the `github.com/project-slug` annotation, so
Backstage's GitHub plugins find the repo.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `owner` | The owning group or user, such as `group:payments`. |
| `type` | The component type, such as `service`, `website` or `library`. |
| `lifecycle` | `experimental`, `production` or `deprecated`. |
| `name` | The component name; defaults to the repo name. |
| `system` | The system the component belongs to, if any. |
| `description` | A one-line description, if any. |
| `path` | Where to write the file. |

**Returns:** True when the file was written; False when the repo already has a
catalog file.

**Example**

```python
ensure(repo, "group:payments", system = "billing", description = "Billing API.")
# returns True
```

=== "After"

    `catalog-info.yaml`

    ```yaml
    apiVersion: backstage.io/v1alpha1
    kind: Component
    metadata:
      name: api
      description: Billing API.
      annotations:
        github.com/project-slug: acme/api
    spec:
      type: service
      lifecycle: production
      owner: group:payments
      system: billing
    ```

**Used in:** [`backstage-onboarding`](https://github.com/rossedman/gh-fleet/tree/main/examples/backstage-onboarding)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/backstage.star#L366)
