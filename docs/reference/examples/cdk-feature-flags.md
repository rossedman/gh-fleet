---
title: cdk-feature-flags
---

# cdk-feature-flags

Turn on a CDK feature flag in every CDK app.

**Uses:** [`cdk.set_context`](../starlark/cdk.md#set_context)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from cdk-feature-flags --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//cdk.star", "set_context")

    def change(repo):
        set_context(repo, "@aws-cdk/aws-s3:createDefaultLoggingPolicy", True)
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: cdk-s3-logging-flag
    description: Turn on a CDK feature flag in every CDK app

    targets:
      orgs: [acme]
      include:
        files: [cdk.json]

    change:
      steps:
        # Sets the flag in each cdk.json's context, keeping key order and
        # indentation. Review `cdk diff` before merging: flags change synthesized
        # resources.
        - name: cdk-feature-flags
          starlark: ./change.star
          timeout: 1m

    pr:
      title: "build: enable CDK default S3 logging policy"
      body: |
        Turns on `@aws-cdk/aws-s3:createDefaultLoggingPolicy` in {{ .Repo.Name }}.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet, deploy]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/cdk-feature-flags)
