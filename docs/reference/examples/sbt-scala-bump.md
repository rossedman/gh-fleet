---
title: sbt-scala-bump
---

# sbt-scala-bump

Move Scala 2.13 builds to 2.13.15 and sbt 1.10.2.

**Uses:** [`sbt.build_files`](../starlark/sbt.md#build_files), [`sbt.set_sbt_version`](../starlark/sbt.md#set_sbt_version), [`sbt.set_scala_version`](../starlark/sbt.md#set_scala_version)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from sbt-scala-bump --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//sbt.star", "build_files", "set_sbt_version", "set_scala_version")

    def change(repo):
        want = env["SCALA"]
        old = set()
        for path in build_files(repo):
            for v in re.find_all(r"2\.13\.\d+", repo.read(path)):
                if v != want:
                    old.add(v)
        for v in sorted(old):
            set_scala_version(repo, want, from_version = v)
        repo.set_var("scala", want)
        set_sbt_version(repo, env["SBT"])
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: scala-2-13-15
    description: Move Scala 2.13 builds to 2.13.15 and sbt 1.10.2

    targets:
      orgs: [acme]
      include:
        languages: [Scala]
        files: [build.sbt]

    change:
      steps:
        # Replaces Scala 2.13.x with 2.13.15 in scalaVersion and crossScalaVersions
        # (cross-built Scala 3 versions stay), and sets sbt.version.
        - name: sbt-scala-bump
          starlark: ./change.star
          timeout: 1m
      env:
        SCALA: 2.13.15
        SBT: 1.10.2

    pr:
      title: "build: scala 2.13.15 and sbt 1.10.2"
      body: |
        Moves {{ .Repo.Name }} to Scala {{ .Vars.scala }} and sbt 1.10.2.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet, dependencies]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/sbt-scala-bump)
