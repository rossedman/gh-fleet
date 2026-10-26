---
title: bazel-module-bump
---

# bazel-module-bump

Move Bazel repos to Bazel 7.4.1 and rules_go 0.50.1.

**Uses:** [`bazel.set_dep`](../starlark/bazel.md#set_dep), [`bazel.set_version`](../starlark/bazel.md#set_version)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from bazel-module-bump --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//bazel.star", "set_dep", "set_version")

    def change(repo):
        set_version(repo, env["BAZEL"])
        set_dep(repo, "rules_go", env["RULES_GO"])
        set_dep(repo, "gazelle", env["GAZELLE"])
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: bazel-7-rules-go
    description: Move Bazel repos to Bazel 7.4.1 and rules_go 0.50.1

    targets:
      orgs: [acme]
      include:
        files: [MODULE.bazel]

    change:
      steps:
        # Pins Bazel 7.4.1 in .bazelversion where the repo pins one, and moves the
        # rules_go and gazelle bazel_deps, whatever order their arguments are in.
        - name: bazel-module-bump
          starlark: ./change.star
          timeout: 1m
      env:
        BAZEL: 7.4.1
        RULES_GO: 0.50.1
        GAZELLE: 0.39.1

    pr:
      title: "build: bazel 7.4.1 and rules_go 0.50.1"
      body: |
        Moves {{ .Repo.Name }} to Bazel 7.4.1, rules_go 0.50.1 and gazelle 0.39.1.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet, dependencies]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/bazel-module-bump)
