---
title: cmake-minimum
---

# cmake-minimum

Require CMake 3.20 and update GoogleTest everywhere.

**Uses:** [`cmake.set_fetchcontent_tag`](../starlark/cmake.md#set_fetchcontent_tag), [`cmake.set_minimum_version`](../starlark/cmake.md#set_minimum_version)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from cmake-minimum --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//cmake.star", "set_fetchcontent_tag", "set_minimum_version")

    def change(repo):
        set_minimum_version(repo, env["CMAKE"])
        set_fetchcontent_tag(repo, "googletest", env["GTEST"])
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: cmake-3-20
    description: Require CMake 3.20 and update GoogleTest everywhere

    targets:
      orgs: [acme]
      include:
        languages: [C++, C]
        files: [CMakeLists.txt]

    change:
      steps:
        # Raises cmake_minimum_required (never lowering it, and keeping a
        # `...max` policy range) and moves the GoogleTest FetchContent tag.
        - name: cmake-minimum
          starlark: ./change.star
          timeout: 1m
      env:
        CMAKE: "3.20"
        GTEST: v1.15.2

    pr:
      title: "build: cmake 3.20 and googletest v1.15.2"
      body: |
        Requires CMake 3.20 in {{ .Repo.Name }} and moves GoogleTest to v1.15.2.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/cmake-minimum)
