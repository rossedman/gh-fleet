---
title: gitignore-entries
---

# gitignore-entries

Make sure local env files are never committed.

**Uses:** [`files.ensure_lines`](../starlark/files.md#ensure_lines)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from gitignore-entries --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//files.star", "ensure_lines")

    def change(repo):
        ensure_lines(repo, ".gitignore", env["ENTRIES"], header = "# Local environment files")
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: ignore-env-files
    description: Make sure local env files are never committed

    targets:
      orgs: [acme]
      exclude:
        topics: [no-fleet]

    change:
      steps:
        # Appends each line of ENTRIES to .gitignore unless it is already there.
        - name: gitignore
          starlark: ./change.star
          timeout: 1m
      env:
        ENTRIES: |
          .env
          .env.*
          !.env.example

    pr:
      title: "chore: ignore local .env files"
      body: |
        Adds `.env` patterns to {{ .Repo.Name }}'s `.gitignore` so local
        secrets can't be committed by accident. `.env.example` stays tracked.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet, security]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/gitignore-entries)
