---
title: swift-package-bump
---

# swift-package-bump

Move Swift packages to swift-log 1.6 and tools version 5.10.

**Uses:** [`swift.set_package`](../starlark/swift.md#set_package), [`swift.set_tools_version`](../starlark/swift.md#set_tools_version)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from swift-package-bump --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//swift.star", "set_package", "set_tools_version")

    def change(repo):
        if set_package(repo, env["URL"], env["VERSION"]):
            set_tools_version(repo, "5.10")
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: swift-log-1-6
    description: Move Swift packages to swift-log 1.6 and tools version 5.10

    targets:
      orgs: [acme]
      include:
        languages: [Swift]
        files: [Package.swift]

    change:
      steps:
        # Sets swift-log to 1.6.1, keeping each manifest's kind of requirement
        # (from:, exact:, upToNextMinor), and raises swift-tools-version to 5.10.
        - name: swift-package-bump
          starlark: ./change.star
          timeout: 1m
      env:
        URL: https://github.com/apple/swift-log
        VERSION: "1.6.1"

    pr:
      title: "build(deps): swift-log 1.6.1"
      body: |
        Updates swift-log in {{ .Repo.Name }} to 1.6.1.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet, dependencies]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/swift-package-bump)
