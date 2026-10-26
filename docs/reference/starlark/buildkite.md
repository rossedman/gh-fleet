---
title: buildkite.star
---

# buildkite.star

Edit Buildkite pipelines: plugin versions, agent queues and step defaults.

Edits are line-based, so comments, anchors and layout in pipeline files
stay as they were. Images used through the docker and docker-compose
plugins are `image:` lines, so `docker.bump_image` moves them too.

```python
load("@fleet//buildkite.star", "pipelines", "plugins", "bump_plugin", "queues", "set_queue", "set_step_defaults")
```

| Function | What it does |
| --- | --- |
| [`pipelines`](#pipelines) | Lists the repo's pipeline files. |
| [`plugins`](#plugins) | Lists every plugin a pipeline uses, with its version. |
| [`bump_plugin`](#bump_plugin) | Moves a plugin to a new version in every pipeline. |
| [`queues`](#queues) | Lists the agent queues pipelines target. |
| [`set_queue`](#set_queue) | Points `agents: queue:` at a new queue in every pipeline. |
| [`set_step_defaults`](#set_step_defaults) | Adds keys, such as `timeout_in_minutes`, to every command step that lacks them. |

## pipelines

```python
pipelines(repo)
```

Lists the repo's pipeline files.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The sorted paths of every YAML file under .buildkite, and of
buildkite.yml or buildkite.yaml at the root.

**Example**

```python
pipelines(repo)
# returns [".buildkite/deploy/pipeline.yaml", ".buildkite/pipeline.yml"]
```

=== "Before"

    `.buildkite/pipeline.yml`

    ```yaml
    steps: []
    ```

    `.buildkite/deploy/pipeline.yaml`

    ```yaml
    steps: []
    ```

    `ci/other.yml`

    ```yaml
    steps: []
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/buildkite.star#L12)

## plugins

```python
plugins(repo)
```

Lists every plugin a pipeline uses, with its version.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** A sorted, de-duplicated list of `name#ref` strings, such as
`docker-compose#v4.14.0`.

**Example**

```python
plugins(repo)
# returns ["docker-compose#v4.14.0", "ecr#v2.7.0"]
```

=== "Before"

    `.buildkite/pipeline.yml`

    ```yaml
    steps:
      - label: test
        plugins:
          - docker-compose#v4.14.0:
              run: app
          - "ecr#v2.7.0":
              login: true
    ```

**Used in:** [`buildkite-plugin-bump`](../examples/buildkite-plugin-bump.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/buildkite.star#L32)

## bump_plugin

```python
bump_plugin(repo, plugin, to, from_ref = None)
```

Moves a plugin to a new version in every pipeline.

The plugin matches as written, such as `docker-compose` or
`acme/deploy`; a full git URL works too. Plugins pinned to a commit SHA
are left alone unless `from_ref` names it.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `plugin` | The plugin name as pipelines write it. |
| `to` | The new version, such as `v5.4.1`. |
| `from_ref` | When set, only this exact version is replaced. |

**Returns:** The list of paths that changed.

**Example**

```python
bump_plugin(repo, "docker-compose", "v5.4.1")
# returns [".buildkite/pipeline.yml"]
```

=== "Before"

    `.buildkite/pipeline.yml`

    ```yaml
    steps:
      - label: test
        plugins:
          - docker-compose#v4.14.0:
              run: app
    ```

=== "After"

    `.buildkite/pipeline.yml`

    ```yaml
    steps:
      - label: test
        plugins:
          - docker-compose#v5.4.1:
              run: app
    ```

**Used in:** [`buildkite-plugin-bump`](../examples/buildkite-plugin-bump.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/buildkite.star#L60)

## queues

```python
queues(repo)
```

Lists the agent queues pipelines target.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The sorted, de-duplicated queue names.

**Example**

```python
queues(repo)
# returns ["default", "deploy"]
```

=== "Before"

    `.buildkite/pipeline.yml`

    ```yaml
    agents:
      queue: default
    steps:
      - command: make deploy
        agents: { queue: "deploy" }
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/buildkite.star#L93)

## set_queue

```python
set_queue(repo, queue, from_queue = None)
```

Points `agents: queue:` at a new queue in every pipeline.

Covers block (`queue: x` under `agents:`) and flow (`agents: { queue: x }`)
forms. Pipelines that set no queue are left alone.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `queue` | The new queue name. |
| `from_queue` | When set, only this queue is replaced, so steps on other queues stay where they are. |

**Returns:** The list of paths that changed.

**Example**

```python
set_queue(repo, "linux-small", from_queue = "default")
# returns [".buildkite/pipeline.yml"]
```

=== "Before"

    `.buildkite/pipeline.yml`

    ```yaml
    agents:
      queue: default  # shared runners
    steps:
      - command: make test
      - command: make deploy
        agents: { queue: "deploy" }
    ```

=== "After"

    `.buildkite/pipeline.yml`

    ```yaml
    agents:
      queue: linux-small  # shared runners
    steps:
      - command: make test
      - command: make deploy
        agents: { queue: "deploy" }
    ```

**Used in:** [`buildkite-agents`](../examples/buildkite-agents.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/buildkite.star#L120)

## set_step_defaults

```python
set_step_defaults(repo, values)
```

Adds keys, such as `timeout_in_minutes`, to every command step that lacks them.

A command step is a list item with a `command` or `commands` key, at
any depth, so steps inside `group` steps count. Keys a step already
sets are left as they are, so this never overrides a step's choice.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `values` | A dict of step keys to scalar values, such as `{"timeout_in_minutes": 30}`. |

**Returns:** The list of paths that changed.

**Example**

```python
set_step_defaults(repo, {"timeout_in_minutes": 20})
# returns [".buildkite/pipeline.yml"]
```

=== "Before"

    `.buildkite/pipeline.yml`

    ```yaml
    steps:
      - label: test
        command: make test
      - label: e2e
        command: make e2e
        timeout_in_minutes: 60
      - wait
    ```

=== "After"

    `.buildkite/pipeline.yml`

    ```yaml
    steps:
      - label: test
        command: make test
        timeout_in_minutes: 20
      - label: e2e
        command: make e2e
        timeout_in_minutes: 60
      - wait
    ```

**Used in:** [`buildkite-agents`](../examples/buildkite-agents.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/buildkite.star#L166)
