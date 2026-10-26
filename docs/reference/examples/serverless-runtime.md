---
title: serverless-runtime
---

# serverless-runtime

Move Serverless services off the retired Node 16 Lambda runtime.

**Uses:** [`serverless.set_runtime`](../starlark/serverless.md#set_runtime)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from serverless-runtime --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//serverless.star", "set_runtime")

    def change(repo):
        set_runtime(repo, env["TO"], from_runtime = env["FROM"])
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: lambda-node-20
    description: Move Serverless services off the retired Node 16 Lambda runtime

    targets:
      orgs: [acme]
      include:
        files: [serverless.yml]

    change:
      steps:
        # Sets nodejs16.x to nodejs20.x on the provider and on every function that
        # overrides it, leaving functions on other languages alone.
        - name: serverless-runtime
          starlark: ./change.star
          timeout: 1m
      env:
        FROM: nodejs16.x
        TO: nodejs20.x

    pr:
      title: "deploy: nodejs20.x Lambda runtime"
      body: |
        Moves {{ .Repo.Name }}'s Lambda functions off nodejs16.x, which AWS
        has retired, to nodejs20.x.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet, deploy]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/serverless-runtime)
