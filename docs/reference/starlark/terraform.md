---
title: terraform.star
---

# terraform.star

Bump Terraform's required_version and provider version constraints.

Edits are line-based, so comments and alignment in .tf files stay as they
were. .terraform.lock.hcl is not touched; run `terraform init -upgrade`
in CI or a `run` step to refresh it.

```python
load("@fleet//terraform.star", "files", "set_required_version", "set_provider_version")
```

| Function | What it does |
| --- | --- |
| [`files`](#files) | Lists the repo's Terraform files. |
| [`set_required_version`](#set_required_version) | Sets every `required_version` constraint. |
| [`set_provider_version`](#set_provider_version) | Sets the version constraint of a provider in `required_providers`. |

## files

```python
files(repo)
```

Lists the repo's Terraform files.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The sorted paths of every .tf file outside .terraform directories.

**Example**

```python
files(repo)
# returns ["main.tf", "modules/vpc/main.tf"]
```

=== "Before"

    `main.tf`

    ```hcl
    terraform {}
    ```

    `modules/vpc/main.tf`

    ```hcl
    terraform {}
    ```

    `.terraform/modules/x/main.tf`

    ```hcl
    terraform {}
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/terraform.star#L8)

## set_required_version

```python
set_required_version(repo, constraint)
```

Sets every `required_version` constraint.

A newer version already there is kept, so a campaign never lowers one.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `constraint` | The new constraint, such as `>= 1.9`. |

**Returns:** The list of paths that changed.

**Example**

```python
set_required_version(repo, ">= 1.9")
# returns ["versions.tf"]
```

=== "Before"

    `versions.tf`

    ```hcl
    terraform {
      required_version = ">= 1.5"
    }
    ```

=== "After"

    `versions.tf`

    ```hcl
    terraform {
      required_version = ">= 1.9"
    }
    ```

**Used in:** [`terraform-provider-bump`](https://github.com/rossedman/gh-fleet/tree/main/examples/terraform-provider-bump)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/terraform.star#L30)

## set_provider_version

```python
set_provider_version(repo, source, constraint)
```

Sets the version constraint of a provider in `required_providers`.

A newer version already there is kept, so a campaign never lowers one.

Works on entries written as a block or on one line
(`aws = { source = "hashicorp/aws", version = "~> 5.0" }`), and on
sources with the `registry.terraform.io/` prefix. Entries without a
`version` are left alone.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `source` | The provider source, such as `hashicorp/aws`. |
| `constraint` | The new constraint, such as `~> 5.0`. |

**Returns:** The list of paths that changed.

**Example**

```python
set_provider_version(repo, "hashicorp/aws", "~> 5.0")
# returns ["versions.tf"]
```

=== "Before"

    `versions.tf`

    ```hcl
    terraform {
      required_providers {
        aws = {
          source  = "hashicorp/aws"
          version = "~> 4.0" # pinned
        }
      }
    }
    ```

=== "After"

    `versions.tf`

    ```hcl
    terraform {
      required_providers {
        aws = {
          source  = "hashicorp/aws"
          version = "~> 5.0" # pinned
        }
      }
    }
    ```

**Used in:** [`terraform-provider-bump`](https://github.com/rossedman/gh-fleet/tree/main/examples/terraform-provider-bump)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/terraform.star#L63)
