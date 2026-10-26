---
title: packer.star
---

# packer.star

Bump plugin versions in Packer templates.

Edits are line-based, so comments and alignment in .pkr.hcl files stay as
they were.

```python
load("@fleet//packer.star", "templates", "set_plugin_version")
```

| Function | What it does |
| --- | --- |
| [`templates`](#templates) | Lists the repo's Packer HCL templates. |
| [`set_plugin_version`](#set_plugin_version) | Sets the `version` of a plugin in `required_plugins`, matched by its `source`. |

## templates

```python
templates(repo)
```

Lists the repo's Packer HCL templates.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The sorted paths of every *.pkr.hcl file.

**Example**

```python
templates(repo)
# returns ["images/base.pkr.hcl"]
```

=== "Before"

    `images/base.pkr.hcl`

    ```text
    packer {}
    ```

    `main.tf`

    ```hcl
    terraform {}
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/packer.star#L10)

## set_plugin_version

```python
set_plugin_version(repo, source, constraint)
```

Sets the `version` of a plugin in `required_plugins`, matched by its `source`.

A newer version already there is kept, so a campaign never lowers one.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `source` | The plugin source, such as `github.com/hashicorp/amazon`. |
| `constraint` | The new constraint, such as `>= 1.3.3`. |

**Returns:** The list of paths that changed.

**Example**

```python
set_plugin_version(repo, "github.com/hashicorp/amazon", ">= 1.3.3")
# returns ["images/base.pkr.hcl"]
```

=== "Before"

    `images/base.pkr.hcl`

    ```text
    packer {
      required_plugins {
        amazon = {
          version = ">= 1.2.0"
          source  = "github.com/hashicorp/amazon"
        }
      }
    }
    ```

=== "After"

    `images/base.pkr.hcl`

    ```text
    packer {
      required_plugins {
        amazon = {
          version = ">= 1.3.3"
          source  = "github.com/hashicorp/amazon"
        }
      }
    }
    ```

**Used in:** [`packer-plugins`](https://github.com/rossedman/gh-fleet/tree/main/examples/packer-plugins)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/packer.star#L27)
