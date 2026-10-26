---
title: jvm-dependency-bump
---

# jvm-dependency-bump

Move Jackson to 2.17.2 in every Maven and Gradle build.

**Uses:** [`gradle.set_dependency`](../starlark/gradle.md#set_dependency), [`maven.set_dependency`](../starlark/maven.md#set_dependency)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from jvm-dependency-bump --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//gradle.star", gradle_dependency = "set_dependency")
    load("@fleet//maven.star", maven_dependency = "set_dependency")

    def change(repo):
        group, artifact, version = env["GROUP"], env["ARTIFACT"], env["VERSION"]
        changed = maven_dependency(repo, group, artifact, version)
        changed += gradle_dependency(repo, group + ":" + artifact, version)
        repo.set_var("files", ", ".join(changed))
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: bump-jackson
    description: Move Jackson to 2.17.2 in every Maven and Gradle build

    targets:
      orgs: [acme]
      include:
        languages: [Java, Kotlin]
        search: "jackson-databind"

    change:
      steps:
        # Maven: the <dependency> version, or the property it points at.
        # Gradle: string coordinates in build scripts and version catalog
        # entries, including version.ref.
        - name: jackson
          starlark: ./change.star
          timeout: 1m
      env:
        GROUP: com.fasterxml.jackson.core
        ARTIFACT: jackson-databind
        VERSION: "2.17.2"

    pr:
      title: "build(deps): jackson-databind 2.17.2"
      body: |
        Updates jackson-databind to 2.17.2 in {{ .Repo.Name }} ({{ .Vars.files }}).

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet, dependencies]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/jvm-dependency-bump)
