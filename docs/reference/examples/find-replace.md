---
title: find-replace
---

# find-replace

Point every link at the new docs domain.

**Uses:** [`files.replace_all`](../starlark/files.md#replace_all)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from find-replace --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//files.star", "replace_all")

    def change(repo):
        changed = replace_all(repo, env["FROM"], env["TO"], exclude = ["**/node_modules/**", "**/vendor/**"])
        repo.set_var("files", len(changed))
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: move-docs-domain
    description: Point every link at the new docs domain

    targets:
      orgs: [acme]
      include:
        # Only repos that mention the old domain; code search keeps this cheap.
        search: "docs.acme.io"

    change:
      steps:
        # Replaces FROM with TO, literally (no regex), in every text file.
        - name: replace
          starlark: ./change.star
          timeout: 2m
      env:
        FROM: "https://docs.acme.io"
        TO: "https://docs.acme.dev"

    pr:
      title: "docs: move links to docs.acme.dev"
      body: |
        `docs.acme.io` is being retired. This replaces every
        `https://docs.acme.io` link in {{ .Repo.Name }} with
        `https://docs.acme.dev`; paths are unchanged.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet, docs]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/find-replace)
