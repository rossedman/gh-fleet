---
title: cocoapods-ios-15
---

# cocoapods-ios-15

Raise every iOS app to iOS 15 and Alamofire 5.9.

**Uses:** [`cocoapods.set_platform`](../starlark/cocoapods.md#set_platform), [`cocoapods.set_pod`](../starlark/cocoapods.md#set_pod)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from cocoapods-ios-15 --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//cocoapods.star", "set_platform", "set_pod")

    def change(repo):
        set_platform(repo, "ios", env["IOS"])
        set_pod(repo, "Alamofire", "~> 5.9")
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: ios-15-minimum
    description: Raise every iOS app to iOS 15 and Alamofire 5.9

    targets:
      orgs: [acme]
      include:
        topics: [ios]
        files: [Podfile]

    change:
      steps:
        # Raises `platform :ios` to 15.0 (never lowering it) and Alamofire to
        # ~> 5.9 in every Podfile.
        - name: cocoapods-ios-15
          starlark: ./change.star
          timeout: 1m
      env:
        IOS: "15.0"

    pr:
      title: "build: iOS 15 and Alamofire 5.9"
      body: |
        Raises {{ .Repo.Name }}'s deployment target to iOS 15 and Alamofire to 5.9.
        Run `pod install` on a Mac to refresh Podfile.lock.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet, dependencies]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/cocoapods-ios-15)
