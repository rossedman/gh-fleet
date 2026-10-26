---
title: remove-files
---

# remove-files

Remove Travis CI config now that everything builds on Actions.

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from remove-files --org your-org
```

## Campaign

=== "change.star"

    ```python
    def change(repo):
        repo.delete(*env["REMOVE"].split())
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: retire-travis
    description: Remove Travis CI config now that everything builds on Actions

    targets:
      orgs: [acme]
      include:
        files: [.travis.yml]

    change:
      steps:
        # Deletes each path in REMOVE that exists; missing ones are skipped, so a
        # repo without any of them is a no-op. Directories go recursively.
        - name: remove
          starlark: ./change.star
          timeout: 1m
      env:
        # Space-separated paths.
        REMOVE: ".travis.yml .travis ci/travis-deploy.sh"

    pr:
      title: "ci: remove Travis CI config"
      body: |
        Travis CI no longer builds {{ .Repo.Name }}; GitHub Actions does. This
        removes the leftover Travis files so nobody edits them by mistake.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet, ci]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/remove-files)
