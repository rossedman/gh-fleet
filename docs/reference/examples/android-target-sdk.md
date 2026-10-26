---
title: android-target-sdk
---

# android-target-sdk

Meet the Google Play target API level for every Android app.

**Uses:** [`android.set_sdk`](../starlark/android.md#set_sdk)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from android-target-sdk --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//android.star", "set_sdk")

    def change(repo):
        level = int(env["SDK"])
        set_sdk(repo, compile = level, target = level)
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: android-target-35
    description: Meet the Google Play target API level for every Android app

    targets:
      orgs: [acme]
      include:
        topics: [android]

    change:
      steps:
        # Raises compileSdk and targetSdk to 35 in every module (Groovy, Kotlin
        # DSL or a version catalog), never lowering a module already ahead.
        - name: android-target-sdk
          starlark: ./change.star
          timeout: 1m
      env:
        SDK: "35"

    pr:
      title: "build: target Android API 35"
      body: |
        Raises {{ .Repo.Name }} to compileSdk and targetSdk 35, which Google Play
        requires for app updates. Test on an API 35 device before merging.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/android-target-sdk)
