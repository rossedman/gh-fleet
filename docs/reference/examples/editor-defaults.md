---
title: editor-defaults
---

# editor-defaults

Give every repo the org's editor and line-ending defaults.

**Uses:** [`editorconfig.set`](../starlark/editorconfig.md#set), [`gitattributes.set`](../starlark/gitattributes.md#set)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from editor-defaults --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//editorconfig.star", "set")
    load("@fleet//gitattributes.star", gitattributes = "set")

    def change(repo):
        set(repo, "", "root", True)
        set(repo, "*", "end_of_line", "lf")
        set(repo, "*", "insert_final_newline", True)
        set(repo, "Makefile", "indent_style", "tab")
        gitattributes(repo, "*", "text=auto")
        gitattributes(repo, "*.sh", "text eol=lf")
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: editor-defaults
    description: Give every repo the org's editor and line-ending defaults

    targets:
      orgs: [acme]
      exclude:
        topics: [no-fleet]

    change:
      steps:
        # Sets .editorconfig defaults (root, LF endings, final newline, tabs in
        # Makefiles) and .gitattributes rules (text=auto, LF for shell scripts),
        # keeping anything a repo already sets for other patterns.
        - name: editor-defaults
          starlark: ./change.star
          timeout: 1m

    pr:
      title: "chore: editor and line-ending defaults"
      body: |
        Adds the org's editor defaults to {{ .Repo.Name }}, so every editor
        writes the same line endings and indentation.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/editor-defaults)
