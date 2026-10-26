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
| [`android.star`](starlark/android.md) | Raise Android SDK levels and the Android Gradle Plugin version. | 3 |
| [`ansible.star`](starlark/ansible.md) | Bump Ansible role and collection versions in requirements files. | 2 |
| [`argocd.star`](starlark/argocd.md) | Edit Argo CD Application manifests: move an application to a new revision or chart version. | 2 |
| [`azure_pipelines.star`](starlark/azure_pipelines.md) | Edit Azure Pipelines YAML: hosted agent images, task versions and template refs. | 4 |
| [`backstage.star`](starlark/backstage.md) | Edit Backstage catalog-info.yaml files: owners, annotations, tags and other entity fields. | 7 |
| [`bazel.star`](starlark/bazel.md) | Bump Bazel itself, Bzlmod module dependencies and http_archive downloads. | 4 |
| [`bitbucket.star`](starlark/bitbucket.md) | Edit Bitbucket Pipelines: step images and pipe versions. | 3 |
| [`buf.star`](starlark/buf.md) | Bump Buf remote plugins, module dependencies and the buf.yaml format. | 2 |
| [`buildkite.star`](starlark/buildkite.md) | Edit Buildkite pipelines: plugin versions, agent queues and step defaults. | 6 |
| [`cargo.star`](starlark/cargo.md) | Bump Rust crate dependencies and rust-version in Cargo.toml files. | 4 |
| [`cdk.star`](starlark/cdk.md) | Set AWS CDK context values, such as feature flags, in cdk.json. | 2 |
| [`changelog.star`](starlark/changelog.md) | Add entries to a Keep a Changelog style CHANGELOG.md. | 2 |
| [`circleci.star`](starlark/circleci.md) | Edit CircleCI configuration: orb versions, executor images, machine images and resource classes. | 6 |
| [`cmake.star`](starlark/cmake.md) | Bump CMake's minimum version and FetchContent dependency tags. | 4 |
| [`cocoapods.star`](starlark/cocoapods.md) | Bump CocoaPods pod versions and the platform version in Podfiles. | 3 |
| [`codeowners.star`](starlark/codeowners.md) | Find and edit CODEOWNERS files. | 3 |
| [`codeql.star`](starlark/codeql.md) | Set up CodeQL code scanning: add a workflow, add languages and move the action. | 5 |
| [`dependabot.star`](starlark/dependabot.md) | Detect package ecosystems, write a Dependabot configuration, and edit existing ones. | 6 |
| [`devcontainer.star`](starlark/devcontainer.md) | Edit Dev Container configs: images, feature versions and top-level settings. | 7 |
| [`docker.star`](starlark/docker.md) | Find Dockerfiles and move images to a new tag in Dockerfiles, Compose files and Kubernetes manifests. | 3 |
| [`dotnet.star`](starlark/dotnet.md) | .NET: bump NuGet packages, target frameworks and the SDK pinned in global.json. | 5 |
| [`drone.star`](starlark/drone.md) | Move step and plugin images in Drone and Woodpecker CI pipelines. | 2 |
| [`editorconfig.star`](starlark/editorconfig.md) | Read and set properties in .editorconfig, line by line. | 2 |
| [`elixir.star`](starlark/elixir.md) | Bump Mix dependencies and the Elixir version requirement in mix.exs. | 3 |
| [`eslint.star`](starlark/eslint.md) | Set rules and presets in ESLint's JSON and YAML configs. | 3 |
| [`files.star`](starlark/files.md) | Keep lines in a file, replace text across the repo, and copy files from the campaign directory. | 5 |
| [`flux.star`](starlark/flux.md) | Edit Flux HelmReleases and sources: chart versions and Git refs. | 3 |
| [`gha.star`](starlark/gha.md) | Find GitHub Actions workflows, bump action versions, set inputs and add permissions. | 8 |
| [`gitattributes.star`](starlark/gitattributes.md) | Set attributes for path patterns in .gitattributes, such as line endings and Git LFS tracking. | 2 |
| [`github.star`](starlark/github.md) | Edit repository files under .github: issue template config, release note categories, labels and funding. | 6 |
| [`gitlab.star`](starlark/gitlab.md) | Edit GitLab CI configuration: job and service images, included template refs, CI/CD component versions and defaults. | 6 |
| [`golangci.star`](starlark/golangci.md) | Edit golangci-lint configs: enable and disable linters and set linter settings. | 5 |
| [`gomod.star`](starlark/gomod.md) | Read and raise the go directive in go.mod files. | 6 |
| [`gradle.star`](starlark/gradle.md) | Bump Gradle dependencies, version catalog entries and the Gradle wrapper. | 6 |
| [`helm.star`](starlark/helm.md) | Bump Helm chart versions, chart dependencies and image tags in values files. | 5 |
| [`helmfile.star`](starlark/helmfile.md) | Bump chart versions of Helmfile releases. | 2 |
| [`jenkins.star`](starlark/jenkins.md) | Edit Jenkinsfiles: shared library versions, agent images and tool versions. | 5 |
| [`jsonfile.star`](starlark/jsonfile.md) | Read and edit any JSON file, keeping key order and indentation. | 5 |
| [`k8s.star`](starlark/k8s.md) | Edit Kubernetes manifests: migrate removed API versions and set labels on every resource. | 4 |
| [`kustomize.star`](starlark/kustomize.md) | Set image tags and names in kustomization files, the usual way to roll out a release with Kustomize. | 2 |
| [`license.star`](starlark/license.md) | Detect a repo's license, update its copyright year and set the license field in package manifests. | 4 |
| [`lintconfig.star`](starlark/lintconfig.md) | Set rules in markdownlint, yamllint and hadolint configs. | 3 |
| [`make.star`](starlark/make.md) | Read and set variables in Makefiles, such as `GO_VERSION ?= 1.22`. | 3 |
| [`markdown.star`](starlark/markdown.md) | Add or replace sections and badges in Markdown files such as README.md. | 3 |
| [`maven.star`](starlark/maven.md) | Bump dependency, plugin and property versions in Maven pom.xml files. | 3 |
| [`npm.star`](starlark/npm.md) | Find package.json files and edit them as data, keeping key order and indentation. | 6 |
| [`packer.star`](starlark/packer.md) | Bump plugin versions in Packer templates. | 2 |
| [`php.star`](starlark/php.md) | Bump Composer package constraints and the PHP version in composer.json. | 3 |
| [`precommit.star`](starlark/precommit.md) | Read and bump hook revisions in .pre-commit-config.yaml. | 2 |
| [`prettier.star`](starlark/prettier.md) | Set Prettier options in whichever config a repo uses. | 3 |
| [`python.star`](starlark/python.md) | Bump pinned Python requirements and the supported Python version. | 4 |
| [`refs.star`](starlark/refs.md) | Find, bump and pin versioned references in any file: `name@ref`, `name#ref` and `source?ref=tag`. | 3 |
| [`renovate.star`](starlark/renovate.md) | Edit Renovate configs: presets, package rules, schedules and other settings. | 5 |
| [`ruby.star`](starlark/ruby.md) | Bump gem versions, the Ruby version and Bundler, and set RuboCop's target Ruby. | 6 |
| [`runtime.star`](starlark/runtime.md) | Set language versions in version files: .nvmrc, .python-version, .tool-versions and friends. | 2 |
| [`sbt.star`](starlark/sbt.md) | Bump sbt, Scala, library dependencies and sbt plugins. | 5 |
| [`scanignore.star`](starlark/scanignore.md) | Manage scanner ignore files: .trivyignore, .semgrepignore and .gitleaksignore. | 4 |
| [`scorecard.star`](starlark/scorecard.md) | Set up OpenSSF Scorecard: add the workflow, move the action and add the README badge. | 4 |
| [`serverless.star`](starlark/serverless.md) | Move Serverless Framework services to a new Lambda runtime. | 2 |
| [`skaffold.star`](starlark/skaffold.md) | Bump Helm chart versions deployed by Skaffold. | 2 |
| [`swift.star`](starlark/swift.md) | Bump Swift Package Manager dependencies and the swift-tools-version. | 3 |
| [`terraform.star`](starlark/terraform.md) | Bump Terraform's required_version and provider version constraints. | 3 |
| [`terragrunt.star`](starlark/terragrunt.md) | Move Terragrunt and Terraform module sources to a new `?ref=` tag. | 2 |
| [`toml.star`](starlark/toml.md) | Read, set and delete keys in TOML files, such as pyproject.toml and Cargo.toml. | 3 |
| [`yamlpath.star`](starlark/yamlpath.md) | Read, set and delete keys in any YAML file by path, keeping comments and layout. | 7 |

