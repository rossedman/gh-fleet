---
title: node-version
---

# node-version

Run every Node project on Node 22.

**Uses:** [`gha.set_input`](../starlark/gha.md#set_input), [`runtime.set_version`](../starlark/runtime.md#set_version)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from node-version --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//gha.star", "set_input")
    load("@fleet//runtime.star", "set_version")

    def change(repo):
        version = repo.vars["node"]
        set_version(repo, "node", version, create = ".nvmrc")
        set_input(repo, "node-version", version, match = r"^\d")
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: node-22
    description: Run every Node project on Node 22

    targets:
      orgs: [acme]
      include:
        languages: [TypeScript, JavaScript]

    change:
      steps:
        # Updates .nvmrc, .node-version and .tool-versions where they exist,
        # creates .nvmrc where nothing pins Node, and sets node-version for
        # actions/setup-node.
        - name: node version
          starlark: ./change.star
          timeout: 1m
      env:
        NODE: "22"

    pr:
      title: "build: node {{ .Vars.node }}"
      body: |
        Pins Node {{ .Vars.node }} for {{ .Repo.Name }} in its version files and
        CI, so local and CI builds use the same runtime.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet]

    vars:
      default:
        node: "22"
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/node-version)
