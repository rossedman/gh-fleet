---
title: ansible-collections
---

# ansible-collections

Move every Ansible project to community.general 9.4.0.

**Uses:** [`ansible.set_version`](../starlark/ansible.md#set_version)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from ansible-collections --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//ansible.star", "set_version")

    def change(repo):
        set_version(repo, env["COLLECTION"], env["VERSION"])
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: community-general-9
    description: Move every Ansible project to community.general 9.4.0

    targets:
      orgs: [acme]
      include:
        search: "community.general filename:requirements.yml"

    change:
      steps:
        # Sets the version of the collection in every requirements.yml that
        # lists it, adding a version where the entry had none.
        - name: ansible-collections
          starlark: ./change.star
          timeout: 1m
      env:
        COLLECTION: community.general
        VERSION: 9.4.0

    pr:
      title: "build(deps): community.general 9.4.0"
      body: |
        Updates community.general to 9.4.0 in {{ .Repo.Name }}.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet, dependencies]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/ansible-collections)
