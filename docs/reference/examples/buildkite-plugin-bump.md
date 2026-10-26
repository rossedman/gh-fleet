---
title: buildkite-plugin-bump
---

# buildkite-plugin-bump

Move every pipeline to docker-compose plugin v5.

**Uses:** [`buildkite.bump_plugin`](../starlark/buildkite.md#bump_plugin), [`buildkite.plugins`](../starlark/buildkite.md#plugins)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from buildkite-plugin-bump --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//buildkite.star", "bump_plugin", "plugins")

    def change(repo):
        name = env["PLUGIN"]
        old = [p.split("#")[1] for p in plugins(repo) if p.split("#")[0] == name]
        repo.set_var("previous", ", ".join(old) or "none")
        bump_plugin(repo, name, env["VERSION"])
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: docker-compose-plugin-v5
    description: Move every pipeline to docker-compose plugin v5

    targets:
      orgs: [acme]
      include:
        files: [.buildkite]

    change:
      steps:
        # Rewrites `docker-compose#v4.x` to the new version in every pipeline
        # under .buildkite, keeping comments and layout. Plugins pinned to a
        # commit SHA are left alone.
        - name: buildkite-plugin-bump
          starlark: ./change.star
          timeout: 1m
      env:
        PLUGIN: docker-compose
        VERSION: v5.4.1

    pr:
      title: "ci: docker-compose plugin v5.4.1"
      body: |
        Moves {{ .Repo.Name }}'s pipelines from {{ .Vars.previous }} to the
        docker-compose plugin v5.4.1.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/buildkite-plugin-bump)
