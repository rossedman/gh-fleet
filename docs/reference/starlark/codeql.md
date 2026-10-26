---
title: codeql.star
---

# codeql.star

Set up CodeQL code scanning: add a workflow, add languages and move the action.

Edits to existing workflows are line-based, so comments and layout stay
as they were.

```python
load("@fleet//codeql.star", "workflows", "languages", "add_language", "set_version", "add_workflow")
```

| Function | What it does |
| --- | --- |
| [`workflows`](#workflows) | Lists the workflows that run CodeQL. |
| [`languages`](#languages) | Lists the languages CodeQL analyzes, across every CodeQL workflow. |
| [`add_language`](#add_language) | Adds a language to every CodeQL workflow that doesn't analyze it yet. |
| [`set_version`](#set_version) | Moves every `github/codeql-action` action (`init`, `analyze`, `upload-sarif`...) to a new version. |
| [`add_workflow`](#add_workflow) | Writes a CodeQL workflow, unless the repo already runs CodeQL. |

## workflows

```python
workflows(repo)
```

Lists the workflows that run CodeQL.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The sorted paths of workflows that use any `github/codeql-action`
action.

**Example**

```python
workflows(repo)
# returns [".github/workflows/codeql.yml"]
```

=== "Before"

    `.github/workflows/codeql.yml`

    ```yaml
    steps:
      - uses: github/codeql-action/init@v3
    ```

    `.github/workflows/ci.yml`

    ```yaml
    steps:
      - uses: actions/checkout@v4
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/codeql.star#L37)

## languages

```python
languages(repo)
```

Lists the languages CodeQL analyzes, across every CodeQL workflow.

Reads `language` matrix lists (flow or block), matrix `include` entries,
and literal `languages` inputs. Older names are reported by their
current ones, so `javascript` is `javascript-typescript`.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The sorted, distinct language names.

**Example**

```python
languages(repo)
# returns ["go", "javascript-typescript"]
```

=== "Before"

    `.github/workflows/codeql.yml`

    ```yaml
    jobs:
      analyze:
        strategy:
          matrix:
            language: ["go", "javascript"]
        steps:
          - uses: github/codeql-action/init@v3
            with:
              languages: ${{ matrix.language }}
    ```

**Used in:** [`codeql-setup`](https://github.com/rossedman/gh-fleet/tree/main/examples/codeql-setup)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/codeql.star#L81)

## add_language

```python
add_language(repo, language, build_mode = None)
```

Adds a language to every CodeQL workflow that doesn't analyze it yet.

Appends to a `language` matrix list (keeping its quotes), to the
matrix `include` entries (with a `build-mode` when the others have one),
or to a literal `languages` input. Older names count as their current
ones, so a workflow with `javascript` already has
`javascript-typescript`.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `language` | The CodeQL language, such as `python` or `actions`. |
| `build_mode` | The `build-mode` for a new `include` entry. Defaults to `autobuild` for c-cpp, go and swift, and `none` otherwise. |

**Returns:** The list of paths that changed.

**Example**

```python
add_language(repo, "actions")
# returns [".github/workflows/codeql.yml"]
```

=== "Before"

    `.github/workflows/codeql.yml`

    ```yaml
    jobs:
      analyze:
        strategy:
          matrix:
            include:
              - language: go
                build-mode: autobuild
        steps:
          - uses: github/codeql-action/init@v4
    ```

=== "After"

    `.github/workflows/codeql.yml`

    ```yaml
    jobs:
      analyze:
        strategy:
          matrix:
            include:
              - language: go
                build-mode: autobuild
              - language: actions
                build-mode: none
        steps:
          - uses: github/codeql-action/init@v4
    ```

**Used in:** [`codeql-setup`](https://github.com/rossedman/gh-fleet/tree/main/examples/codeql-setup)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/codeql.star#L158)

## set_version

```python
set_version(repo, to, from_ref = None)
```

Moves every `github/codeql-action` action (`init`, `analyze`, `upload-sarif`...) to a new version.

Refs pinned to a commit SHA are left alone.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `to` | The new ref, such as `v4`. |
| `from_ref` | When set, only this exact ref is replaced. |

**Returns:** The list of paths that changed.

**Example**

```python
set_version(repo, "v4")
# returns [".github/workflows/codeql.yml"]
```

=== "Before"

    `.github/workflows/codeql.yml`

    ```yaml
    steps:
      - uses: github/codeql-action/init@v3
      - uses: github/codeql-action/analyze@v3
    ```

=== "After"

    `.github/workflows/codeql.yml`

    ```yaml
    steps:
      - uses: github/codeql-action/init@v4
      - uses: github/codeql-action/analyze@v4
    ```

**Used in:** [`codeql-setup`](https://github.com/rossedman/gh-fleet/tree/main/examples/codeql-setup)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/codeql.star#L209)

## add_workflow

```python
add_workflow(
    repo,
    languages,
    branch = None,
    cron = "30 1 * * 1",
    path = ".github/workflows/codeql.yml",
    versions = {},
)
```

Writes a CodeQL workflow, unless the repo already runs CodeQL.

The workflow runs on pushes and pull requests to `branch` and weekly,
with one matrix entry per language and least-privilege permissions.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `languages` | The CodeQL languages, such as `["go", "actions"]`. Each gets `build-mode: autobuild` for c-cpp, go and swift, and `none` otherwise. |
| `branch` | The branch to scan; defaults to the repo's default branch, or `main` when it isn't known. |
| `cron` | The weekly schedule. |
| `path` | Where to write the workflow. |
| `versions` | Refs to use instead of the defaults, keyed by action: `actions/checkout` (v7) and `github/codeql-action` (v4). |

**Returns:** True when the workflow was written; False when a workflow already
uses CodeQL.

**Example**

```python
add_workflow(repo, ["go", "actions"], branch = "main")
# returns True
```

=== "After"

    `.github/workflows/codeql.yml`

    ```yaml
    name: CodeQL

    on:
      push:
        branches: [main]
      pull_request:
        branches: [main]
      schedule:
        - cron: "30 1 * * 1"

    permissions:
      contents: read

    jobs:
      analyze:
        name: Analyze (${{ matrix.language }})
        runs-on: ubuntu-latest
        permissions:
          actions: read
          contents: read
          security-events: write
        strategy:
          fail-fast: false
          matrix:
            include:
              - language: go
                build-mode: autobuild
              - language: actions
                build-mode: none
        steps:
          - uses: actions/checkout@v7
          - uses: github/codeql-action/init@v4
            with:
              languages: ${{ matrix.language }}
              build-mode: ${{ matrix.build-mode }}
          - uses: github/codeql-action/analyze@v4
            with:
              category: /language:${{ matrix.language }}
    ```

**Used in:** [`codeql-setup`](https://github.com/rossedman/gh-fleet/tree/main/examples/codeql-setup)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/codeql.star#L235)
