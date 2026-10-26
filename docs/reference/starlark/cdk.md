---
title: cdk.star
---

# cdk.star

Set AWS CDK context values, such as feature flags, in cdk.json.

Edits keep cdk.json's key order and indentation. The aws-cdk-lib version
itself lives in package.json, pyproject.toml, pom.xml or go.mod; use
`npm.set_dependency`, `python.set_requirement`, `maven.set_dependency` or a
`go get` step for it.

```python
load("@fleet//cdk.star", "apps", "set_context")
```

| Function | What it does |
| --- | --- |
| [`apps`](#apps) | Lists the repo's CDK apps. |
| [`set_context`](#set_context) | Sets a context value, such as a feature flag, in every cdk.json. |

## apps

```python
apps(repo)
```

Lists the repo's CDK apps.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The sorted paths of every cdk.json outside node_modules and cdk.out.

**Example**

```python
apps(repo)
# returns ["cdk.json", "infra/cdk.json"]
```

=== "Before"

    `cdk.json`

    ```json
    {"app": "npx ts-node bin/app.ts"}
    ```

    `infra/cdk.json`

    ```json
    {"app": "python3 app.py"}
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/cdk.star#L11)

## set_context

```python
set_context(repo, key, value, only_existing = False)
```

Sets a context value, such as a feature flag, in every cdk.json.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `key` | The context key, such as `@aws-cdk/aws-lambda:recognizeLayerVersion`. |
| `value` | Any JSON value. |
| `only_existing` | When True, only apps that already set the key change. |

**Returns:** The list of paths that changed.

**Example**

```python
set_context(repo, "@aws-cdk/aws-s3:createDefaultLoggingPolicy", True)
# returns ["cdk.json"]
```

=== "Before"

    `cdk.json`

    ```json
    {
      "app": "npx ts-node bin/app.ts",
      "context": {
        "@aws-cdk/core:stackRelativeExports": true
      }
    }
    ```

=== "After"

    `cdk.json`

    ```json
    {
      "app": "npx ts-node bin/app.ts",
      "context": {
        "@aws-cdk/core:stackRelativeExports": true,
        "@aws-cdk/aws-s3:createDefaultLoggingPolicy": true
      }
    }
    ```

**Used in:** [`cdk-feature-flags`](https://github.com/rossedman/gh-fleet/tree/main/examples/cdk-feature-flags)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/cdk.star#L28)
