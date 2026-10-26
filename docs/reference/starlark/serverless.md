---
title: serverless.star
---

# serverless.star

Move Serverless Framework services to a new Lambda runtime.

AWS retires Lambda runtimes on a schedule, so this is a recurring
fleet-wide change. Edits are line-based, so comments and variables in
serverless.yml stay as they were.

```python
load("@fleet//serverless.star", "files", "set_runtime")
```

| Function | What it does |
| --- | --- |
| [`files`](#files) | Lists the repo's Serverless configs. |
| [`set_runtime`](#set_runtime) | Sets the Lambda runtime of the provider and of every function that overrides it. |

## files

```python
files(repo)
```

Lists the repo's Serverless configs.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The sorted paths of every serverless.yml and serverless.yaml
outside node_modules.

**Example**

```python
files(repo)
# returns ["serverless.yml", "services/jobs/serverless.yaml"]
```

=== "Before"

    `serverless.yml`

    ```yaml
    service: api
    ```

    `services/jobs/serverless.yaml`

    ```yaml
    service: jobs
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/serverless.star#L10)

## set_runtime

```python
set_runtime(repo, runtime, from_runtime = None)
```

Sets the Lambda runtime of the provider and of every function that overrides it.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `runtime` | The new runtime, such as `nodejs20.x` or `python3.12`. |
| `from_runtime` | When set, only this runtime is replaced, so functions on other languages stay as they are. |

**Returns:** The list of paths that changed.

**Example**

```python
set_runtime(repo, "nodejs20.x", from_runtime = "nodejs16.x")
# returns ["serverless.yml"]
```

=== "Before"

    `serverless.yml`

    ```yaml
    service: api
    provider:
      name: aws
      runtime: nodejs16.x  # retired
    functions:
      hello:
        handler: handler.hello
      report:
        handler: report.main
        runtime: python3.12
    ```

=== "After"

    `serverless.yml`

    ```yaml
    service: api
    provider:
      name: aws
      runtime: nodejs20.x  # retired
    functions:
      hello:
        handler: handler.hello
      report:
        handler: report.main
        runtime: python3.12
    ```

**Used in:** [`serverless-runtime`](../examples/serverless-runtime.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/serverless.star#L28)
