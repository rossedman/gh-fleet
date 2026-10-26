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
| [`ansible.star`](starlark/ansible.md) | Bump Ansible role and collection versions in requirements files. | 2 |
| [`argocd.star`](starlark/argocd.md) | Edit Argo CD Application manifests: move an application to a new revision or chart version. | 2 |
| [`buildkite.star`](starlark/buildkite.md) | Edit Buildkite pipelines: plugin versions, agent queues and step defaults. | 6 |
| [`cargo.star`](starlark/cargo.md) | Bump Rust crate dependencies and rust-version in Cargo.toml files. | 4 |
| [`changelog.star`](starlark/changelog.md) | Add entries to a Keep a Changelog style CHANGELOG.md. | 2 |
| [`circleci.star`](starlark/circleci.md) | Edit CircleCI configuration: orb versions, executor images, machine images and resource classes. | 6 |
| [`cmake.star`](starlark/cmake.md) | Bump CMake's minimum version and FetchContent dependency tags. | 4 |
| [`codeowners.star`](starlark/codeowners.md) | Find and edit CODEOWNERS files. | 3 |
| [`dependabot.star`](starlark/dependabot.md) | Detect package ecosystems and write a Dependabot configuration. | 4 |
| [`docker.star`](starlark/docker.md) | Find Dockerfiles and move images to a new tag in Dockerfiles, Compose files and Kubernetes manifests. | 3 |
| [`dotnet.star`](starlark/dotnet.md) | .NET: bump NuGet packages, target frameworks and the SDK pinned in global.json. | 5 |
| [`editorconfig.star`](starlark/editorconfig.md) | Read and set properties in .editorconfig, line by line. | 2 |
| [`files.star`](starlark/files.md) | Keep lines in a file, replace text across the repo, and copy files from the campaign directory. | 5 |
| [`flux.star`](starlark/flux.md) | Edit Flux HelmReleases and sources: chart versions and Git refs. | 3 |
| [`gha.star`](starlark/gha.md) | Find GitHub Actions workflows, bump action versions, set inputs and add permissions. | 8 |
| [`gitattributes.star`](starlark/gitattributes.md) | Set attributes for path patterns in .gitattributes, such as line endings and Git LFS tracking. | 2 |
| [`gitlab.star`](starlark/gitlab.md) | Edit GitLab CI configuration: job and service images, included template refs, CI/CD component versions and defaults. | 6 |
| [`gomod.star`](starlark/gomod.md) | Read and raise the go directive in go.mod files. | 5 |
| [`gradle.star`](starlark/gradle.md) | Bump Gradle dependencies, version catalog entries and the Gradle wrapper. | 4 |
| [`helm.star`](starlark/helm.md) | Bump Helm chart versions, chart dependencies and image tags in values files. | 5 |
| [`jenkins.star`](starlark/jenkins.md) | Edit Jenkinsfiles: shared library versions, agent images and tool versions. | 5 |
| [`jsonfile.star`](starlark/jsonfile.md) | Read and edit any JSON file, keeping key order and indentation. | 5 |
| [`k8s.star`](starlark/k8s.md) | Edit Kubernetes manifests: migrate removed API versions and set labels on every resource. | 4 |
| [`kustomize.star`](starlark/kustomize.md) | Set image tags and names in kustomization files, the usual way to roll out a release with Kustomize. | 2 |
| [`license.star`](starlark/license.md) | Detect a repo's license, update its copyright year and set the license field in package manifests. | 4 |
| [`make.star`](starlark/make.md) | Read and set variables in Makefiles, such as `GO_VERSION ?= 1.22`. | 3 |
| [`markdown.star`](starlark/markdown.md) | Add or replace sections and badges in Markdown files such as README.md. | 3 |
| [`maven.star`](starlark/maven.md) | Bump dependency, plugin and property versions in Maven pom.xml files. | 3 |
| [`npm.star`](starlark/npm.md) | Find package.json files and edit them as data, keeping key order and indentation. | 4 |
| [`precommit.star`](starlark/precommit.md) | Read and bump hook revisions in .pre-commit-config.yaml. | 2 |
| [`python.star`](starlark/python.md) | Bump pinned Python requirements and the supported Python version. | 4 |
| [`refs.star`](starlark/refs.md) | Find, bump and pin versioned references in any file: `name@ref`, `name#ref` and `source?ref=tag`. | 3 |
| [`runtime.star`](starlark/runtime.md) | Set language versions in version files: .nvmrc, .python-version, .tool-versions and friends. | 2 |
| [`terraform.star`](starlark/terraform.md) | Bump Terraform's required_version and provider version constraints. | 3 |
| [`terragrunt.star`](starlark/terragrunt.md) | Move Terragrunt and Terraform module sources to a new `?ref=` tag. | 2 |
| [`toml.star`](starlark/toml.md) | Read, set and delete keys in TOML files, such as pyproject.toml and Cargo.toml. | 3 |
| [`yamlpath.star`](starlark/yamlpath.md) | Read, set and delete keys in any YAML file by path, keeping comments and layout. | 7 |

