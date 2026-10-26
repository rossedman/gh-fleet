---
title: azure_pipelines.star
---

# azure_pipelines.star

Edit Azure Pipelines YAML: hosted agent images, task versions and template refs.

Edits are line-based, so comments and layout stay as they were.

```python
load("@fleet//azure_pipelines.star", "pipelines", "set_vm_image", "bump_task", "set_template_ref")
```

| Function | What it does |
| --- | --- |
| [`pipelines`](#pipelines) | Lists the repo's Azure Pipelines files. |
| [`set_vm_image`](#set_vm_image) | Sets `vmImage` on every pool that uses a Microsoft-hosted image. |
| [`bump_task`](#bump_task) | Moves a task to a new major version, such as `Docker@1` to `Docker@2`. |
| [`set_template_ref`](#set_template_ref) | Sets the `ref` of a template repository under `resources.repositories`. |

## pipelines

```python
pipelines(repo)
```

Lists the repo's Azure Pipelines files.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The sorted paths of azure-pipelines*.yml files anywhere and of YAML
files under .azure-pipelines and .azuredevops.

**Example**

```python
pipelines(repo)
# returns [".azure-pipelines/templates/build.yml", "azure-pipelines.yml"]
```

=== "Before"

    `azure-pipelines.yml`

    ```yaml
    trigger: [main]
    ```

    `.azure-pipelines/templates/build.yml`

    ```yaml
    steps: []
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/azure_pipelines.star#L11)

## set_vm_image

```python
set_vm_image(repo, image, from_image = None)
```

Sets `vmImage` on every pool that uses a Microsoft-hosted image.

Block (`pool:` then `vmImage:`) and flow (`pool: { vmImage: ... }`) forms
both work.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `image` | The new image, such as `ubuntu-24.04`. |
| `from_image` | When set, only this image is replaced, such as a retired `ubuntu-20.04`. |

**Returns:** The list of paths that changed.

**Example**

```python
set_vm_image(repo, "ubuntu-24.04", from_image = "ubuntu-20.04")
# returns ["azure-pipelines.yml"]
```

=== "Before"

    `azure-pipelines.yml`

    ```yaml
    pool:
      vmImage: ubuntu-20.04  # hosted
    jobs:
      - job: windows
        pool:
          vmImage: windows-2022
    ```

=== "After"

    `azure-pipelines.yml`

    ```yaml
    pool:
      vmImage: ubuntu-24.04  # hosted
    jobs:
      - job: windows
        pool:
          vmImage: windows-2022
    ```

**Used in:** [`azure-ubuntu-24`](../examples/azure-ubuntu-24.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/azure_pipelines.star#L29)

## bump_task

```python
bump_task(repo, task, major, from_major = None)
```

Moves a task to a new major version, such as `Docker@1` to `Docker@2`.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `task` | The task name, such as `PublishPipelineArtifact`. |
| `major` | The new major version, such as `1`. |
| `from_major` | When set, only this version is replaced. |

**Returns:** The list of paths that changed.

**Example**

```python
bump_task(repo, "Docker", "2")
# returns ["azure-pipelines.yml"]
```

=== "Before"

    `azure-pipelines.yml`

    ```yaml
    steps:
      - task: Docker@1
        inputs:
          command: build
      - task: DockerCompose@0
    ```

=== "After"

    `azure-pipelines.yml`

    ```yaml
    steps:
      - task: Docker@2
        inputs:
          command: build
      - task: DockerCompose@0
    ```

**Used in:** [`azure-ubuntu-24`](../examples/azure-ubuntu-24.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/azure_pipelines.star#L75)

## set_template_ref

```python
set_template_ref(repo, name, ref)
```

Sets the `ref` of a template repository under `resources.repositories`.

A newer version already there is kept, so a campaign never lowers one.

The repository is matched by its `name` (the project/repo path), and a
missing `ref` is added.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `name` | The repository's `name`, such as `acme/pipeline-templates`. |
| `ref` | The new ref, such as `refs/tags/v2.0.0`. |

**Returns:** The list of paths that changed.

**Example**

```python
set_template_ref(repo, "acme/pipeline-templates", "refs/tags/v2.0.0")
# returns ["azure-pipelines.yml"]
```

=== "Before"

    `azure-pipelines.yml`

    ```yaml
    resources:
      repositories:
        - repository: templates
          type: github
          name: acme/pipeline-templates
          ref: refs/tags/v1.4.0
    ```

=== "After"

    `azure-pipelines.yml`

    ```yaml
    resources:
      repositories:
        - repository: templates
          type: github
          name: acme/pipeline-templates
          ref: refs/tags/v2.0.0
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/azure_pipelines.star#L104)
