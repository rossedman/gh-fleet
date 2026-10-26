---
title: sync-files
---

# sync-files

Keep shared repo files identical everywhere.

**Uses:** [`files.sync`](../starlark/files.md#sync)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from sync-files --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//files.star", "sync")

    def change(repo):
        sync(repo, "files")
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: sync-standard-files
    description: Keep shared repo files identical everywhere

    targets:
      orgs: [acme]
      exclude:
        topics: [no-fleet]

    change:
      steps:
        # Copies everything under files/ (next to this campaign) into each repo,
        # replacing what is there. Edit a file once, re-run apply, and every
        # open PR is updated.
        - name: sync files
          starlark: ./change.star
          timeout: 1m

    pr:
      title: "chore: sync standard repo files"
      body: |
        Brings the shared files in {{ .Repo.FullName }} in line with the
        org standard: `.editorconfig` and the pull request template.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet]
    ```

=== "files/.editorconfig"

    ```text
    root = true

    [*]
    charset = utf-8
    end_of_line = lf
    insert_final_newline = true
    trim_trailing_whitespace = true
    indent_style = space
    indent_size = 2

    [{Makefile,*.go}]
    indent_style = tab
    ```

=== "files/.github/pull_request_template.md"

    ```markdown
    ## What and why

    ## How it was tested

    - [ ] Tests added or updated
    - [ ] Docs updated
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/sync-files)
