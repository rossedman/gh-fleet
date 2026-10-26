---
title: packer-plugins
---

# packer-plugins

Move Packer templates to the current amazon plugin.

**Uses:** [`packer.set_plugin_version`](../starlark/packer.md#set_plugin_version)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from packer-plugins --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//packer.star", "set_plugin_version")

    def change(repo):
        set_plugin_version(repo, env["SOURCE"], env["CONSTRAINT"])
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: packer-amazon-plugin
    description: Move Packer templates to the current amazon plugin

    targets:
      orgs: [acme]
      include:
        search: "github.com/hashicorp/amazon extension:hcl"

    change:
      steps:
        # Sets the version constraint of the amazon plugin in required_plugins,
        # in block and one-line forms.
        - name: packer-plugins
          starlark: ./change.star
          timeout: 1m
      env:
        SOURCE: github.com/hashicorp/amazon
        CONSTRAINT: ">= 1.3.3"

    pr:
      title: "build: packer amazon plugin >= 1.3.3"
      body: |
        Moves {{ .Repo.Name }}'s Packer templates to the amazon plugin >= 1.3.3.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet, dependencies]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/packer-plugins)
