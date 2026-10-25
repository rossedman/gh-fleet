---
title: Starlark library
---

# Starlark library

Helpers for the changes campaigns make most, built into gh fleet. They need no network and are versioned with the binary. Load them by module:

```python
load("@fleet//files.star", "ensure_lines")

def change(repo):
    ensure_lines(repo, ".gitignore", [".env"])
```

Every helper:

- takes `repo`, the value passed to `change(repo)`, as its first argument;
- is **idempotent**: run it on a repo that already has the change and it writes nothing, so the repo comes out as a no-op;
- edits files **line by line** where a parse and re-encode would lose comments or layout;
- **returns what it changed**, usually a list of paths, so the script can tell the PR about it with `repo.set_var`.

This reference is generated from the library's docstrings, and every example on these pages runs as a test, so it always matches the code. See [Starlark changes](../guides/starlark.md) for how scripts run.

## Modules

| Module | What it does | Functions |
| --- | --- | --- |
| [`cargo.star`](starlark/cargo.md) | Bump Rust crate dependencies and rust-version in Cargo.toml files. | 4 |
| [`codeowners.star`](starlark/codeowners.md) | Find and edit CODEOWNERS files. | 3 |
| [`dependabot.star`](starlark/dependabot.md) | Detect package ecosystems and write a Dependabot configuration. | 4 |
| [`docker.star`](starlark/docker.md) | Find Dockerfiles and move images to a new tag in Dockerfiles, Compose files and Kubernetes manifests. | 3 |
| [`files.star`](starlark/files.md) | Keep lines in a file, replace text across the repo, and copy files from the campaign directory. | 5 |
| [`gha.star`](starlark/gha.md) | Find GitHub Actions workflows, bump action versions, set inputs and add permissions. | 7 |
| [`gomod.star`](starlark/gomod.md) | Read and raise the go directive in go.mod files. | 5 |
| [`markdown.star`](starlark/markdown.md) | Add or replace sections and badges in Markdown files such as README.md. | 3 |
| [`npm.star`](starlark/npm.md) | Find package.json files and edit them as data, keeping key order and indentation. | 4 |
| [`precommit.star`](starlark/precommit.md) | Read and bump hook revisions in .pre-commit-config.yaml. | 2 |
| [`python.star`](starlark/python.md) | Bump pinned Python requirements and the supported Python version. | 4 |
| [`runtime.star`](starlark/runtime.md) | Set language versions in version files: .nvmrc, .python-version, .tool-versions and friends. | 2 |
| [`terraform.star`](starlark/terraform.md) | Bump Terraform's required_version and provider version constraints. | 3 |

## All functions

