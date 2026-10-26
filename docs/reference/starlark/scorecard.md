---
title: scorecard.star
---

# scorecard.star

Set up OpenSSF Scorecard: add the workflow, move the action and add the README badge.

```python
load("@fleet//scorecard.star", "workflow", "set_version", "add_workflow", "add_badge")
```

| Function | What it does |
| --- | --- |
| [`workflow`](#workflow) | Finds the workflow that runs Scorecard. |
| [`set_version`](#set_version) | Moves `ossf/scorecard-action` to a new version. |
| [`add_workflow`](#add_workflow) | Writes the Scorecard workflow, unless the repo already runs Scorecard. |
| [`add_badge`](#add_badge) | Adds the OpenSSF Scorecard badge to the README, next to its other badges. |

## workflow

```python
workflow(repo)
```

Finds the workflow that runs Scorecard.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The path of the first workflow that uses `ossf/scorecard-action`,
or None.

**Example**

```python
workflow(repo)
# returns ".github/workflows/scorecards.yml"
```

=== "Before"

    `.github/workflows/scorecards.yml`

    ```yaml
    steps:
      - uses: ossf/scorecard-action@v2.3.1
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/scorecard.star#L13)

## set_version

```python
set_version(repo, to, from_ref = None)
```

Moves `ossf/scorecard-action` to a new version.

Refs pinned to a commit SHA are left alone.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `to` | The new ref, such as `v2.5.0`. |
| `from_ref` | When set, only this exact ref is replaced. |

**Returns:** The list of paths that changed.

**Example**

```python
set_version(repo, "v2.5.0")
# returns [".github/workflows/scorecard.yml"]
```

=== "Before"

    `.github/workflows/scorecard.yml`

    ```yaml
    steps:
      - uses: ossf/scorecard-action@v2.3.1 # pinned by hand
    ```

=== "After"

    `.github/workflows/scorecard.yml`

    ```yaml
    steps:
      - uses: ossf/scorecard-action@v2.5.0 # pinned by hand
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/scorecard.star#L33)

## add_workflow

```python
add_workflow(
    repo,
    branch = None,
    cron = "30 1 * * 6",
    publish = True,
    path = ".github/workflows/scorecard.yml",
    versions = {},
)
```

Writes the Scorecard workflow, unless the repo already runs Scorecard.

The workflow follows the one the Scorecard project recommends: it runs
on pushes to `branch`, on branch protection changes and weekly, and
uploads results to code scanning.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `branch` | The branch to analyze; defaults to the repo's default branch, or `main` when it isn't known. |
| `cron` | The weekly schedule. |
| `publish` | Publish results to the public Scorecard API, which the badge needs. Private repos should pass False. |
| `path` | Where to write the workflow. |
| `versions` | Refs to use instead of the defaults, keyed by action: `actions/checkout` (v7), `actions/upload-artifact` (v7), `github/codeql-action` (v4) and `ossf/scorecard-action` (v2.5.0). |

**Returns:** True when the workflow was written; False when one already runs
Scorecard.

**Example**

```python
add_workflow(repo, branch = "main", publish = False)
# returns True
```

=== "After"

    `.github/workflows/scorecard.yml`

    ```yaml
    name: Scorecard

    on:
      branch_protection_rule:
      push:
        branches: [main]
      schedule:
        - cron: "30 1 * * 6"

    permissions: read-all

    jobs:
      analysis:
        name: Scorecard analysis
        runs-on: ubuntu-latest
        permissions:
          security-events: write
        steps:
          - uses: actions/checkout@v7
            with:
              persist-credentials: false
          - uses: ossf/scorecard-action@v2.5.0
            with:
              results_file: results.sarif
              results_format: sarif
              publish_results: false
          - uses: actions/upload-artifact@v7
            with:
              name: SARIF file
              path: results.sarif
              retention-days: 5
          - uses: github/codeql-action/upload-sarif@v4
            with:
              sarif_file: results.sarif
    ```

**Used in:** [`scorecard-rollout`](https://github.com/rossedman/gh-fleet/tree/main/examples/scorecard-rollout)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/scorecard.star#L57)

## add_badge

```python
add_badge(repo, path = "README.md")
```

Adds the OpenSSF Scorecard badge to the README, next to its other badges.

The badge needs published results (`add_workflow(publish = True)`). It
goes where `markdown.ensure_badge` puts badges: after the badges the
README already shows, else under its title.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `path` | The README to edit. |

**Returns:** True when the badge was added; False when the README already links
to the repo's Scorecard results, or doesn't exist.

**Example**

```python
add_badge(repo)
# returns True
```

=== "Before"

    `README.md`

    ```markdown
    # api

    [![CI](https://github.com/acme/api/actions/workflows/ci.yml/badge.svg)](https://github.com/acme/api/actions)

    The API service.
    ```

=== "After"

    `README.md`

    ```markdown
    # api

    [![CI](https://github.com/acme/api/actions/workflows/ci.yml/badge.svg)](https://github.com/acme/api/actions)
    [![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/acme/api/badge)](https://scorecard.dev/viewer/?uri=github.com/acme/api)

    The API service.
    ```

**Used in:** [`scorecard-rollout`](https://github.com/rossedman/gh-fleet/tree/main/examples/scorecard-rollout)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/scorecard.star#L162)
