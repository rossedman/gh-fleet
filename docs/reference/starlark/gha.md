---
title: gha.star
---

# gha.star

Find GitHub Actions workflows, bump action versions, set inputs and add permissions.

Edits are line-based rather than a YAML round trip, so comments, quoting
and formatting stay exactly as they were.

```python
load("@fleet//gha.star", "workflows", "action_files", "bump_action", "set_input", "ensure_permissions", "unpinned", "pin", "rename_branch")
```

| Function | What it does |
| --- | --- |
| [`workflows`](#workflows) | Lists the repo's workflow files. |
| [`action_files`](#action_files) | Lists the workflows and every local action's metadata file. |
| [`bump_action`](#bump_action) | Moves an action to a new version wherever a workflow or local action uses it. |
| [`set_input`](#set_input) | Sets every `<key>: <value>` line in the workflows, such as an action input. |
| [`ensure_permissions`](#ensure_permissions) | Adds a top-level `permissions` block to each workflow that has none. |
| [`unpinned`](#unpinned) | Lists each remote action used at a tag or branch rather than a commit SHA. |
| [`pin`](#pin) | Pins actions to commit SHAs, keeping the old ref as a comment. |
| [`rename_branch`](#rename_branch) | Renames a branch in workflow triggers and branch checks, as when `master` becomes `main`. |

## workflows

```python
workflows(repo)
```

Lists the repo's workflow files.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The sorted paths of every .yml and .yaml file directly under
.github/workflows.

**Example**

```python
workflows(repo)
# returns [".github/workflows/ci.yml", ".github/workflows/release.yaml"]
```

=== "Before"

    `.github/workflows/ci.yml`

    ```yaml
    on: push
    ```

    `.github/workflows/release.yaml`

    ```yaml
    on: release
    ```

    `docs/ci.yml`

    ```yaml
    not a workflow
    ```

**Used in:** [`runner-label`](https://github.com/rossedman/gh-fleet/tree/main/examples/runner-label)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/gha.star#L11)

## action_files

```python
action_files(repo)
```

Lists the workflows and every local action's metadata file.

Local actions are the action.yml or action.yaml files a repo keeps for
its own workflows, often under .github/actions.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The sorted paths of the workflows and the action.yml and action.yaml
files anywhere in the repo.

**Example**

```python
action_files(repo)
# returns [".github/actions/setup/action.yml", ".github/workflows/ci.yml"]
```

=== "Before"

    `.github/workflows/ci.yml`

    ```yaml
    on: push
    ```

    `.github/actions/setup/action.yml`

    ```yaml
    runs:
      using: composite
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/gha.star#L31)

## bump_action

```python
bump_action(repo, action, to, from_ref = None)
```

Moves an action to a new version wherever a workflow or local action uses it.

Steps that use a sub-path of the action (`owner/repo/path`) match too.
Refs pinned to a full commit SHA are left alone; use `pin` for those.
Quotes and trailing comments are kept.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `action` | The action as `owner/repo`, such as `actions/checkout`. |
| `to` | The new ref, such as `v4`. |
| `from_ref` | When set, only this exact ref is replaced, so `v3` never matches `v3.5.0`; otherwise every non-SHA ref is. |

**Returns:** The list of paths that changed.

**Example**

```python
bump_action(repo, "actions/checkout", "v4", from_ref = "v3")
# returns [".github/workflows/ci.yml"]
```

=== "Before"

    `.github/workflows/ci.yml`

    ```yaml
    jobs:
      test:
        steps:
          - uses: actions/checkout@v3
          - uses: actions/checkout@v3.5.0
          - uses: actions/setup-go@v5
    ```

=== "After"

    `.github/workflows/ci.yml`

    ```yaml
    jobs:
      test:
        steps:
          - uses: actions/checkout@v4
          - uses: actions/checkout@v3.5.0
          - uses: actions/setup-go@v5
    ```

**Used in:** [`upgrade-action`](https://github.com/rossedman/gh-fleet/tree/main/examples/upgrade-action)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/gha.star#L53)

## set_input

```python
set_input(repo, key, value, match = None)
```

Sets every `<key>: <value>` line in the workflows, such as an action input.

Typical keys are `go-version`, `node-version` and `python-version`.
Quotes around the old value are kept.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `key` | The key to set. |
| `value` | The new value. |
| `match` | A regular expression the current value must match to be replaced; for example `r"^\d"` leaves `stable` and matrix expressions alone. |

**Returns:** The list of paths that changed.

**Example**

```python
set_input(repo, "go-version", "1.27", match = r"^\d")
# returns [".github/workflows/ci.yml"]
```

=== "Before"

    `.github/workflows/ci.yml`

    ```yaml
    steps:
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - uses: actions/setup-go@v5
        with:
          go-version: stable
    ```

=== "After"

    `.github/workflows/ci.yml`

    ```yaml
    steps:
      - uses: actions/setup-go@v5
        with:
          go-version: '1.27'
      - uses: actions/setup-go@v5
        with:
          go-version: stable
    ```

**Used in:** [`bump-go-version`](https://github.com/rossedman/gh-fleet/tree/main/examples/bump-go-version), [`node-version`](https://github.com/rossedman/gh-fleet/tree/main/examples/node-version), [`python-version`](https://github.com/rossedman/gh-fleet/tree/main/examples/python-version)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/gha.star#L104)

## ensure_permissions

```python
ensure_permissions(repo, permissions = {"contents": "read"})
```

Adds a top-level `permissions` block to each workflow that has none.

The block goes right above the top-level `jobs:` key. Workflows that
already set top-level permissions, even broader ones, and files without a
`jobs:` key are left alone.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `permissions` | The scopes and levels to write, in order. |

**Returns:** The list of paths that changed.

**Example**

```python
ensure_permissions(repo, {"contents": "read"})
# returns [".github/workflows/ci.yml"]
```

=== "Before"

    `.github/workflows/ci.yml`

    ```yaml
    on: push

    jobs:
      test:
        runs-on: ubuntu-latest
    ```

=== "After"

    `.github/workflows/ci.yml`

    ```yaml
    on: push

    permissions:
      contents: read

    jobs:
      test:
        runs-on: ubuntu-latest
    ```

**Used in:** [`workflow-permissions`](https://github.com/rossedman/gh-fleet/tree/main/examples/workflow-permissions)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/gha.star#L156)

## unpinned

```python
unpinned(repo)
```

Lists each remote action used at a tag or branch rather than a commit SHA.

Sub-paths are dropped (`github/codeql-action/init@v3` is listed as
`github/codeql-action@v3`), since the ref belongs to the repository.
Local actions (`./path`) and `docker://` images are not listed. Hand the
list to something that can resolve refs, such as `gh api` in a `run`
step, and the results to `pin`.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The sorted, de-duplicated `owner/repo@ref` strings.

**Example**

```python
unpinned(repo)
# returns ["actions/checkout@v4", "github/codeql-action@v3"]
```

=== "Before"

    `.github/workflows/ci.yml`

    ```yaml
    steps:
      - uses: actions/checkout@v4
      - uses: github/codeql-action/init@v3
      - uses: ./.github/actions/setup
      - uses: actions/setup-go@0aaccfd150d50ccaeb58ebd88d36e91967a5f35b # v5
    ```

**Used in:** [`pin-actions`](https://github.com/rossedman/gh-fleet/tree/main/examples/pin-actions)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/gha.star#L206)

## pin

```python
pin(repo, shas)
```

Pins actions to commit SHAs, keeping the old ref as a comment.

`uses: owner/repo@ref` becomes `uses: owner/repo@<sha> # ref`, so
Dependabot can still update the pin. Refs missing from `shas` are left as
they are.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `shas` | Maps `owner/repo@ref`, as `unpinned` lists them, to a full commit SHA. A dict works, and so does a string of space-separated `owner/repo@ref=sha` pairs, which is easy to hand over from a shell step. A value that is not a SHA fails the run. |

**Returns:** The list of paths that changed.

**Example**

```python
pin(repo, {"actions/checkout@v4": "08eba0b27e820071cde6df949e0beb9ba4906955"})
# returns [".github/workflows/ci.yml"]
```

=== "Before"

    `.github/workflows/ci.yml`

    ```yaml
    steps:
      - uses: actions/checkout@v4
    ```

=== "After"

    `.github/workflows/ci.yml`

    ```yaml
    steps:
      - uses: actions/checkout@08eba0b27e820071cde6df949e0beb9ba4906955 # v4
    ```

**Used in:** [`pin-actions`](https://github.com/rossedman/gh-fleet/tree/main/examples/pin-actions)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/gha.star#L237)

## rename_branch

```python
rename_branch(repo, old, new)
```

Renames a branch in workflow triggers and branch checks, as when `master` becomes `main`.

Changes the `branches` and `branches-ignore` lists of triggers, in flow
(`[master, dev]`) and block form, and `refs/heads/<old>` anywhere in a
workflow, such as in `if:` conditions. Glob patterns such as `master-*`
and branches that only start with the old name are left alone.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `old` | The old branch name. |
| `new` | The new branch name. |

**Returns:** The list of paths that changed.

**Example**

```python
rename_branch(repo, "master", "main")
# returns [".github/workflows/ci.yml"]
```

=== "Before"

    `.github/workflows/ci.yml`

    ```yaml
    on:
      push:
        branches: [master, "release/*"]
      pull_request:
        branches:
          - master  # default branch
          - master-legacy
    jobs:
      deploy:
        if: github.ref == 'refs/heads/master'
    ```

=== "After"

    `.github/workflows/ci.yml`

    ```yaml
    on:
      push:
        branches: [main, "release/*"]
      pull_request:
        branches:
          - main  # default branch
          - master-legacy
    jobs:
      deploy:
        if: github.ref == 'refs/heads/main'
    ```

**Used in:** [`branch-rename`](https://github.com/rossedman/gh-fleet/tree/main/examples/branch-rename)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/gha.star#L291)