| Function | What it does |
| --- | --- |
| [`gha.action_files`](starlark/gha.md#action_files) | Lists the workflows and every local action's metadata file. |
| [`gha.bump_action`](starlark/gha.md#bump_action) | Moves an action to a new version wherever a workflow or local action uses it. |
| [`docker.bump_base_image`](starlark/docker.md#bump_base_image) | Moves `FROM` lines of an image from one tag to another in every Dockerfile. |
| [`docker.bump_image`](starlark/docker.md#bump_image) | Moves `image:` lines from one tag to another in YAML files. |
| [`dependabot.config`](starlark/dependabot.md#config) | Builds the text of a dependabot.yml with one update entry per ecosystem. |
| [`docker.dockerfiles`](starlark/docker.md#dockerfiles) | Lists the repo's Dockerfiles. |
| [`dependabot.ecosystems`](starlark/dependabot.md#ecosystems) | Detects the package ecosystems a repo uses, from files at its root. |
| [`markdown.ensure_badge`](starlark/markdown.md#ensure_badge) | Adds a badge under the first heading of a Markdown file. |
| [`files.ensure_lines`](starlark/files.md#ensure_lines) | Appends each of `lines` that a file does not already contain. |
| [`gha.ensure_permissions`](starlark/gha.md#ensure_permissions) | Adds a top-level `permissions` block to each workflow that has none. |
| [`terraform.files`](starlark/terraform.md#files) | Lists the repo's Terraform files. |
| [`codeowners.find`](starlark/codeowners.md#find) | Finds the CODEOWNERS file GitHub uses. |
| [`dependabot.find`](starlark/dependabot.md#find) | Finds the repo's Dependabot configuration. |
| [`gomod.go_version`](starlark/gomod.md#go_version) | Reads the version in a go.mod's `go` directive. |
| [`markdown.has_section`](starlark/markdown.md#has_section) | Reports whether a Markdown file has a heading with this text. |
| [`cargo.manifests`](starlark/cargo.md#manifests) | Lists the repo's Cargo manifests. |
| [`gomod.modules`](starlark/gomod.md#modules) | Lists the repo's Go modules. |
| [`gomod.outdated`](starlark/gomod.md#outdated) | Finds the modules that require a dependency below a version. |
| [`codeowners.owners`](starlark/codeowners.md#owners) | Reads the owners of a CODEOWNERS pattern. |
| [`npm.packages`](starlark/npm.md#packages) | Lists the repo's package.json files. |
| [`gha.pin`](starlark/gha.md#pin) | Pins actions to commit SHAs, keeping the old ref as a comment. |
| [`npm.read_package`](starlark/npm.md#read_package) | Reads a package.json as a dict whose keys keep the file's order. |
| [`files.remove_lines`](starlark/files.md#remove_lines) | Removes every line equal to one of `lines`, after trimming whitespace. |
| [`files.replace_all`](starlark/files.md#replace_all) | Replaces every occurrence of a string with another, literally, in text files. |
| [`python.requirements_files`](starlark/python.md#requirements_files) | Lists the repo's requirements files. |
| [`gomod.requires`](starlark/gomod.md#requires) | Reads every module a go.mod requires, in single-line and block form. |
| [`python.requires_python`](starlark/python.md#requires_python) | Reads the `requires-python` of a pyproject.toml. |
| [`precommit.revs`](starlark/precommit.md#revs) | Reads the revision each hook repository is pinned to. |
| [`cargo.rust_version`](starlark/cargo.md#rust_version) | Reads the `rust-version` of a Cargo.toml. |
| [`cargo.set_dependency`](starlark/cargo.md#set_dependency) | Sets a crate's version wherever a Cargo.toml already depends on it. |
| [`npm.set_dependency`](starlark/npm.md#set_dependency) | Sets a dependency's version range in every package.json that already lists it. |
| [`gomod.set_go_version`](starlark/gomod.md#set_go_version) | Raises the `go` directive in every go.mod, never lowering it. |
| [`gha.set_input`](starlark/gha.md#set_input) | Sets every `<key>: <value>` line in the workflows, such as an action input. |
| [`codeowners.set_owners`](starlark/codeowners.md#set_owners) | Makes a list of users or teams the owners of a pattern. |
| [`terraform.set_provider_version`](starlark/terraform.md#set_provider_version) | Sets the version constraint of a provider in `required_providers`. |
| [`terraform.set_required_version`](starlark/terraform.md#set_required_version) | Sets every `required_version` constraint. |
| [`python.set_requirement`](starlark/python.md#set_requirement) | Sets a package's version specifier wherever it is already required. |
| [`python.set_requires_python`](starlark/python.md#set_requires_python) | Sets `requires-python` in a pyproject.toml that already has it. |
| [`precommit.set_rev`](starlark/precommit.md#set_rev) | Pins a hook repository to a new revision, keeping quotes and comments. |
| [`cargo.set_rust_version`](starlark/cargo.md#set_rust_version) | Raises `rust-version` in every Cargo.toml that sets it, never lowering it. |
| [`markdown.set_section`](starlark/markdown.md#set_section) | Makes the section under a heading contain the given text. |
| [`runtime.set_version`](starlark/runtime.md#set_version) | Sets a runtime's version in every version file the repo already has. |
| [`files.sync`](starlark/files.md#sync) | Copies every file under a directory next to the campaign file into the repo. |
| [`gha.unpinned`](starlark/gha.md#unpinned) | Lists each remote action used at a tag or branch rather than a commit SHA. |
| [`runtime.version`](starlark/runtime.md#version) | Reads the version a repo pins for a runtime. |
| [`gha.workflows`](starlark/gha.md#workflows) | Lists the repo's workflow files. |
| [`dependabot.write_if_missing`](starlark/dependabot.md#write_if_missing) | Writes .github/dependabot.yml for the detected ecosystems unless a config exists. |
| [`files.write_if_missing`](starlark/files.md#write_if_missing) | Writes a file unless it, or any of the alternative paths, already exists. |
| [`npm.write_package`](starlark/npm.md#write_package) | Writes a dict as a package.json, keeping the file's indentation. |