## All functions

| Function | What it does |
| --- | --- |
| [`gha.action_files`](starlark/gha.md#action_files) | Lists the workflows and every local action's metadata file. |
| [`changelog.add_entry`](starlark/changelog.md#add_entry) | Adds a bullet under the Unreleased release, in a section such as Changed or Security. |
| [`argocd.applications`](starlark/argocd.md#applications) | Lists the YAML files that define Argo CD Applications or ApplicationSets. |
| [`gradle.build_files`](starlark/gradle.md#build_files) | Lists the build scripts and version catalogs. |
| [`refs.bump`](starlark/refs.md#bump) | Moves every reference to `name` to a new ref. |
| [`gha.bump_action`](starlark/gha.md#bump_action) | Moves an action to a new version wherever a workflow or local action uses it. |
| [`docker.bump_base_image`](starlark/docker.md#bump_base_image) | Moves `FROM` lines of an image from one tag to another in every Dockerfile. |
| [`gitlab.bump_component`](starlark/gitlab.md#bump_component) | Moves an included CI/CD component to a new version. |
| [`circleci.bump_image`](starlark/circleci.md#bump_image) | Moves a Docker executor image from one tag to another. |
| [`docker.bump_image`](starlark/docker.md#bump_image) | Moves `image:` lines from one tag to another in YAML files. |
| [`gitlab.bump_image`](starlark/gitlab.md#bump_image) | Moves a job or service image from one tag to another. |
| [`jenkins.bump_image`](starlark/jenkins.md#bump_image) | Moves a container image used by an agent or step from one tag to another. |
| [`jenkins.bump_library`](starlark/jenkins.md#bump_library) | Moves a shared library to a new version in every Jenkinsfile. |
| [`circleci.bump_orb`](starlark/circleci.md#bump_orb) | Moves an orb to a new version in every config. |
| [`buildkite.bump_plugin`](starlark/buildkite.md#bump_plugin) | Moves a plugin to a new version in every pipeline. |
| [`helm.charts`](starlark/helm.md#charts) | Lists the repo's Helm charts. |
| [`gitlab.components`](starlark/gitlab.md#components) | Lists the CI/CD components a config includes, with their versions. |
| [`dependabot.config`](starlark/dependabot.md#config) | Builds the text of a dependabot.yml with one update entry per ecosystem. |
| [`circleci.configs`](starlark/circleci.md#configs) | Lists the repo's CircleCI configs. |
| [`gitlab.configs`](starlark/gitlab.md#configs) | Lists the repo's GitLab CI files. |
| [`jsonfile.delete`](starlark/jsonfile.md#delete) | Removes a key by dotted key path. |
| [`toml.delete`](starlark/toml.md#delete) | Removes a key from a table. |
| [`yamlpath.delete`](starlark/yamlpath.md#delete) | Removes a key and everything nested under it. |
| [`license.detect`](starlark/license.md#detect) | Identifies the license from the license file's text. |
| [`docker.dockerfiles`](starlark/docker.md#dockerfiles) | Lists the repo's Dockerfiles. |
| [`dependabot.ecosystems`](starlark/dependabot.md#ecosystems) | Detects the package ecosystems a repo uses, from files at its root. |
| [`markdown.ensure_badge`](starlark/markdown.md#ensure_badge) | Adds a badge under the first heading of a Markdown file. |
| [`files.ensure_lines`](starlark/files.md#ensure_lines) | Appends each of `lines` that a file does not already contain. |
| [`gha.ensure_permissions`](starlark/gha.md#ensure_permissions) | Adds a top-level `permissions` block to each workflow that has none. |
| [`cmake.files`](starlark/cmake.md#files) | Lists the repo's CMake files. |
| [`kustomize.files`](starlark/kustomize.md#files) | Lists the repo's kustomization files. |
| [`terraform.files`](starlark/terraform.md#files) | Lists the repo's Terraform files. |
| [`changelog.find`](starlark/changelog.md#find) | Finds the repo's changelog. |
| [`codeowners.find`](starlark/codeowners.md#find) | Finds the CODEOWNERS file GitHub uses. |
| [`dependabot.find`](starlark/dependabot.md#find) | Finds the repo's Dependabot configuration. |
| [`license.find`](starlark/license.md#find) | Finds the repo's license file. |
| [`refs.find`](starlark/refs.md#find) | Lists every reference to `name` in the files matching `files`. |
| [`editorconfig.get`](starlark/editorconfig.md#get) | Reads a property. |
| [`jsonfile.get`](starlark/jsonfile.md#get) | Reads a value by dotted key path. |
| [`toml.get`](starlark/toml.md#get) | Reads a key's value. |
| [`yamlpath.get`](starlark/yamlpath.md#get) | Reads a scalar value by key path. |
| [`yamlpath.get_in`](starlark/yamlpath.md#get_in) | Reads a scalar value by key path from YAML text rather than a file. |
| [`make.get_var`](starlark/make.md#get_var) | Reads the value of the first assignment to a variable. |
| [`gomod.go_version`](starlark/gomod.md#go_version) | Reads the version in a go.mod's `go` directive. |
| [`markdown.has_section`](starlark/markdown.md#has_section) | Reports whether a Markdown file has a heading with this text. |
| [`flux.helm_releases`](starlark/flux.md#helm_releases) | Lists the YAML files that define Flux HelmReleases. |
| [`jenkins.jenkinsfiles`](starlark/jenkins.md#jenkinsfiles) | Lists the repo's Jenkinsfiles. |
| [`jenkins.libraries`](starlark/jenkins.md#libraries) | Lists the shared libraries Jenkinsfiles load, with their versions. |
| [`make.makefiles`](starlark/make.md#makefiles) | Lists the repo's Makefiles. |
| [`cargo.manifests`](starlark/cargo.md#manifests) | Lists the repo's Cargo manifests. |
| [`k8s.manifests`](starlark/k8s.md#manifests) | Lists the YAML files that hold Kubernetes resources. |
| [`k8s.migrate_api_versions`](starlark/k8s.md#migrate_api_versions) | Moves resources off API versions Kubernetes removed. |
| [`cmake.minimum_version`](starlark/cmake.md#minimum_version) | Reads the version from `cmake_minimum_required`. |
| [`gomod.modules`](starlark/gomod.md#modules) | Lists the repo's Go modules. |
| [`circleci.orbs`](starlark/circleci.md#orbs) | Lists the orbs configs use, with their versions. |
| [`gomod.outdated`](starlark/gomod.md#outdated) | Finds the modules that require a dependency below a version. |
| [`codeowners.owners`](starlark/codeowners.md#owners) | Reads the owners of a CODEOWNERS pattern. |
| [`npm.packages`](starlark/npm.md#packages) | Lists the repo's package.json files. |
| [`gha.pin`](starlark/gha.md#pin) | Pins actions to commit SHAs, keeping the old ref as a comment. |
| [`refs.pin`](starlark/refs.md#pin) | Pins references to commit SHAs, keeping the old ref in a comment. |
| [`buildkite.pipelines`](starlark/buildkite.md#pipelines) | Lists the repo's pipeline files. |
| [`buildkite.plugins`](starlark/buildkite.md#plugins) | Lists every plugin a pipeline uses, with its version. |
| [`maven.poms`](starlark/maven.md#poms) | Lists the repo's pom.xml files. |
| [`dotnet.projects`](starlark/dotnet.md#projects) | Lists the project files and shared MSBuild props files. |
| [`buildkite.queues`](starlark/buildkite.md#queues) | Lists the agent queues pipelines target. |
| [`jsonfile.read`](starlark/jsonfile.md#read) | Reads a JSON file into a value whose dict keys keep the file's order. |
| [`npm.read_package`](starlark/npm.md#read_package) | Reads a package.json as a dict whose keys keep the file's order. |
| [`files.remove_lines`](starlark/files.md#remove_lines) | Removes every line equal to one of `lines`, after trimming whitespace. |
| [`gha.rename_branch`](starlark/gha.md#rename_branch) | Renames a branch in workflow triggers and branch checks, as when `master` becomes `main`. |
| [`files.replace_all`](starlark/files.md#replace_all) | Replaces every occurrence of a string with another, literally, in text files. |
| [`ansible.requirements`](starlark/ansible.md#requirements) | Lists the repo's Ansible Galaxy requirements files. |
| [`python.requirements_files`](starlark/python.md#requirements_files) | Lists the repo's requirements files. |
| [`gomod.requires`](starlark/gomod.md#requires) | Reads every module a go.mod requires, in single-line and block form. |
| [`python.requires_python`](starlark/python.md#requires_python) | Reads the `requires-python` of a pyproject.toml. |
| [`k8s.resources`](starlark/k8s.md#resources) | Lists every resource in the repo's manifests. |
| [`precommit.revs`](starlark/precommit.md#revs) | Reads the revision each hook repository is pinned to. |
| [`cargo.rust_version`](starlark/cargo.md#rust_version) | Reads the `rust-version` of a Cargo.toml. |
| [`dotnet.sdk_version`](starlark/dotnet.md#sdk_version) | Reads the .NET SDK version pinned in global.json. |
| [`editorconfig.set`](starlark/editorconfig.md#set) | Sets a property, adding it, or its section, when missing. |
| [`gitattributes.set`](starlark/gitattributes.md#set) | Gives a pattern attributes, merging with any it already has. |
| [`jsonfile.set`](starlark/jsonfile.md#set) | Sets a value by dotted key path, adding missing objects on the way. |
| [`toml.set`](starlark/toml.md#set) | Sets a key, adding it, and its table, when missing. |
| [`yamlpath.set`](starlark/yamlpath.md#set) | Sets a key to a value, adding the key and any missing parents if needed. |
| [`helm.set_app_version`](starlark/helm.md#set_app_version) | Sets a chart's `appVersion` in its Chart.yaml, keeping its quotes. |
| [`flux.set_chart_version`](starlark/flux.md#set_chart_version) | Sets the chart `version` of every HelmRelease that installs a chart. |
| [`helm.set_chart_version`](starlark/helm.md#set_chart_version) | Sets a chart's `version` in its Chart.yaml. |
| [`gitlab.set_default`](starlark/gitlab.md#set_default) | Sets a key under `default:` in .gitlab-ci.yml, adding `default:` if needed. |
| [`cargo.set_dependency`](starlark/cargo.md#set_dependency) | Sets a crate's version wherever a Cargo.toml already depends on it. |
| [`gradle.set_dependency`](starlark/gradle.md#set_dependency) | Sets a library's version wherever the build declares it. |
| [`helm.set_dependency`](starlark/helm.md#set_dependency) | Sets the version of a chart dependency in every Chart.yaml that lists it. |
| [`maven.set_dependency`](starlark/maven.md#set_dependency) | Sets the version of a dependency or plugin wherever a pom.xml declares one. |
| [`npm.set_dependency`](starlark/npm.md#set_dependency) | Sets a dependency's version range in every package.json that already lists it. |
| [`cmake.set_fetchcontent_tag`](starlark/cmake.md#set_fetchcontent_tag) | Sets the `GIT_TAG` of a `FetchContent_Declare` dependency. |
| [`flux.set_git_tag`](starlark/flux.md#set_git_tag) | Sets `ref.tag` of every GitRepository that points at a URL. |
| [`gomod.set_go_version`](starlark/gomod.md#set_go_version) | Raises the `go` directive in every go.mod, never lowering it. |
| [`kustomize.set_image`](starlark/kustomize.md#set_image) | Sets `newTag`, `newName` or `digest` for an image in every kustomization's `images` list. |
| [`helm.set_image_tag`](starlark/helm.md#set_image_tag) | Sets the `tag` next to `repository: <repository>` in values files. |
| [`yamlpath.set_in`](starlark/yamlpath.md#set_in) | Sets a key in YAML text rather than a file, the same way `set` does. |
| [`gitlab.set_include_ref`](starlark/gitlab.md#set_include_ref) | Sets the `ref` of every `include` that pulls templates from another project. |
| [`gha.set_input`](starlark/gha.md#set_input) | Sets every `<key>: <value>` line in the workflows, such as an action input. |
| [`k8s.set_label`](starlark/k8s.md#set_label) | Sets a label on every resource, and on the pods workloads create. |
| [`circleci.set_machine_image`](starlark/circleci.md#set_machine_image) | Sets the `image` of every machine executor, such as when an Ubuntu image is retired. |
| [`cmake.set_minimum_version`](starlark/cmake.md#set_minimum_version) | Raises `cmake_minimum_required` to a version, never lowering it. |
| [`codeowners.set_owners`](starlark/codeowners.md#set_owners) | Makes a list of users or teams the owners of a pattern. |
| [`dotnet.set_package`](starlark/dotnet.md#set_package) | Sets a NuGet package's version wherever a project already references it. |
| [`maven.set_property`](starlark/maven.md#set_property) | Sets a `<properties>` entry, such as `java.version`, in every pom.xml that defines it. |
| [`terraform.set_provider_version`](starlark/terraform.md#set_provider_version) | Sets the version constraint of a provider in `required_providers`. |
| [`buildkite.set_queue`](starlark/buildkite.md#set_queue) | Points `agents: queue:` at a new queue in every pipeline. |
| [`terraform.set_required_version`](starlark/terraform.md#set_required_version) | Sets every `required_version` constraint. |
| [`python.set_requirement`](starlark/python.md#set_requirement) | Sets a package's version specifier wherever it is already required. |
| [`python.set_requires_python`](starlark/python.md#set_requires_python) | Sets `requires-python` in a pyproject.toml that already has it. |
| [`circleci.set_resource_class`](starlark/circleci.md#set_resource_class) | Sets `resource_class` on every job and executor that sets one. |
| [`precommit.set_rev`](starlark/precommit.md#set_rev) | Pins a hook repository to a new revision, keeping quotes and comments. |
| [`cargo.set_rust_version`](starlark/cargo.md#set_rust_version) | Raises `rust-version` in every Cargo.toml that sets it, never lowering it. |
| [`dotnet.set_sdk_version`](starlark/dotnet.md#set_sdk_version) | Sets the .NET SDK version in a global.json that already pins one. |
| [`markdown.set_section`](starlark/markdown.md#set_section) | Makes the section under a heading contain the given text. |
| [`yamlpath.set_sibling`](starlark/yamlpath.md#set_sibling) | Sets `key` in every mapping where `match_key` has the value `match_value`, list items included. |
| [`terragrunt.set_source_ref`](starlark/terragrunt.md#set_source_ref) | Sets the `?ref=` of every module source that matches. |
| [`license.set_spdx`](starlark/license.md#set_spdx) | Sets the license field in package manifests that already have one. |
| [`buildkite.set_step_defaults`](starlark/buildkite.md#set_step_defaults) | Adds keys, such as `timeout_in_minutes`, to every command step that lacks them. |
| [`dotnet.set_target_framework`](starlark/dotnet.md#set_target_framework) | Sets the target framework of every project, such as `net8.0`. |
| [`argocd.set_target_revision`](starlark/argocd.md#set_target_revision) | Sets `targetRevision` for every source that points at a repository. |
| [`jenkins.set_tool`](starlark/jenkins.md#set_tool) | Sets the installation a `tools` block asks for, such as `maven 'Maven 3.9'`. |
| [`make.set_var`](starlark/make.md#set_var) | Sets every assignment to a variable, in every Makefile that has one. |
| [`ansible.set_version`](starlark/ansible.md#set_version) | Sets the version of a role or collection wherever a requirements file lists it. |
| [`runtime.set_version`](starlark/runtime.md#set_version) | Sets a runtime's version in every version file the repo already has. |
| [`gradle.set_wrapper_version`](starlark/gradle.md#set_wrapper_version) | Points the Gradle wrapper at a new version, never lowering it. |
| [`license.set_year`](starlark/license.md#set_year) | Extends the copyright year in the license file to `year`. |
| [`terragrunt.sources`](starlark/terragrunt.md#sources) | Lists every module source pinned with `?ref=`. |
| [`files.sync`](starlark/files.md#sync) | Copies every file under a directory next to the campaign file into the repo. |
| [`gitattributes.track_lfs`](starlark/gitattributes.md#track_lfs) | Tracks a pattern with Git LFS, as `git lfs track` would. |
| [`gha.unpinned`](starlark/gha.md#unpinned) | Lists each remote action used at a tag or branch rather than a commit SHA. |
| [`yamlpath.values`](starlark/yamlpath.md#values) | Reads every scalar whose key path matches a pattern with `*` wildcards. |
| [`runtime.version`](starlark/runtime.md#version) | Reads the version a repo pins for a runtime. |
| [`gha.workflows`](starlark/gha.md#workflows) | Lists the repo's workflow files. |
| [`gradle.wrapper_version`](starlark/gradle.md#wrapper_version) | Reads the Gradle version the wrapper downloads. |
| [`jsonfile.write`](starlark/jsonfile.md#write) | Writes a value as JSON, using the indentation the file already has. |
| [`dependabot.write_if_missing`](starlark/dependabot.md#write_if_missing) | Writes .github/dependabot.yml for the detected ecosystems unless a config exists. |
| [`files.write_if_missing`](starlark/files.md#write_if_missing) | Writes a file unless it, or any of the alternative paths, already exists. |
| [`npm.write_package`](starlark/npm.md#write_package) | Writes a dict as a package.json, keeping the file's indentation. |