## All functions

| Function | What it does |
| --- | --- |
| [`gha.action_files`](starlark/gha.md#action_files) | Lists the workflows and every local action's metadata file. |
| [`scanignore.add`](starlark/scanignore.md#add) | Adds entries to a scanner's ignore file, creating it if needed. |
| [`scorecard.add_badge`](starlark/scorecard.md#add_badge) | Adds the OpenSSF Scorecard badge to the README, below its first heading. |
| [`github.add_contact_link`](starlark/github.md#add_contact_link) | Adds a contact link to the new-issue page, unless one with this URL exists. |
| [`changelog.add_entry`](starlark/changelog.md#add_entry) | Adds a bullet under the Unreleased release, in a section such as Changed or Security. |
| [`eslint.add_extends`](starlark/eslint.md#add_extends) | Adds a preset to `extends` unless it is already there. |
| [`devcontainer.add_feature`](starlark/devcontainer.md#add_feature) | Adds a feature to a config that doesn't have it yet, in any version. |
| [`codeql.add_language`](starlark/codeql.md#add_language) | Adds a language to every CodeQL workflow that doesn't analyze it yet. |
| [`renovate.add_package_rule`](starlark/renovate.md#add_package_rule) | Appends a rule to `packageRules` unless an identical rule is already there. |
| [`renovate.add_preset`](starlark/renovate.md#add_preset) | Adds a preset to `extends` unless it is already there. |
| [`github.add_release_category`](starlark/github.md#add_release_category) | Adds a category to the release notes GitHub generates, unless one with this title exists. |
| [`backstage.add_tag`](starlark/backstage.md#add_tag) | Adds a tag to `metadata.tags`, keeping the list's flow or block style. |
| [`dependabot.add_update`](starlark/dependabot.md#add_update) | Adds an update entry to an existing Dependabot config, unless one covers that ecosystem and directory. |
| [`codeql.add_workflow`](starlark/codeql.md#add_workflow) | Writes a CodeQL workflow, unless the repo already runs CodeQL. |
| [`scorecard.add_workflow`](starlark/scorecard.md#add_workflow) | Writes the Scorecard workflow, unless the repo already runs Scorecard. |
| [`argocd.applications`](starlark/argocd.md#applications) | Lists the YAML files that define Argo CD Applications or ApplicationSets. |
| [`cdk.apps`](starlark/cdk.md#apps) | Lists the repo's CDK apps. |
| [`gradle.build_files`](starlark/gradle.md#build_files) | Lists the build scripts and version catalogs. |
| [`sbt.build_files`](starlark/sbt.md#build_files) | Lists the sbt build files. |
| [`refs.bump`](starlark/refs.md#bump) | Moves every reference to `name` to a new ref. |
| [`gha.bump_action`](starlark/gha.md#bump_action) | Moves an action to a new version wherever a workflow or local action uses it. |
| [`docker.bump_base_image`](starlark/docker.md#bump_base_image) | Moves `FROM` lines of an image from one tag to another in every Dockerfile. |
| [`gitlab.bump_component`](starlark/gitlab.md#bump_component) | Moves an included CI/CD component to a new version. |
| [`bitbucket.bump_image`](starlark/bitbucket.md#bump_image) | Moves the default or a step's `image` from one tag to another, keeping variants. |
| [`circleci.bump_image`](starlark/circleci.md#bump_image) | Moves a Docker executor image from one tag to another. |
| [`devcontainer.bump_image`](starlark/devcontainer.md#bump_image) | Moves the config's `image` from one tag to another. |
| [`docker.bump_image`](starlark/docker.md#bump_image) | Moves `image:` lines from one tag to another in YAML files. |
| [`drone.bump_image`](starlark/drone.md#bump_image) | Moves a step or plugin image from one tag to another, keeping variants. |
| [`gitlab.bump_image`](starlark/gitlab.md#bump_image) | Moves a job or service image from one tag to another. |
| [`jenkins.bump_image`](starlark/jenkins.md#bump_image) | Moves a container image used by an agent or step from one tag to another. |
| [`jenkins.bump_library`](starlark/jenkins.md#bump_library) | Moves a shared library to a new version in every Jenkinsfile. |
| [`circleci.bump_orb`](starlark/circleci.md#bump_orb) | Moves an orb to a new version in every config. |
| [`bitbucket.bump_pipe`](starlark/bitbucket.md#bump_pipe) | Moves a pipe to a new version. |
| [`buildkite.bump_plugin`](starlark/buildkite.md#bump_plugin) | Moves a plugin to a new version in every pipeline. |
| [`azure_pipelines.bump_task`](starlark/azure_pipelines.md#bump_task) | Moves a task to a new major version, such as `Docker@1` to `Docker@2`. |
| [`helm.charts`](starlark/helm.md#charts) | Lists the repo's Helm charts. |
| [`gitlab.components`](starlark/gitlab.md#components) | Lists the CI/CD components a config includes, with their versions. |
| [`dependabot.config`](starlark/dependabot.md#config) | Builds the text of a dependabot.yml with one update entry per ecosystem. |
| [`golangci.config_version`](starlark/golangci.md#config_version) | Reads the config format version. |
| [`circleci.configs`](starlark/circleci.md#configs) | Lists the repo's CircleCI configs. |
| [`gitlab.configs`](starlark/gitlab.md#configs) | Lists the repo's GitLab CI files. |
| [`jsonfile.delete`](starlark/jsonfile.md#delete) | Removes a key by dotted key path. |
| [`toml.delete`](starlark/toml.md#delete) | Removes a key from a table. |
| [`yamlpath.delete`](starlark/yamlpath.md#delete) | Removes a key and everything nested under it. |
| [`license.detect`](starlark/license.md#detect) | Identifies the license from the license file's text. |
| [`golangci.disable_linter`](starlark/golangci.md#disable_linter) | Disables a linter, removing it from `enable` if it was there. |
| [`docker.dockerfiles`](starlark/docker.md#dockerfiles) | Lists the repo's Dockerfiles. |
| [`dependabot.ecosystems`](starlark/dependabot.md#ecosystems) | Detects the package ecosystems a repo uses, from files at its root. |
| [`golangci.enable_linter`](starlark/golangci.md#enable_linter) | Enables a linter, removing it from `disable` if it was there. |
| [`backstage.ensure`](starlark/backstage.md#ensure) | Writes a Component catalog file, unless the repo already has one. |
| [`markdown.ensure_badge`](starlark/markdown.md#ensure_badge) | Adds a badge under the first heading of a Markdown file. |
| [`files.ensure_lines`](starlark/files.md#ensure_lines) | Appends each of `lines` that a file does not already contain. |
| [`gha.ensure_permissions`](starlark/gha.md#ensure_permissions) | Adds a top-level `permissions` block to each workflow that has none. |
| [`backstage.entities`](starlark/backstage.md#entities) | Lists the entities in every catalog file. |
| [`scanignore.entries`](starlark/scanignore.md#entries) | Lists the entries in a scanner's ignore file. |
| [`devcontainer.features`](starlark/devcontainer.md#features) | Lists the features a config installs, with their versions. |
| [`backstage.files`](starlark/backstage.md#files) | Lists the repo's Backstage catalog files. |
| [`cmake.files`](starlark/cmake.md#files) | Lists the repo's CMake files. |
| [`devcontainer.files`](starlark/devcontainer.md#files) | Lists the repo's Dev Container configs. |
| [`helmfile.files`](starlark/helmfile.md#files) | Lists the repo's Helmfiles. |
| [`kustomize.files`](starlark/kustomize.md#files) | Lists the repo's kustomization files. |
| [`serverless.files`](starlark/serverless.md#files) | Lists the repo's Serverless configs. |
| [`skaffold.files`](starlark/skaffold.md#files) | Lists the repo's Skaffold configs. |
| [`terraform.files`](starlark/terraform.md#files) | Lists the repo's Terraform files. |
| [`changelog.find`](starlark/changelog.md#find) | Finds the repo's changelog. |
| [`codeowners.find`](starlark/codeowners.md#find) | Finds the CODEOWNERS file GitHub uses. |
| [`dependabot.find`](starlark/dependabot.md#find) | Finds the repo's Dependabot configuration. |
| [`eslint.find`](starlark/eslint.md#find) | Finds the repo's ESLint config. |
| [`golangci.find`](starlark/golangci.md#find) | Finds the repo's golangci-lint config. |
| [`license.find`](starlark/license.md#find) | Finds the repo's license file. |
| [`prettier.find`](starlark/prettier.md#find) | Finds the repo's Prettier config. |
| [`refs.find`](starlark/refs.md#find) | Lists every reference to `name` in the files matching `files`. |
| [`renovate.find`](starlark/renovate.md#find) | Finds the repo's Renovate config. |
| [`ruby.gemfiles`](starlark/ruby.md#gemfiles) | Lists the Gemfiles and gemspecs. |
| [`editorconfig.get`](starlark/editorconfig.md#get) | Reads a property. |
| [`jsonfile.get`](starlark/jsonfile.md#get) | Reads a value by dotted key path. |
| [`prettier.get`](starlark/prettier.md#get) | Reads an option from the Prettier config. |
| [`toml.get`](starlark/toml.md#get) | Reads a key's value. |
| [`yamlpath.get`](starlark/yamlpath.md#get) | Reads a scalar value by key path. |
| [`yamlpath.get_in`](starlark/yamlpath.md#get_in) | Reads a scalar value by key path from YAML text rather than a file. |
| [`make.get_var`](starlark/make.md#get_var) | Reads the value of the first assignment to a variable. |
| [`gomod.go_version`](starlark/gomod.md#go_version) | Reads the version in a go.mod's `go` directive. |
| [`lintconfig.hadolint_ignore`](starlark/lintconfig.md#hadolint_ignore) | Adds rules to the `ignored` list of .hadolint.yaml, creating the file if needed. |
| [`markdown.has_section`](starlark/markdown.md#has_section) | Reports whether a Markdown file has a heading with this text. |
| [`flux.helm_releases`](starlark/flux.md#helm_releases) | Lists the YAML files that define Flux HelmReleases. |
| [`github.issue_templates`](starlark/github.md#issue_templates) | Lists the repo's issue templates and forms. |
| [`jenkins.jenkinsfiles`](starlark/jenkins.md#jenkinsfiles) | Lists the repo's Jenkinsfiles. |
| [`codeql.languages`](starlark/codeql.md#languages) | Lists the languages CodeQL analyzes, across every CodeQL workflow. |
| [`jenkins.libraries`](starlark/jenkins.md#libraries) | Lists the shared libraries Jenkinsfiles load, with their versions. |
| [`make.makefiles`](starlark/make.md#makefiles) | Lists the repo's Makefiles. |
| [`cargo.manifests`](starlark/cargo.md#manifests) | Lists the repo's Cargo manifests. |
| [`k8s.manifests`](starlark/k8s.md#manifests) | Lists the YAML files that hold Kubernetes resources. |
| [`php.manifests`](starlark/php.md#manifests) | Lists the repo's composer.json files. |
| [`swift.manifests`](starlark/swift.md#manifests) | Lists the repo's Package.swift files. |
| [`lintconfig.markdownlint_rule`](starlark/lintconfig.md#markdownlint_rule) | Sets a markdownlint rule, such as `MD013` (line length). |
| [`k8s.migrate_api_versions`](starlark/k8s.md#migrate_api_versions) | Moves resources off API versions Kubernetes removed. |
| [`cmake.minimum_version`](starlark/cmake.md#minimum_version) | Reads the version from `cmake_minimum_required`. |
| [`gomod.modules`](starlark/gomod.md#modules) | Lists the repo's Go modules. |
| [`circleci.orbs`](starlark/circleci.md#orbs) | Lists the orbs configs use, with their versions. |
| [`gomod.outdated`](starlark/gomod.md#outdated) | Finds the modules that require a dependency below a version. |
| [`codeowners.owners`](starlark/codeowners.md#owners) | Reads the owners of a CODEOWNERS pattern. |
| [`npm.packages`](starlark/npm.md#packages) | Lists the repo's package.json files. |
| [`gha.pin`](starlark/gha.md#pin) | Pins actions to commit SHAs, keeping the old ref as a comment. |
| [`refs.pin`](starlark/refs.md#pin) | Pins references to commit SHAs, keeping the old ref in a comment. |
| [`azure_pipelines.pipelines`](starlark/azure_pipelines.md#pipelines) | Lists the repo's Azure Pipelines files. |
| [`buildkite.pipelines`](starlark/buildkite.md#pipelines) | Lists the repo's pipeline files. |
| [`drone.pipelines`](starlark/drone.md#pipelines) | Lists the repo's Drone and Woodpecker pipeline files. |
| [`bitbucket.pipes`](starlark/bitbucket.md#pipes) | Lists the pipes the pipeline uses, with their versions. |
| [`buildkite.plugins`](starlark/buildkite.md#plugins) | Lists every plugin a pipeline uses, with its version. |
| [`cocoapods.podfiles`](starlark/cocoapods.md#podfiles) | Lists the repo's Podfiles. |
| [`maven.poms`](starlark/maven.md#poms) | Lists the repo's pom.xml files. |
| [`dotnet.projects`](starlark/dotnet.md#projects) | Lists the project files and shared MSBuild props files. |
| [`elixir.projects`](starlark/elixir.md#projects) | Lists the repo's mix.exs files. |
| [`scanignore.prune_expired`](starlark/scanignore.md#prune_expired) | Removes .trivyignore entries whose expiry date has passed. |
| [`buildkite.queues`](starlark/buildkite.md#queues) | Lists the agent queues pipelines target. |
| [`devcontainer.read`](starlark/devcontainer.md#read) | Reads a Dev Container config, ignoring comments and trailing commas. |
| [`jsonfile.read`](starlark/jsonfile.md#read) | Reads a JSON file into a value whose dict keys keep the file's order. |
| [`npm.read_package`](starlark/npm.md#read_package) | Reads a package.json as a dict whose keys keep the file's order. |
| [`scanignore.remove`](starlark/scanignore.md#remove) | Removes entries from a scanner's ignore file. |
| [`files.remove_lines`](starlark/files.md#remove_lines) | Removes every line equal to one of `lines`, after trimming whitespace. |
| [`renovate.remove_preset`](starlark/renovate.md#remove_preset) | Removes a preset from `extends`, such as a retired `config:base`. |
| [`gha.rename_branch`](starlark/gha.md#rename_branch) | Renames a branch in workflow triggers and branch checks, as when `master` becomes `main`. |
| [`files.replace_all`](starlark/files.md#replace_all) | Replaces every occurrence of a string with another, literally, in text files. |
| [`ansible.requirements`](starlark/ansible.md#requirements) | Lists the repo's Ansible Galaxy requirements files. |
| [`python.requirements_files`](starlark/python.md#requirements_files) | Lists the repo's requirements files. |
| [`gomod.requires`](starlark/gomod.md#requires) | Reads every module a go.mod requires, in single-line and block form. |
| [`python.requires_python`](starlark/python.md#requires_python) | Reads the `requires-python` of a pyproject.toml. |
| [`k8s.resources`](starlark/k8s.md#resources) | Lists every resource in the repo's manifests. |
| [`precommit.revs`](starlark/precommit.md#revs) | Reads the revision each hook repository is pinned to. |
| [`cargo.rust_version`](starlark/cargo.md#rust_version) | Reads the `rust-version` of a Cargo.toml. |
| [`android.sdk_levels`](starlark/android.md#sdk_levels) | Reads the SDK levels every module's build file sets. |
| [`dotnet.sdk_version`](starlark/dotnet.md#sdk_version) | Reads the .NET SDK version pinned in global.json. |
| [`devcontainer.set`](starlark/devcontainer.md#set) | Sets a top-level setting, such as `postCreateCommand` or `remoteUser`. |
| [`editorconfig.set`](starlark/editorconfig.md#set) | Sets a property, adding it, or its section, when missing. |
| [`gitattributes.set`](starlark/gitattributes.md#set) | Gives a pattern attributes, merging with any it already has. |
| [`jsonfile.set`](starlark/jsonfile.md#set) | Sets a value by dotted key path, adding missing objects on the way. |
| [`toml.set`](starlark/toml.md#set) | Sets a key, adding it, and its table, when missing. |
| [`yamlpath.set`](starlark/yamlpath.md#set) | Sets a key to a value, adding the key and any missing parents if needed. |
| [`android.set_agp_version`](starlark/android.md#set_agp_version) | Moves the Android Gradle Plugin to a version. |
| [`backstage.set_annotation`](starlark/backstage.md#set_annotation) | Sets a `metadata.annotations` entry, such as `backstage.io/techdocs-ref`. |
| [`helm.set_app_version`](starlark/helm.md#set_app_version) | Sets a chart's `appVersion` in its Chart.yaml, keeping its quotes. |
| [`bazel.set_archive`](starlark/bazel.md#set_archive) | Points an `http_archive` at a new download, with its checksum. |
| [`github.set_blank_issues`](starlark/github.md#set_blank_issues) | Allows or disallows issues that don't use a template. |
| [`ruby.set_bundler`](starlark/ruby.md#set_bundler) | Sets the Bundler version a Gemfile.lock was `BUNDLED WITH`. |
| [`flux.set_chart_version`](starlark/flux.md#set_chart_version) | Sets the chart `version` of every HelmRelease that installs a chart. |
| [`helm.set_chart_version`](starlark/helm.md#set_chart_version) | Sets a chart's `version` in its Chart.yaml. |
| [`helmfile.set_chart_version`](starlark/helmfile.md#set_chart_version) | Sets the `version` of every release that installs a chart. |
| [`cdk.set_context`](starlark/cdk.md#set_context) | Sets a context value, such as a feature flag, in every cdk.json. |
| [`gitlab.set_default`](starlark/gitlab.md#set_default) | Sets a key under `default:` in .gitlab-ci.yml, adding `default:` if needed. |
| [`bazel.set_dep`](starlark/bazel.md#set_dep) | Sets the version of a `bazel_dep` in every MODULE.bazel that declares it. |
| [`elixir.set_dep`](starlark/elixir.md#set_dep) | Sets a dependency's requirement wherever a mix.exs lists it with one. |
| [`buf.set_dep_ref`](starlark/buf.md#set_dep_ref) | Pins a module dependency in buf.yaml to a label or commit, such as `buf.build/googleapis/googleapis:<commit>`. |
| [`cargo.set_dependency`](starlark/cargo.md#set_dependency) | Sets a crate's version wherever a Cargo.toml already depends on it. |
| [`gradle.set_dependency`](starlark/gradle.md#set_dependency) | Sets a library's version wherever the build declares it. |
| [`helm.set_dependency`](starlark/helm.md#set_dependency) | Sets the version of a chart dependency in every Chart.yaml that lists it. |
| [`maven.set_dependency`](starlark/maven.md#set_dependency) | Sets the version of a dependency or plugin wherever a pom.xml declares one. |
| [`npm.set_dependency`](starlark/npm.md#set_dependency) | Sets a dependency's version range in every package.json that already lists it. |
| [`sbt.set_dependency`](starlark/sbt.md#set_dependency) | Sets the version of a library or sbt plugin wherever the build declares it. |
| [`elixir.set_elixir`](starlark/elixir.md#set_elixir) | Sets the `elixir:` requirement in each project definition. |
| [`devcontainer.set_feature_version`](starlark/devcontainer.md#set_feature_version) | Moves a feature to a new major version, such as `docker-in-docker:1` to `:2`. |
| [`cmake.set_fetchcontent_tag`](starlark/cmake.md#set_fetchcontent_tag) | Sets the `GIT_TAG` of a `FetchContent_Declare` dependency. |
| [`backstage.set_field`](starlark/backstage.md#set_field) | Sets any entity field by key path, such as `spec.lifecycle` or `spec.system`. |
| [`github.set_funding`](starlark/github.md#set_funding) | Sets a sponsor link in .github/FUNDING.yml, such as `github: [acme]`. |
| [`ruby.set_gem`](starlark/ruby.md#set_gem) | Sets a gem's version requirement wherever a Gemfile or gemspec lists it. |
| [`flux.set_git_tag`](starlark/flux.md#set_git_tag) | Sets `ref.tag` of every GitRepository that points at a URL. |
| [`gomod.set_go_version`](starlark/gomod.md#set_go_version) | Raises the `go` directive in every go.mod, never lowering it. |
| [`skaffold.set_helm_release_version`](starlark/skaffold.md#set_helm_release_version) | Sets the `version` of every Helm release that installs a remote chart. |
| [`kustomize.set_image`](starlark/kustomize.md#set_image) | Sets `newTag`, `newName` or `digest` for an image in every kustomization's `images` list. |
| [`helm.set_image_tag`](starlark/helm.md#set_image_tag) | Sets the `tag` next to `repository: <repository>` in values files. |
| [`yamlpath.set_in`](starlark/yamlpath.md#set_in) | Sets a key in YAML text rather than a file, the same way `set` does. |
| [`gitlab.set_include_ref`](starlark/gitlab.md#set_include_ref) | Sets the `ref` of every `include` that pulls templates from another project. |
| [`gha.set_input`](starlark/gha.md#set_input) | Sets every `<key>: <value>` line in the workflows, such as an action input. |
| [`gradle.set_kotlin_version`](starlark/gradle.md#set_kotlin_version) | Moves every Kotlin Gradle plugin, and the Kotlin standard library, to one version. |
| [`github.set_label`](starlark/github.md#set_label) | Adds or updates a label in a label sync file, such as .github/labels.yml. |
| [`k8s.set_label`](starlark/k8s.md#set_label) | Sets a label on every resource, and on the pods workloads create. |
| [`circleci.set_machine_image`](starlark/circleci.md#set_machine_image) | Sets the `image` of every machine executor, such as when an Ubuntu image is retired. |
| [`cmake.set_minimum_version`](starlark/cmake.md#set_minimum_version) | Raises `cmake_minimum_required` to a version, never lowering it. |
| [`npm.set_npmrc`](starlark/npm.md#set_npmrc) | Sets a `key=value` setting in .npmrc, adding it, or the file, when missing. |
| [`dependabot.set_option`](starlark/dependabot.md#set_option) | Sets an option on update entries, such as `open-pull-requests-limit` or `rebase-strategy`. |
| [`prettier.set_option`](starlark/prettier.md#set_option) | Sets an option in the Prettier config, keeping its format. |
| [`renovate.set_option`](starlark/renovate.md#set_option) | Sets a top-level or nested option, such as `schedule` or `lockFileMaintenance.enabled`. |
| [`backstage.set_owner`](starlark/backstage.md#set_owner) | Sets `spec.owner`, such as after a team is renamed or a service changes hands. |
| [`codeowners.set_owners`](starlark/codeowners.md#set_owners) | Makes a list of users or teams the owners of a pattern. |
| [`dotnet.set_package`](starlark/dotnet.md#set_package) | Sets a NuGet package's version wherever a project already references it. |
| [`swift.set_package`](starlark/swift.md#set_package) | Sets the version of a package dependency wherever a manifest declares it. |
| [`npm.set_package_manager`](starlark/npm.md#set_package_manager) | Sets the `packageManager` field Corepack uses, such as `pnpm@9.12.0`, where a package already has one. |
| [`php.set_php`](starlark/php.md#set_php) | Sets the PHP version a project requires, and optionally its `config.platform.php`. |
| [`cocoapods.set_platform`](starlark/cocoapods.md#set_platform) | Raises the deployment target in `platform :ios, '13.0'`, never lowering it. |
| [`gradle.set_plugin`](starlark/gradle.md#set_plugin) | Sets the version of a Gradle plugin wherever the build applies or declares it. |
| [`buf.set_plugin_version`](starlark/buf.md#set_plugin_version) | Moves a remote plugin to a new version in every buf.gen.yaml. |
| [`packer.set_plugin_version`](starlark/packer.md#set_plugin_version) | Sets the `version` of a plugin in `required_plugins`, matched by its `source`. |
| [`cocoapods.set_pod`](starlark/cocoapods.md#set_pod) | Sets a pod's version requirement wherever a Podfile lists it with one. |
| [`maven.set_property`](starlark/maven.md#set_property) | Sets a `<properties>` entry, such as `java.version`, in every pom.xml that defines it. |
| [`terraform.set_provider_version`](starlark/terraform.md#set_provider_version) | Sets the version constraint of a provider in `required_providers`. |
| [`buildkite.set_queue`](starlark/buildkite.md#set_queue) | Points `agents: queue:` at a new queue in every pipeline. |
| [`php.set_require`](starlark/php.md#set_require) | Sets a package's constraint wherever `require` or `require-dev` lists it. |
| [`ruby.set_required_ruby`](starlark/ruby.md#set_required_ruby) | Sets `required_ruby_version` in every gemspec that declares it. |
| [`terraform.set_required_version`](starlark/terraform.md#set_required_version) | Sets every `required_version` constraint. |
| [`python.set_requirement`](starlark/python.md#set_requirement) | Sets a package's version specifier wherever it is already required. |
| [`python.set_requires_python`](starlark/python.md#set_requires_python) | Sets `requires-python` in a pyproject.toml that already has it. |
| [`circleci.set_resource_class`](starlark/circleci.md#set_resource_class) | Sets `resource_class` on every job and executor that sets one. |
| [`precommit.set_rev`](starlark/precommit.md#set_rev) | Pins a hook repository to a new revision, keeping quotes and comments. |
| [`ruby.set_rubocop_target`](starlark/ruby.md#set_rubocop_target) | Sets `AllCops: TargetRubyVersion` in .rubocop.yml. |
| [`ruby.set_ruby`](starlark/ruby.md#set_ruby) | Sets the Ruby version a Gemfile pins with `ruby "x"`. |
| [`eslint.set_rule`](starlark/eslint.md#set_rule) | Sets a rule's level, and optionally its options. |
| [`serverless.set_runtime`](starlark/serverless.md#set_runtime) | Sets the Lambda runtime of the provider and of every function that overrides it. |
| [`cargo.set_rust_version`](starlark/cargo.md#set_rust_version) | Raises `rust-version` in every Cargo.toml that sets it, never lowering it. |
| [`sbt.set_sbt_version`](starlark/sbt.md#set_sbt_version) | Sets `sbt.version` in project/build.properties. |
| [`sbt.set_scala_version`](starlark/sbt.md#set_scala_version) | Sets `scalaVersion`, and matching entries of `crossScalaVersions`. |
| [`android.set_sdk`](starlark/android.md#set_sdk) | Raises the compile, target and min SDK levels, never lowering one. |
| [`dotnet.set_sdk_version`](starlark/dotnet.md#set_sdk_version) | Sets the .NET SDK version in a global.json that already pins one. |
| [`markdown.set_section`](starlark/markdown.md#set_section) | Makes the section under a heading contain the given text. |
| [`golangci.set_setting`](starlark/golangci.md#set_setting) | Sets a linter's setting, in `linters-settings` (v1) or `linters.settings` (v2). |
| [`yamlpath.set_sibling`](starlark/yamlpath.md#set_sibling) | Sets `key` in every mapping where `match_key` has the value `match_value`, list items included. |
| [`terragrunt.set_source_ref`](starlark/terragrunt.md#set_source_ref) | Sets the `?ref=` of every module source that matches. |
| [`license.set_spdx`](starlark/license.md#set_spdx) | Sets the license field in package manifests that already have one. |
| [`buildkite.set_step_defaults`](starlark/buildkite.md#set_step_defaults) | Adds keys, such as `timeout_in_minutes`, to every command step that lacks them. |
| [`dotnet.set_target_framework`](starlark/dotnet.md#set_target_framework) | Sets the target framework of every project, such as `net8.0`. |
| [`argocd.set_target_revision`](starlark/argocd.md#set_target_revision) | Sets `targetRevision` for every source that points at a repository. |
| [`azure_pipelines.set_template_ref`](starlark/azure_pipelines.md#set_template_ref) | Sets the `ref` of a template repository under `resources.repositories`. |
| [`jenkins.set_tool`](starlark/jenkins.md#set_tool) | Sets the installation a `tools` block asks for, such as `maven 'Maven 3.9'`. |
| [`swift.set_tools_version`](starlark/swift.md#set_tools_version) | Raises the `// swift-tools-version:` line, never lowering it. |
| [`sbt.set_val`](starlark/sbt.md#set_val) | Sets a string `val`, such as `val catsVersion = "2.9.0"`, wherever it is defined. |
| [`make.set_var`](starlark/make.md#set_var) | Sets every assignment to a variable, in every Makefile that has one. |
| [`ansible.set_version`](starlark/ansible.md#set_version) | Sets the version of a role or collection wherever a requirements file lists it. |
| [`bazel.set_version`](starlark/bazel.md#set_version) | Sets the Bazel version in .bazelversion, when the repo pins one. |
| [`codeql.set_version`](starlark/codeql.md#set_version) | Moves every `github/codeql-action` action (`init`, `analyze`, `upload-sarif`...) to a new version. |
| [`runtime.set_version`](starlark/runtime.md#set_version) | Sets a runtime's version in every version file the repo already has. |
| [`scorecard.set_version`](starlark/scorecard.md#set_version) | Moves `ossf/scorecard-action` to a new version. |
| [`azure_pipelines.set_vm_image`](starlark/azure_pipelines.md#set_vm_image) | Sets `vmImage` on every pool that uses a Microsoft-hosted image. |
| [`gomod.set_work_go_version`](starlark/gomod.md#set_work_go_version) | Raises the `go` directive in go.work, never lowering it. |
| [`gradle.set_wrapper_version`](starlark/gradle.md#set_wrapper_version) | Points the Gradle wrapper at a new version, never lowering it. |
| [`license.set_year`](starlark/license.md#set_year) | Extends the copyright year in the license file to `year`. |
| [`terragrunt.sources`](starlark/terragrunt.md#sources) | Lists every module source pinned with `?ref=`. |
| [`files.sync`](starlark/files.md#sync) | Copies every file under a directory next to the campaign file into the repo. |
| [`packer.templates`](starlark/packer.md#templates) | Lists the repo's Packer HCL templates. |
| [`gitattributes.track_lfs`](starlark/gitattributes.md#track_lfs) | Tracks a pattern with Git LFS, as `git lfs track` would. |
| [`gha.unpinned`](starlark/gha.md#unpinned) | Lists each remote action used at a tag or branch rather than a commit SHA. |
| [`yamlpath.values`](starlark/yamlpath.md#values) | Reads every scalar whose key path matches a pattern with `*` wildcards. |
| [`bazel.version`](starlark/bazel.md#version) | Reads the Bazel version pinned in .bazelversion. |
| [`runtime.version`](starlark/runtime.md#version) | Reads the version a repo pins for a runtime. |
| [`scorecard.workflow`](starlark/scorecard.md#workflow) | Finds the workflow that runs Scorecard. |
| [`codeql.workflows`](starlark/codeql.md#workflows) | Lists the workflows that run CodeQL. |
| [`gha.workflows`](starlark/gha.md#workflows) | Lists the repo's workflow files. |
| [`gradle.wrapper_version`](starlark/gradle.md#wrapper_version) | Reads the Gradle version the wrapper downloads. |
| [`jsonfile.write`](starlark/jsonfile.md#write) | Writes a value as JSON, using the indentation the file already has. |
| [`dependabot.write_if_missing`](starlark/dependabot.md#write_if_missing) | Writes .github/dependabot.yml for the detected ecosystems unless a config exists. |
| [`files.write_if_missing`](starlark/files.md#write_if_missing) | Writes a file unless it, or any of the alternative paths, already exists. |
| [`npm.write_package`](starlark/npm.md#write_package) | Writes a dict as a package.json, keeping the file's indentation. |
| [`lintconfig.yamllint_rule`](starlark/lintconfig.md#yamllint_rule) | Sets a yamllint rule, such as turning off `document-start`. |
