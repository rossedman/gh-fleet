---
title: python-version
---

# python-version

Move every Python service to Python 3.12.

**Uses:** [`gha.set_input`](../starlark/gha.md#set_input), [`python.set_requires_python`](../starlark/python.md#set_requires_python), [`runtime.set_version`](../starlark/runtime.md#set_version)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from python-version --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//gha.star", "set_input")
    load("@fleet//python.star", "set_requires_python")
    load("@fleet//runtime.star", "set_version")

    def change(repo):
        version = env["PYTHON"]
        set_version(repo, "python", version)
        set_requires_python(repo, ">=" + version)
        # Only plain version pins: leave matrix expressions such as ${{ matrix.py }}
        # alone, and never lower a job that already tests a newer Python.
        set_input(repo, "python-version", version, match = r"^\d", never_lower = True)
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: python-3-12
    description: Move every Python service to Python 3.12

    targets:
      orgs: [acme]
      include:
        languages: [Python]

    change:
      steps:
        # Updates .python-version, .tool-versions, `requires-python` and the
        # python-version input of actions/setup-python, wherever each is used.
        - name: python version
          starlark: ./change.star
          timeout: 1m
      env:
        PYTHON: "3.12"

    pr:
      title: "build: python 3.12"
      body: |
        Moves {{ .Repo.Name }} to Python 3.12: the version files, the
        `requires-python` floor and the version CI installs.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      draft: true
      labels: [fleet]
    ```

## Test cases

`gh fleet test` runs the change on each case's `before/` files and checks the result against `after/`. See [Testing a campaign](../../guides/testing.md).

### already-newer

=== "Before"

    `.github/workflows/release.yml`

    ```yaml
    jobs:
      build:
        steps:
          - uses: actions/setup-python@v6
            with:
              python-version: "3.15"
              allow-prereleases: true
    ```

    `pyproject.toml`

    ```toml
    [project]
    name = "black"
    requires-python = ">=3.13"
    ```

=== "After"

    No changes: the campaign leaves this repo alone.

### upgrade

=== "Before"

    `.github/workflows/ci.yml`

    ```yaml
    jobs:
      test:
        strategy:
          matrix:
            python: ["3.10", "3.11"]
        steps:
          - uses: actions/setup-python@v5
            with:
              python-version: ${{ matrix.python }}
      lint:
        steps:
          - uses: actions/setup-python@v5
            with:
              python-version: "3.10"
    ```

    `.python-version`

    ```text
    3.11
    ```

    `pyproject.toml`

    ```toml
    [project]
    name = "api"
    requires-python = ">=3.10"
    ```

=== "After"

    `.github/workflows/ci.yml`

    ```yaml
    jobs:
      test:
        strategy:
          matrix:
            python: ["3.10", "3.11"]
        steps:
          - uses: actions/setup-python@v5
            with:
              python-version: ${{ matrix.python }}
      lint:
        steps:
          - uses: actions/setup-python@v5
            with:
              python-version: "3.12"
    ```

    `.python-version`

    ```text
    3.12
    ```

    `pyproject.toml`

    ```toml
    [project]
    name = "api"
    requires-python = ">=3.12"
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/python-version)
