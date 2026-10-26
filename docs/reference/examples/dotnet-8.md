---
title: dotnet-8
---

# dotnet-8

Move .NET 6 projects to .NET 8.

**Uses:** [`dotnet.set_package`](../starlark/dotnet.md#set_package), [`dotnet.set_sdk_version`](../starlark/dotnet.md#set_sdk_version), [`dotnet.set_target_framework`](../starlark/dotnet.md#set_target_framework)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from dotnet-8 --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//dotnet.star", "set_package", "set_sdk_version", "set_target_framework")

    def change(repo):
        if set_target_framework(repo, env["TO"], from_framework = env["FROM"]):
            set_sdk_version(repo, env["SDK"])
            set_package(repo, "Microsoft.AspNetCore.Mvc.Testing", "8.0.8")
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: dotnet-8
    description: Move .NET 6 projects to .NET 8

    targets:
      orgs: [acme]
      include:
        languages: [C#]

    change:
      steps:
        # Retargets net6.0 to net8.0 (leaving other targets of multi-targeted
        # projects alone), pins the 8.0 SDK in global.json where one is
        # pinned, and moves the ASP.NET Core test host to the matching major.
        - name: dotnet
          starlark: ./change.star
          timeout: 1m
      env:
        FROM: net6.0
        TO: net8.0
        SDK: "8.0.400"

    pr:
      title: "build: .NET 8"
      body: |
        Moves {{ .Repo.Name }} from .NET 6, which is out of support, to .NET 8.
        Review the breaking changes listed by Microsoft before merging.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      draft: true
      labels: [fleet]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/dotnet-8)
