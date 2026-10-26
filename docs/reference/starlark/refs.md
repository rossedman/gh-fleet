---
title: refs.star
---

# refs.star

Find, bump and pin versioned references in any file: `name@ref`, `name#ref` and `source?ref=tag`.

The same shape shows up across tools: GitHub Actions and CircleCI orbs
(`actions/checkout@v4`, `circleci/node@5.1.0`), Jenkins shared libraries
(`@Library('ci-lib@v2')`), Buildkite plugins (`docker-compose#v4.14.0`) and
Terraform module sources (`git::https://github.com/acme/modules.git//vpc?ref=v1.2.0`).
These helpers work on any of them, line by line, so comments and layout
are kept. The tool-specific modules, such as `buildkite` and `terragrunt`,
build on them.

A name may use `*` to match any run of characters other than spaces,
quotes, brackets and separators, as in `acme/*` or
`git::https://github.com/acme/modules.git//*`.

```python
load("@fleet//refs.star", "find", "bump", "pin")
```

| Function | What it does |
| --- | --- |
| [`find`](#find) | Lists every reference to `name` in the files matching `files`. |
| [`bump`](#bump) | Moves every reference to `name` to a new ref. |
| [`pin`](#pin) | Pins references to commit SHAs, keeping the old ref in a comment. |

## find

```python
find(repo, files, name, sep = "@", exclude = [])
```

Lists every reference to `name` in the files matching `files`.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `files` | Glob patterns of the files to search, such as `[".buildkite/**/*.yml"]`. |
| `name` | The name before the separator; `*` matches any run of name characters. |
| `sep` | The separator between name and ref: `@`, `#` or `?ref=`. |
| `exclude` | Glob patterns of files to skip. |

**Returns:** A list of dicts with `path`, `line` (counting from 1), `name` and
`ref`, in file and line order.

**Example**

```python
[r["name"] + " " + r["ref"] for r in find(repo, [".circleci/*.yml"], "circleci/*")]
# returns ["circleci/node 5.1.0", "circleci/aws-cli 4.0"]
```

=== "Before"

    `.circleci/config.yml`

    ```yaml
    orbs:
      node: circleci/node@5.1.0
      aws-cli: circleci/aws-cli@4.0
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/refs.star#L34)

## bump

```python
bump(repo, files, name, to, sep = "@", from_ref = None, exclude = [])
```

Moves every reference to `name` to a new ref.

References pinned to a full commit SHA are left alone unless
`from_ref` names that SHA. With `from_ref`, only that exact ref is
replaced, so `v3` never matches `v3.5.0`.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `files` | Glob patterns of the files to edit. |
| `name` | The name before the separator; `*` matches any run of name characters. |
| `to` | The new ref. |
| `sep` | The separator between name and ref: `@`, `#` or `?ref=`. |
| `from_ref` | When set, only this exact ref is replaced. |
| `exclude` | Glob patterns of files to skip. |

**Returns:** The list of paths that changed.

**Example**

```python
bump(repo, ["Jenkinsfile", "**/Jenkinsfile"], "ci-lib", "v3.0")
# returns ["Jenkinsfile"]
```

=== "Before"

    `Jenkinsfile`

    ```text
    @Library('ci-lib@v2.3') _
    pipeline { agent any }
    ```

=== "After"

    `Jenkinsfile`

    ```text
    @Library('ci-lib@v3.0') _
    pipeline { agent any }
    ```

**Used in:** [`jenkins-shared-library`](https://github.com/rossedman/gh-fleet/tree/main/examples/jenkins-shared-library)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/refs.star#L66)

## pin

```python
pin(repo, files, name, shas, sep = "@", comment = "#", exclude = [])
```

Pins references to commit SHAs, keeping the old ref in a comment.

`name@v4` becomes `name@<sha>`, and ` # v4` is added at the end of the
line so update tools such as Dependabot still know the version. Refs
missing from `shas` are left as they are.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `files` | Glob patterns of the files to edit. |
| `name` | The name before the separator; `*` matches any run of name characters. |
| `shas` | Maps `name<sep>ref` to a full commit SHA; a dict, or a string of space-separated `name<sep>ref=sha` pairs. |
| `sep` | The separator between name and ref. |
| `comment` | The comment marker of the file type, such as `#` or `//`; None to add no comment. |
| `exclude` | Glob patterns of files to skip. |

**Returns:** The list of paths that changed.

**Example**

```python
pin(repo, [".buildkite/*.yml"], "docker-compose", {"docker-compose#v4.14.0": "1111111111111111111111111111111111111111"}, sep = "#")
# returns [".buildkite/pipeline.yml"]
```

=== "Before"

    `.buildkite/pipeline.yml`

    ```yaml
    steps:
      - plugins:
          - docker-compose#v4.14.0:
              run: app
    ```

=== "After"

    `.buildkite/pipeline.yml`

    ```yaml
    steps:
      - plugins:
          - docker-compose#1111111111111111111111111111111111111111: # v4.14.0
              run: app
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/refs.star#L114)
