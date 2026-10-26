---
title: Examples
---

# Examples

Campaigns for the changes people make most, ready to copy from [`examples/`](https://github.com/rossedman/gh-fleet/tree/main/examples). Each directory
holds the campaign file and a `change.star` with the [Starlark](guides/starlark.md)
that does the work, plus a PR body, template or files to sync where needed.
Each name below opens a page with every file of the campaign and its test
cases, generated from the directory so it always matches.
Start from one with `gh fleet init`, which copies it with its test cases
and points it at your organization:

```sh
gh fleet init --from sync-files --org your-org
gh fleet targets
gh fleet plan --repo your-org/some-repo
```

Every change is written in Starlark on top of the
[built-in library](reference/starlark-library.md) (`load("@fleet//gha.star", ...)`), usually in a
handful of lines: no shell quoting, no GNU-versus-BSD differences, and no access
to anything outside the clone. Where a change needs a real tool (`gh api`,
`go get`, `npm`, `cargo`), Starlark decides what to do and hands a list to a
short inline `run` step, never more than a few lines. Every example is tested
in [`examples_test.go`](https://github.com/rossedman/gh-fleet/blob/main/examples/examples_test.go), [`library_test.go`](https://github.com/rossedman/gh-fleet/blob/main/examples/library_test.go), [`more_test.go`](https://github.com/rossedman/gh-fleet/blob/main/examples/more_test.go) and [`starlark_test.go`](https://github.com/rossedman/gh-fleet/blob/main/examples/starlark_test.go), including that a second run changes nothing, so re-running a
campaign is safe.

## Files and text

| Campaign | What it changes | Shows |
| --- | --- | --- |
| [`sync-files`](reference/examples/sync-files.md) | Copies the files under `files/` into every repo, replacing them | `files.sync` and the read-only `campaign` object; edit once, re-apply to update every PR |
| [`find-replace`](reference/examples/find-replace.md) | Replaces a literal string in every text file | code search targeting, `env` parameters, `files.replace_all` skipping binaries |
| [`remove-files`](reference/examples/remove-files.md) | Deletes retired files and directories | `repo.delete` with paths that cannot leave the repo |
| [`gitignore-entries`](reference/examples/gitignore-entries.md) | Appends missing patterns to `.gitignore` | multi-line `env` values, `files.ensure_lines` |
| [`readme-section`](reference/examples/readme-section.md) | Writes a Security section and a CI badge into every README | `markdown.set_section` and `markdown.ensure_badge`, per-repo URLs |
| [`security-policy`](reference/examples/security-policy.md) | Adds `SECURITY.md` from a template, unless one exists | a template shipped with the campaign, filled in per repo |
| [`add-codeowners`](reference/examples/add-codeowners.md) | Writes `.github/CODEOWNERS` naming each repo's team | per-repo variables from defaults, match rules and a CSV, read as `repo.vars` |
| [`changelog-entry`](reference/examples/changelog-entry.md) | Patches lodash and records it under Unreleased > Security in `CHANGELOG.md` | `changelog.add_entry` as part of another change |
| [`license-year`](reference/examples/license-year.md) | Extends the copyright year in MIT and Apache-2.0 licenses | `license.detect` and `license.set_year` |
| [`editor-defaults`](reference/examples/editor-defaults.md) | Sets `.editorconfig` and `.gitattributes` defaults | `editorconfig.set` and `gitattributes.set`, merging with what is there |
| [`github-repo-files`](reference/examples/github-repo-files.md) | Adds release note categories, a label and a security link on the new-issue page | `github.add_release_category`, `set_label` and `add_contact_link` |

## Dependencies

| Campaign | What it changes | Shows |
| --- | --- | --- |
| [`go-module-bump`](reference/examples/go-module-bump.md) | `go get module@version` and `go mod tidy` in every module below that version | Starlark picks the modules (`gomod.outdated`), a three-line `run` step updates them |
| [`npm-package-bump`](reference/examples/npm-package-bump.md) | Sets a package's range in every `package.json` and refreshes the lockfile | `npm.set_dependency`, then a short `run` step reads `FLEET_VAR_DIRS` |
| [`python-requirement-bump`](reference/examples/python-requirement-bump.md) | Pins a package in `requirements*.txt` and `pyproject.toml` | `python.set_requirement` keeping extras, markers and comments |
| [`cargo-dependency-bump`](reference/examples/cargo-dependency-bump.md) | Sets a crate's version in every `Cargo.toml` and refreshes `Cargo.lock` | `cargo.set_dependency` for plain and table forms |
| [`jvm-dependency-bump`](reference/examples/jvm-dependency-bump.md) | Moves jackson-databind to 2.17.2 in Maven poms and Gradle builds | `maven.set_dependency` following `${property}` references, `gradle.set_dependency` including `version.ref` |
| [`android-target-sdk`](reference/examples/android-target-sdk.md) | Raises compileSdk and targetSdk to 35 in every Android module | `android.set_sdk` on Groovy, Kotlin DSL and catalogs, never lowering |
| [`ruby-rails-bump`](reference/examples/ruby-rails-bump.md) | Moves the Rails requirement to `~> 7.2` and re-locks only rails | `ruby.set_gem`, then a one-line `bundle lock` step |
| [`php-laravel-11`](reference/examples/php-laravel-11.md) | Moves Laravel apps to Laravel 11 and PHP 8.2, then updates the lock for laravel only | `php.set_require` and `php.set_php`, then a one-line `composer update` |
| [`sbt-scala-bump`](reference/examples/sbt-scala-bump.md) | Moves Scala 2.13 builds to 2.13.15 and sbt 1.10.2, keeping cross-built Scala 3 | `sbt.set_scala_version` with `from_version`, and `sbt.set_sbt_version` |
| [`bazel-module-bump`](reference/examples/bazel-module-bump.md) | Moves Bazel to 7.4.1 and rules_go and gazelle to new versions | `bazel.set_version` and `bazel.set_dep` on multi-line calls |
| [`swift-package-bump`](reference/examples/swift-package-bump.md) | Moves swift-log to 1.6.1 and raises swift-tools-version | `swift.set_package` keeping the kind of requirement |
| [`cocoapods-ios-15`](reference/examples/cocoapods-ios-15.md) | Raises iOS apps to iOS 15 and Alamofire 5.9 | `cocoapods.set_platform` (never lowering) and `cocoapods.set_pod` |
| [`elixir-phoenix`](reference/examples/elixir-phoenix.md) | Moves Phoenix apps to 1.7.14 on Elixir 1.17 and OTP 27 | `elixir.set_dep`, `elixir.set_elixir` and `runtime.set_version` |
| [`buf-plugins`](reference/examples/buf-plugins.md) | Moves Buf remote plugins for Go to current versions | `buf.set_plugin_version` |
| [`node-pnpm-9`](reference/examples/node-pnpm-9.md) | Moves pnpm projects to pnpm 9 and turns on engine-strict | `npm.set_package_manager` and `npm.set_npmrc` |
| [`terraform-provider-bump`](reference/examples/terraform-provider-bump.md) | Moves the AWS provider to `~> 5.0` and raises `required_version` | `terraform.set_provider_version` on block and one-line entries |
| [`terragrunt-module-ref`](reference/examples/terragrunt-module-ref.md) | Moves every `?ref=` of a shared Terraform modules repo to v2.0.0 | `terragrunt.set_source_ref` with a `*` source, `.tf` and `terragrunt.hcl` alike |
| [`ansible-collections`](reference/examples/ansible-collections.md) | Sets a collection's version in every `requirements.yml` | `ansible.set_version`, adding a missing `version` |
| [`cmake-minimum`](reference/examples/cmake-minimum.md) | Raises `cmake_minimum_required` and moves a FetchContent tag | `cmake.set_minimum_version` keeping `...max` ranges |
| [`node-engines`](reference/examples/node-engines.md) | Requires Node 22 in every `package.json`, never lowering a newer requirement | JSON edited as data with key order kept, values handed back to the PR |
| [`docker-base-image`](reference/examples/docker-base-image.md) | Moves `FROM node:18` to `node:22`, keeping variants and stages | `docker.bump_base_image`, draft PRs |
| [`compose-image-bump`](reference/examples/compose-image-bump.md) | Moves `image: redis:7` to `redis:8` in Compose files, Kubernetes manifests and Dockerfiles | `docker.bump_image` |
| [`add-dependabot`](reference/examples/add-dependabot.md) | Writes `.github/dependabot.yml` for the ecosystems a repo uses | detecting what a repo contains with `dependabot.ecosystems` |
| [`dependabot-ecosystems`](reference/examples/dependabot-ecosystems.md) | Adds docker and github-actions entries to existing Dependabot configs | `dependabot.add_update` and `dependabot.set_option` |

## Config files

| Campaign | What it changes | Shows |
| --- | --- | --- |
| [`ruff-config`](reference/examples/ruff-config.md) | Adds shared ruff rules to `pyproject.toml` without replacing a project's own, adding tables where missing | `toml.get` and `toml.set` keeping comments |
| [`renovate-preset`](reference/examples/renovate-preset.md) | Points every `renovate.json` at a shared preset | `jsonfile.set` keeping key order and indentation |
| [`renovate-automerge`](reference/examples/renovate-automerge.md) | Moves off `config:base` and automerges minor and patch devDependency updates | `renovate.replace_preset` and `add_package_rule` |
| [`golangci-v2`](reference/examples/golangci-v2.md) | Migrates golangci-lint configs to v2 and enables gosec | Starlark, the official `migrate` command, then Starlark again |
| [`devcontainer-image`](reference/examples/devcontainer-image.md) | Moves Dev Containers to Go 1.23 and adds the GitHub CLI feature | `devcontainer` edits that keep JSONC comments |
| [`prettier-eslint`](reference/examples/prettier-eslint.md) | Sets shared Prettier options and an ESLint rule, reporting flat configs for review | `prettier.set_option`, `eslint.set_rule` and `eslint.find` |
| [`lint-configs`](reference/examples/lint-configs.md) | Sets markdownlint and yamllint rules and adds a hadolint ignore | `lintconfig` helpers across three config formats |
| [`backstage-onboarding`](reference/examples/backstage-onboarding.md) | Writes `catalog-info.yaml` where missing, owned by the repo's `team` custom property, with TechDocs and language tags | `backstage.ensure`, `set_annotation` and `add_tag` with `repo.properties` |
| [`backstage-team-rename`](reference/examples/backstage-team-rename.md) | Moves catalog entities from one team to another after a reorg | `backstage.set_owner` matching every form of the group ref |

## Runtimes

| Campaign | What it changes | Shows |
| --- | --- | --- |
| [`bump-go-version`](reference/examples/bump-go-version.md) | Raises the `go` directive and workflow `go-version` pins | `exclude.repos`, `gomod.set_go_version` never lowering a version |
| [`python-version`](reference/examples/python-version.md) | Moves `.python-version`, `requires-python` and `setup-python` to 3.12 | three helpers in one step: `runtime`, `python` and `gha` |
| [`make-go-version`](reference/examples/make-go-version.md) | Sets `GO_VERSION` in Makefiles and raises `go.mod` to match | `make.set_var` keeping `?=`, `:=` and comments |
| [`node-version`](reference/examples/node-version.md) | Pins Node 22 in `.nvmrc`, `.tool-versions` and `setup-node`, creating `.nvmrc` where nothing pins it | `runtime.set_version` with `create`, the version as a variable |
| [`gradle-wrapper-bump`](reference/examples/gradle-wrapper-bump.md) | Moves the Gradle wrapper to 8.10.2, with the right checksum for `-bin` or `-all` | `gradle.set_wrapper_version`, never lowering a version |
| [`dotnet-8`](reference/examples/dotnet-8.md) | Retargets `net6.0` projects to `net8.0` and pins the 8.0 SDK | `dotnet.set_target_framework` on single and multi-targeted projects |

## CI and security

| Campaign | What it changes | Shows |
| --- | --- | --- |
| [`upgrade-action`](reference/examples/upgrade-action.md) | Rewrites `uses: actions/checkout@v3` to `@v4` | exact ref matching (`v3` never matches `v3.5.0`) |
| [`pin-actions`](reference/examples/pin-actions.md) | Pins every `uses:` to a commit SHA, keeping the tag as a comment | three steps: Starlark finds refs, `gh api` resolves them, Starlark pins |
| [`workflow-permissions`](reference/examples/workflow-permissions.md) | Adds a read-only top-level `permissions:` to workflows without one | line-based YAML edits so comments and layout survive |
| [`pre-commit-rev`](reference/examples/pre-commit-rev.md) | Moves a hook repository to a new `rev` in `.pre-commit-config.yaml` | `precommit.set_rev`, the old rev reported in the PR |
| [`runner-label`](reference/examples/runner-label.md) | Pins jobs on `ubuntu-latest` to `ubuntu-24.04` | `yamlpath.values` with a `*` wildcard, then `yamlpath.set` |
| [`buildkite-plugin-bump`](reference/examples/buildkite-plugin-bump.md) | Moves the docker-compose plugin to v5.4.1 in every Buildkite pipeline | `buildkite.bump_plugin`, reporting the old version |
| [`buildkite-agents`](reference/examples/buildkite-agents.md) | Moves steps to a new agent queue and adds `timeout_in_minutes` where missing | `buildkite.set_queue` and `buildkite.set_step_defaults`, including grouped steps |
| [`jenkins-shared-library`](reference/examples/jenkins-shared-library.md) | Moves a Jenkins shared library and a CircleCI orb to v3 | `jenkins.bump_library` and `circleci.bump_orb`, both built on `refs.bump` |
| [`gitlab-image-bump`](reference/examples/gitlab-image-bump.md) | Moves GitLab CI job and service images from Node 18 to 22 | `gitlab.bump_image` on `image:`, `name:` and `services` forms |
| [`gitlab-templates`](reference/examples/gitlab-templates.md) | Moves included templates and CI/CD components to v2, and makes jobs interruptible | `gitlab.set_include_ref`, `gitlab.bump_component`, `gitlab.set_default` |
| [`circleci-orbs`](reference/examples/circleci-orbs.md) | Moves the node orb to 5.2.0 in every CircleCI config | `circleci.bump_orb`, reporting the old version |
| [`circleci-machine-image`](reference/examples/circleci-machine-image.md) | Moves machine jobs off retired Ubuntu 20.04 images | `circleci.set_machine_image` with a `*` match, leaving Docker images alone |
| [`jenkins-agent-image`](reference/examples/jenkins-agent-image.md) | Moves Jenkins agents to Maven 3.9 and `tools` to JDK 21 | `jenkins.bump_image` and `jenkins.set_tool` |
| [`azure-ubuntu-24`](reference/examples/azure-ubuntu-24.md) | Moves Azure Pipelines to `ubuntu-24.04` agents and a newer task version | `azure_pipelines.set_vm_image` and `bump_task` |
| [`bitbucket-pipes`](reference/examples/bitbucket-pipes.md) | Moves the aws-s3-deploy pipe and Node images in Bitbucket Pipelines | `bitbucket.bump_pipe` and `bump_image`, listing pipes first |
| [`drone-golang`](reference/examples/drone-golang.md) | Moves Drone and Woodpecker steps to Go 1.23 | `drone.bump_image` on both CI systems |
| [`branch-rename`](reference/examples/branch-rename.md) | Updates workflow branch filters and `refs/heads/` checks after `master` becomes `main` | `gha.rename_branch` |
| [`codeql-setup`](reference/examples/codeql-setup.md) | Adds a CodeQL workflow for the repo's language, or adds `actions` to an existing one and moves it to v4 | `codeql.add_workflow`, `add_language` and `set_version` with `repo.language` |
| [`scorecard-rollout`](reference/examples/scorecard-rollout.md) | Adds the OpenSSF Scorecard workflow and README badge | `scorecard.add_workflow` and `add_badge` |
| [`trivy-ignore-prune`](reference/examples/trivy-ignore-prune.md) | Removes expired `.trivyignore` entries and lists the findings that came back | a shell step for the date, then `scanignore.prune_expired` |

## Deployments

| Campaign | What it changes | Shows |
| --- | --- | --- |
| [`helm-image-tag`](reference/examples/helm-image-tag.md) | Rolls an image tag out to every Helm chart that deploys it, bumping `appVersion` and the chart version | `helm.set_image_tag`, then reading a value back with `yamlpath.get` |
| [`kustomize-image`](reference/examples/kustomize-image.md) | Sets `newTag` for an image in every Kustomize overlay that lists it | `kustomize.set_image` |
| [`argocd-revision`](reference/examples/argocd-revision.md) | Points Argo CD Applications at a new revision of the platform manifests | `argocd.set_target_revision` across `source` and `sources` |
| [`flux-chart-bump`](reference/examples/flux-chart-bump.md) | Moves a Flux HelmRelease to a new chart version | `flux.set_chart_version` |
| [`k8s-api-migration`](reference/examples/k8s-api-migration.md) | Moves manifests off API versions Kubernetes removed | `k8s.migrate_api_versions` on multi-document files |
| [`k8s-labels`](reference/examples/k8s-labels.md) | Labels every resource and pod template with the owning team | `k8s.set_label` with a per-repo variable |
| [`helmfile-chart`](reference/examples/helmfile-chart.md) | Moves a Helmfile release to a new chart version | `helmfile.set_chart_version` across `helmfile.d/` |
| [`serverless-runtime`](reference/examples/serverless-runtime.md) | Moves Serverless Framework functions to the nodejs20.x runtime | `serverless.set_runtime` on the provider and per function |
| [`cdk-feature-flags`](reference/examples/cdk-feature-flags.md) | Turns on a CDK feature flag in apps that already set flags | `cdk.set_context` with `only_existing` |
| [`packer-plugins`](reference/examples/packer-plugins.md) | Raises the Packer amazon plugin constraint | `packer.set_plugin_version` in `required_plugins` |

## Writing your own

The [writing changes guide](guides/writing-changes.md) covers how steps run. Start from the example closest to your change, and check the
[library reference](reference/starlark-library.md) before writing a helper yourself. A few habits
make changes safe to run across hundreds of repos:

- **Make it idempotent.** Check before you change, so a second run is a
  no-op. Every `@fleet` helper already does.
- **Take parameters from `env` or `vars`.** The same script then serves many
  campaigns, and `vars` can differ per repo.
- **Keep shell to glue.** Let Starlark find the files and decide, then hand a
  list to a `run` step through `repo.set_var` and `FLEET_VAR_<NAME>`. If a
  `run` step grows past a few lines, the logic belongs in Starlark.
- **Never run untrusted code.** Package managers get `--ignore-scripts`, and
  nothing is installed globally.
