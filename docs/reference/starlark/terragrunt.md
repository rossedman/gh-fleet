---
title: terragrunt.star
---

# terragrunt.star

Move Terragrunt and Terraform module sources to a new `?ref=` tag.

Edits are line-based, so comments and alignment stay as they were.

```python
load("@fleet//terragrunt.star", "sources", "set_source_ref")
```

| Function | What it does |
| --- | --- |
| [`sources`](#sources) | Lists every module source pinned with `?ref=`. |
| [`set_source_ref`](#set_source_ref) | Sets the `?ref=` of every module source that matches. |

## sources

```python
sources(repo)
```

Lists every module source pinned with `?ref=`.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** A list of dicts with `path`, `source` and `ref`, in file order.

**Example**

```python
[s["source"] + " " + s["ref"] for s in sources(repo)]
# returns ["git::https://github.com/acme/modules.git//vpc v1.2.0"]
```

=== "Before"

    `live/prod/vpc/terragrunt.hcl`

    ```text
    terraform {
      source = "git::https://github.com/acme/modules.git//vpc?ref=v1.2.0"
    }
    ```

**Used in:** [`terragrunt-module-ref`](https://github.com/rossedman/gh-fleet/tree/main/examples/terragrunt-module-ref)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/terragrunt.star#L11)

## set_source_ref

```python
set_source_ref(repo, source, ref)
```

Sets the `?ref=` of every module source that matches.

Covers `source` in terragrunt.hcl `terraform` blocks and in Terraform
`module` blocks. Sources pinned to a commit SHA are left alone.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `source` | The source before `?ref=`; `*` matches any run of characters, so `git::https://github.com/acme/modules.git//*` covers every module in that repository. |
| `ref` | The new tag or branch. |

**Returns:** The list of paths that changed.

**Example**

```python
set_source_ref(repo, "git::https://github.com/acme/modules.git//*", "v1.3.0")
# returns ["live/prod/vpc/terragrunt.hcl", "main.tf"]
```

=== "Before"

    `live/prod/vpc/terragrunt.hcl`

    ```text
    terraform {
      source = "git::https://github.com/acme/modules.git//vpc?ref=v1.2.0"
    }
    ```

    `main.tf`

    ```hcl
    module "db" {
      source = "git::https://github.com/acme/modules.git//rds?ref=v1.1.0"
    }
    ```

=== "After"

    `live/prod/vpc/terragrunt.hcl`

    ```text
    terraform {
      source = "git::https://github.com/acme/modules.git//vpc?ref=v1.3.0"
    }
    ```

    `main.tf`

    ```hcl
    module "db" {
      source = "git::https://github.com/acme/modules.git//rds?ref=v1.3.0"
    }
    ```

**Used in:** [`terragrunt-module-ref`](https://github.com/rossedman/gh-fleet/tree/main/examples/terragrunt-module-ref)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/terragrunt.star#L28)
