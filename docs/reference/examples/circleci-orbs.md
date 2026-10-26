---
title: circleci-orbs
---

# circleci-orbs

Move every CircleCI config to the node orb 5.2.0.

**Uses:** [`circleci.bump_orb`](../starlark/circleci.md#bump_orb), [`circleci.orbs`](../starlark/circleci.md#orbs)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from circleci-orbs --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//circleci.star", "bump_orb", "orbs")

    def change(repo):
        old = [o.split("@")[1] for o in orbs(repo) if o.split("@")[0] == env["ORB"]]
        repo.set_var("previous", ", ".join(old) or "none")
        bump_orb(repo, env["ORB"], env["VERSION"])
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: circleci-node-orb-5-2
    description: Move every CircleCI config to the node orb 5.2.0

    targets:
      orgs: [acme]
      include:
        files: [.circleci]

    change:
      steps:
        # Rewrites `circleci/node@<any>` to 5.2.0 in every config under .circleci,
        # and reports the versions it replaced.
        - name: circleci-orbs
          starlark: ./change.star
          timeout: 1m
      env:
        ORB: circleci/node
        VERSION: "5.2.0"

    pr:
      title: "ci: circleci/node orb 5.2.0"
      body: |
        Updates the node orb in {{ .Repo.Name }} from {{ .Vars.previous }} to 5.2.0.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/circleci-orbs)
