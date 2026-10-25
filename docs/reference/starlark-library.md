---
title: Starlark library reference
---

# Starlark library reference

The helpers built into gh fleet, generated from their docstrings. Load them by module, as in `load("@fleet//files.star", "ensure_lines")`; they need no network and are versioned with the binary. See [Starlark changes](../guides/starlark.md) for how scripts run.

| Module | What it does |
| --- | --- |
| [`@fleet//codeowners.star`](#codeownersstar) | Find and edit CODEOWNERS files. |
| [`@fleet//dependabot.star`](#dependabotstar) | Detect package ecosystems and write a Dependabot configuration. |
| [`@fleet//docker.star`](#dockerstar) | Find Dockerfiles and move base images to a new tag. |
| [`@fleet//files.star`](#filesstar) | Keep lines in a file, replace text across the repo, and copy files from the campaign directory. |
| [`@fleet//gha.star`](#ghastar) | Find GitHub Actions workflows, bump action versions, set inputs and add permissions. |
| [`@fleet//gomod.star`](#gomodstar) | Read and raise the go directive in go.mod files. |
| [`@fleet//npm.star`](#npmstar) | Find package.json files and edit them as data, keeping key order and indentation. |

## codeowners.star

Find and edit CODEOWNERS files.

```python
load("@fleet//codeowners.star", "find", "owners", "set_owners")
```

### codeowners.find

```python
find(repo)
```

Returns the path of the CODEOWNERS file GitHub uses, or None.

GitHub reads the first of .github/CODEOWNERS, CODEOWNERS and
docs/CODEOWNERS that exists.

### codeowners.owners

```python
owners(repo, pattern = "*")
```

Returns the owners of the last rule for exactly `pattern`, or an empty list.

### codeowners.set_owners

```python
set_owners(repo, pattern, owners, header = None)
```

Makes `owners` the owners of `pattern`, editing the existing rule or adding one.

`owners` is a list or a space-separated string of users or teams; a
leading `@` is added when missing. With no CODEOWNERS file, one is
created at .github/CODEOWNERS, starting with `header` when given.

Returns True when the file changed.

## dependabot.star

Detect package ecosystems and write a Dependabot configuration.

```python
load("@fleet//dependabot.star", "find", "ecosystems", "config", "write_if_missing")
```

### dependabot.find

```python
find(repo)
```

Returns the path of the repo's Dependabot config, or None.

### dependabot.ecosystems

```python
ecosystems(repo)
```

Returns the Dependabot ecosystems the repo uses, judged by files at its root.

### dependabot.config

```python
config(ecosystems, interval = "weekly", group = True, prefixes = {})
```

Returns the text of a dependabot.yml with one update entry per ecosystem.

Each entry checks `/` on `interval`. With `group`, minor and patch
updates come as one PR per ecosystem. Commit messages start with the
ecosystem's entry in `prefixes`, or `ci(deps)` for github-actions and
`build(deps)` for the rest.

### dependabot.write_if_missing

```python
write_if_missing(repo, interval = "weekly", group = True, prefixes = {})
```

Writes .github/dependabot.yml for the detected ecosystems unless a config exists.

Repos with a config, or with no ecosystem detected, are left alone.
Returns the ecosystems written, or an empty list.

## docker.star

Find Dockerfiles and move base images to a new tag.

```python
load("@fleet//docker.star", "dockerfiles", "bump_base_image")
```

### docker.dockerfiles

```python
dockerfiles(repo)
```

Returns the paths of every Dockerfile, *.Dockerfile and Dockerfile.* outside node_modules.

### docker.bump_base_image

```python
bump_base_image(repo, image, from_tag, to_tag)
```

Rewrites `FROM <image>:<from_tag>` to `<image>:<to_tag>` in every Dockerfile.

A variant suffix is kept, so with `from_tag = "18"` and `to_tag = "22"`,
`node:18-alpine` becomes `node:22-alpine`, while `node:18.2` is not
touched. `--platform` flags and stage names are kept, and images pinned
by digest only (`node@sha256:...`) never match.

Returns the paths that changed.

## files.star

Keep lines in a file, replace text across the repo, and copy files from the campaign directory.

Every helper is idempotent: run it on a repo that already has the change
and it writes nothing, so the repo comes out as a no-op.

```python
load("@fleet//files.star", "ensure_lines", "remove_lines", "replace_all", "write_if_missing", "sync")
```

### files.ensure_lines

```python
ensure_lines(repo, path, lines, header = None)
```

Appends each of `lines` that `path` does not already contain.

`lines` is a list or a newline-separated string. Lines are compared
after trimming whitespace, and blank entries are skipped. The new lines
go at the end of the file, after a blank line and `header` (for example
a comment) when one is given. The file is created if needed.

Returns the lines that were added.

### files.remove_lines

```python
remove_lines(repo, path, lines)
```

Removes every line of `path` equal to one of `lines` after trimming whitespace.

A missing file is left alone. Returns the lines that were removed.

### files.replace_all

```python
replace_all(repo, old, new, include = ["**"], exclude = [])
```

Replaces every occurrence of `old` with `new`, literally, in text files.

`include` and `exclude` are glob patterns (`**` matches any depth).
Binary files, which contain a NUL byte, are skipped.

Returns the paths that changed.

### files.write_if_missing

```python
write_if_missing(repo, path, content, also = [])
```

Writes `path` unless it, or any path in `also`, already exists.

Use `also` for files that can live in several places, such as
CODEOWNERS. Returns True when the file was written.

### files.sync

```python
sync(repo, src, dest = "", exclude = [])
```

Copies every file under `src` in the campaign directory into the repo.

Files land at the same relative path under `dest` (the repo root by
default), replacing what is there. Files already identical are not
touched, and files the repo has but `src` lacks are kept. New files are
written with mode 0644; an existing file keeps its mode.

Returns the repo paths that changed.

## gha.star

Find GitHub Actions workflows, bump action versions, set inputs and add permissions.

Edits are line-based rather than a YAML round trip, so comments, quoting
and formatting stay exactly as they were.

```python
load("@fleet//gha.star", "workflows", "action_files", "bump_action", "set_input", "ensure_permissions")
```

### gha.workflows

```python
workflows(repo)
```

Returns the paths of the repo's workflow files under .github/workflows.

### gha.action_files

```python
action_files(repo)
```

Returns the workflows plus every local action's action.yml or action.yaml.

### gha.bump_action

```python
bump_action(repo, action, to, from_ref = None)
```

Rewrites `uses: <action>@<ref>` to `uses: <action>@<to>` in workflows and local actions.

`action` is `owner/repo`; steps that use a sub-path (`owner/repo/path`)
match too. Refs pinned to a full commit SHA are left alone. With
`from_ref`, only that exact ref is replaced, so `v3` does not match
`v3.1`.

Returns the paths that changed.

### gha.set_input

```python
set_input(repo, key, value, match = None)
```

Sets every `<key>: <value>` line in workflows, such as `go-version` or `node-version`.

Quotes around the old value are kept. With `match`, a regular
expression, only current values it matches are replaced; for example
`match = r"^\d"` leaves `go-version: stable` alone.

Returns the paths that changed.

### gha.ensure_permissions

```python
ensure_permissions(repo, permissions = {"contents": "read"})
```

Adds a top-level `permissions` block to each workflow that has none.

The block goes right above the top-level `jobs:` key. Workflows that
already set top-level permissions, even broader ones, are left alone.

Returns the paths that changed.

## gomod.star

Read and raise the go directive in go.mod files.

```python
load("@fleet//gomod.star", "modules", "go_version", "set_go_version")
```

### gomod.modules

```python
modules(repo)
```

Returns the paths of every go.mod outside vendor and testdata directories.

### gomod.go_version

```python
go_version(repo, path = "go.mod")
```

Returns the version in a go.mod's `go` directive, or None when it has none or does not exist.

### gomod.set_go_version

```python
set_go_version(repo, version, paths = None)
```

Raises the `go` directive to `version` in every go.mod, never lowering it.

`paths` limits the change to some go.mod files; by default it is every
module from `modules(repo)`. A `toolchain` line older than or equal to
the new version is removed, since the go directive now covers it; a
newer one is kept.

Returns the paths that changed.

## npm.star

Find package.json files and edit them as data, keeping key order and indentation.

These helpers edit package.json only. To refresh package-lock.json, follow
the Starlark step with a `run` step such as
`npm install --package-lock-only --ignore-scripts`.

```python
load("@fleet//npm.star", "packages", "read_package", "write_package", "set_dependency")
```

### npm.packages

```python
packages(repo)
```

Returns the paths of every package.json outside node_modules.

### npm.read_package

```python
read_package(repo, path = "package.json")
```

Returns a package.json decoded into a dict whose keys keep the file's order.

### npm.write_package

```python
write_package(repo, path, pkg)
```

Writes `pkg` to `path` as JSON, using the indentation the file already has (two spaces for a new file).

### npm.set_dependency

```python
set_dependency(repo, name, version, sections = SECTIONS)
```

Sets `name` to `version` in each package.json that already lists it in one of `sections`.

Packages that do not depend on `name` are left alone, so this never
adds a dependency. Returns the paths that changed.
