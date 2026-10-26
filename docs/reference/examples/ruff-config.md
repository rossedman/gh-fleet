---
title: ruff-config
---

# ruff-config

Give every Python repo the org's ruff settings.

**Uses:** [`toml.get`](../starlark/toml.md#get), [`toml.set`](../starlark/toml.md#set)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from ruff-config --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//toml.star", "get", "set")

    def change(repo):
        if not repo.exists("pyproject.toml"):
            return

        # A project that already picked a line length keeps it.
        if get(repo, "pyproject.toml", "tool.ruff", "line-length") == None:
            set(repo, "pyproject.toml", "tool.ruff", "line-length", 100)

        # extend-select adds the shared rules to whatever the project selects,
        # instead of replacing its own list.
        set(repo, "pyproject.toml", "tool.ruff.lint", "extend-select", ["E", "F", "I"])
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: ruff-settings
    description: Give every Python repo the org's ruff settings

    targets:
      orgs: [acme]
      include:
        languages: [Python]
        files: [pyproject.toml]

    change:
      steps:
        # Adds the shared rules to [tool.ruff.lint] extend-select, and sets
        # [tool.ruff] line-length where the project hasn't picked one, adding the
        # tables if needed, without touching comments or the rest of the file.
        - name: pyproject.toml
          starlark: ./change.star
          timeout: 1m

    pr:
      title: "chore: standard ruff settings"
      body: |
        Applies the org's ruff settings to {{ .Repo.Name }}: pyflakes,
        pycodestyle and import-sorting rules on top of the project's own, and
        100-character lines unless it already sets a line length.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet]
    ```

## Test cases

`gh fleet test` runs the change on each case's `before/` files and checks the result against `after/`. See [Testing a campaign](../../guides/testing.md).

### no-ruff-yet

=== "Before"

    `pyproject.toml`

    ```toml
    [project]
    name = "api"
    ```

=== "After"

    `pyproject.toml`

    ```toml
    [project]
    name = "api"

    [tool.ruff]
    line-length = 100

    [tool.ruff.lint]
    extend-select = ["E", "F", "I"]
    ```

### project-rules-kept

=== "Before"

    `pyproject.toml`

    ```toml
    [tool.ruff]
    line-length = 120
    target-version = 'py310'

    [tool.ruff.lint]
    select = [
        'F',      # Pyflakes
        'E',      # pycodestyle (Error)
        'D',      # pydocstyle
    ]
    ```

=== "After"

    `pyproject.toml`

    ```toml
    [tool.ruff]
    line-length = 120
    target-version = 'py310'

    [tool.ruff.lint]
    select = [
        'F',      # Pyflakes
        'E',      # pycodestyle (Error)
        'D',      # pydocstyle
    ]
    extend-select = ["E", "F", "I"]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/ruff-config)
