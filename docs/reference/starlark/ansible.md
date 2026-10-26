---
title: ansible.star
---

# ansible.star

Bump Ansible role and collection versions in requirements files.

Edits are line-based, so comments and layout stay as they were.

```python
load("@fleet//ansible.star", "requirements", "set_version")
```

| Function | What it does |
| --- | --- |
| [`requirements`](#requirements) | Lists the repo's Ansible Galaxy requirements files. |
| [`set_version`](#set_version) | Sets the version of a role or collection wherever a requirements file lists it. |

## requirements

```python
requirements(repo)
```

Lists the repo's Ansible Galaxy requirements files.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The sorted paths of every requirements.yml or requirements.yaml,
including those under roles/ and collections/.

**Example**

```python
requirements(repo)
# returns ["requirements.yml", "roles/requirements.yml"]
```

=== "Before"

    `requirements.yml`

    ```yaml
    collections: []
    ```

    `roles/requirements.yml`

    ```yaml
    roles: []
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/ansible.star#L8)

## set_version

```python
set_version(repo, name, version)
```

Sets the version of a role or collection wherever a requirements file lists it.

Entries are matched by `name`, and for roles also by `src`. A missing
`version` key is added.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `name` | The collection or role name, such as `community.general`. |
| `version` | The new version or range, such as `9.4.0` or `>=9.0.0`. |

**Returns:** The list of paths that changed.

**Example**

```python
set_version(repo, "community.general", "9.4.0")
# returns ["requirements.yml"]
```

=== "Before"

    `requirements.yml`

    ```yaml
    collections:
      - name: community.general
        version: 7.0.0
      - name: ansible.posix
        version: 1.5.4
    ```

=== "After"

    `requirements.yml`

    ```yaml
    collections:
      - name: community.general
        version: 9.4.0
      - name: ansible.posix
        version: 1.5.4
    ```

**Used in:** [`ansible-collections`](https://github.com/rossedman/gh-fleet/tree/main/examples/ansible-collections)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/ansible.star#L26)
