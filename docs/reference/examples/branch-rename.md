---
title: branch-rename
---

# branch-rename

Update workflows after renaming master to main.

**Uses:** [`gha.rename_branch`](../starlark/gha.md#rename_branch)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from branch-rename --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//gha.star", "rename_branch")

    def change(repo):
        rename_branch(repo, env["OLD"], env["NEW"])
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: master-to-main
    description: Update workflows after renaming master to main

    targets:
      orgs: [acme]
      include:
        files: [.github/workflows]

    change:
      steps:
        # Renames master in workflow trigger branch lists and refs/heads/master
        # checks. Run it after renaming the default branch, so CI keeps running
        # on pushes and pull requests.
        - name: branch-rename
          starlark: ./change.star
          timeout: 1m
      env:
        OLD: master
        NEW: main

    pr:
      title: "ci: master is now main"
      body: |
        Updates {{ .Repo.Name }}'s workflows for the default branch rename from
        `master` to `main`.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/branch-rename)
