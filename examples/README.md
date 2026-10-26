# Example campaigns

Campaigns for the changes people make most, ready to copy. Each directory
holds the campaign file and a `change.star` with the [Starlark](https://rossedman.github.io/gh-fleet/guides/starlark/)
that does the work, plus a PR body, template or files to sync where needed.
Change `orgs: [acme]` to your organization, then:

```sh
cd examples/sync-files
gh fleet targets
gh fleet plan --repo your-org/some-repo
```

Every change is written in Starlark on top of the
[built-in library](https://rossedman.github.io/gh-fleet/reference/starlark-library/) (`load("@fleet//gha.star", ...)`), usually in a
handful of lines: no shell quoting, no GNU-versus-BSD differences, and no access
to anything outside the clone. Where a change needs a real tool (`gh api`,
`go get`, `npm`, `cargo`), Starlark decides what to do and hands a list to a
short inline `run` step, never more than a few lines. Every example is tested
in `examples_test.go`, `library_test.go`, `more_test.go` and `starlark_test.go`, including that a second run changes nothing, so re-running a
campaign is safe.

## Files and text

| Campaign | What it changes | Shows |
| --- | --- | --- |
| [`sync-files`](sync-files/) | Copies the files under `files/` into every repo, replacing them | `files.sync` and the read-only `campaign` object; edit once, re-apply to update every PR |
| [`find-replace`](find-replace/) | Replaces a literal string in every text file | code search targeting, `env` parameters, `files.replace_all` skipping binaries |
| [`remove-files`](remove-files/) | Deletes retired files and directories | `repo.delete` with paths that cannot leave the repo |
| [`gitignore-entries`](gitignore-entries/) | Appends missing patterns to `.gitignore` | multi-line `env` values, `files.ensure_lines` |
| [`readme-section`](readme-section/) | Writes a Security section and a CI badge into every README | `markdown.set_section` and `markdown.ensure_badge`, per-repo URLs |
| [`security-policy`](security-policy/) | Adds `SECURITY.md` from a template, unless one exists | a template shipped with the campaign, filled in per repo |
| [`add-codeowners`](add-codeowners/) | Writes `.github/CODEOWNERS` naming each repo's team | per-repo variables from defaults, match rules and a CSV, read as `repo.vars` |
| [`changelog-entry`](changelog-entry/) | Patches lodash and records it under Unreleased > Security in `CHANGELOG.md` | `changelog.add_entry` as part of another change |
| [`license-year`](license-year/) | Extends the copyright year in MIT and Apache-2.0 licenses | `license.detect` and `license.set_year` |
| [`editor-defaults`](editor-defaults/) | Sets `.editorconfig` and `.gitattributes` defaults | `editorconfig.set` and `gitattributes.set`, merging with what is there |
| [`github-repo-files`](github-repo-files/) | Adds release note categories, a label and a security link on the new-issue page | `github.add_release_category`, `set_label` and `add_contact_link` |

## Dependencies

| Campaign | What it changes | Shows |
| --- | --- | --- |
| [`go-module-bump`](go-module-bump/) | `go get module@version` and `go mod tidy` in every module below that version | Starlark picks the modules (`gomod.outdated`), a three-line `run` step updates them |
| [`npm-package-bump`](npm-package-bump/) | Sets a package's range in every `package.json` and refreshes the lockfile | `npm.set_dependency`, then a short `run` step reads `FLEET_VAR_DIRS` |
| [`python-requirement-bump`](python-requirement-bump/) | Pins a package in `requirements*.txt` and `pyproject.toml` | `python.set_requirement` keeping extras, markers and comments |
| [`cargo-dependency-bump`](cargo-dependency-bump/) | Sets a crate's version in every `Cargo.toml` and refreshes `Cargo.lock` | `cargo.set_dependency` for plain and table forms |
| [`jvm-dependency-bump`](jvm-dependency-bump/) | Moves jackson-databind to 2.17.2 in Maven poms and Gradle builds | `maven.set_dependency` following `${property}` references, `gradle.set_dependency` including `version.ref` |
| [`android-target-sdk`](android-target-sdk/) | Raises compileSdk and targetSdk to 35 in every Android module | `android.set_sdk` on Groovy, Kotlin DSL and catalogs, never lowering |
| [`ruby-rails-bump`](ruby-rails-bump/) | Moves the Rails requirement to `~> 7.2` and re-locks only rails | `ruby.set_gem`, then a one-line `bundle lock` step |
| [`php-laravel-11`](php-laravel-11/) | Moves Laravel apps to Laravel 11 and PHP 8.2, then updates the lock for laravel only | `php.set_require` and `php.set_php`, then a one-line `composer update` |
| [`sbt-scala-bump`](sbt-scala-bump/) | Moves Scala 2.13 builds to 2.13.15 and sbt 1.10.2, keeping cross-built Scala 3 | `sbt.set_scala_version` with `from_version`, and `sbt.set_sbt_version` |
| [`bazel-module-bump`](bazel-module-bump/) | Moves Bazel to 7.4.1 and rules_go and gazelle to new versions | `bazel.set_version` and `bazel.set_dep` on multi-line calls |
| [`swift-package-bump`](swift-package-bump/) | Moves swift-log to 1.6.1 and raises swift-tools-version | `swift.set_package` keeping the kind of requirement |
| [`cocoapods-ios-15`](cocoapods-ios-15/) | Raises iOS apps to iOS 15 and Alamofire 5.9 | `cocoapods.set_platform` (never lowering) and `cocoapods.set_pod` |
| [`elixir-phoenix`](elixir-phoenix/) | Moves Phoenix apps to 1.7.14 on Elixir 1.17 and OTP 27 | `elixir.set_dep`, `elixir.set_elixir` and `runtime.set_version` |
| [`buf-plugins`](buf-plugins/) | Moves Buf remote plugins for Go to current versions | `buf.set_plugin_version` |
| [`node-pnpm-9`](node-pnpm-9/) | Moves pnpm projects to pnpm 9 and turns on engine-strict | `npm.set_package_manager` and `npm.set_npmrc` |
| [`terraform-provider-bump`](terraform-provider-bump/) | Moves the AWS provider to `~> 5.0` and raises `required_version` | `terraform.set_provider_version` on block and one-line entries |
| [`terragrunt-module-ref`](terragrunt-module-ref/) | Moves every `?ref=` of a shared Terraform modules repo to v2.0.0 | `terragrunt.set_source_ref` with a `*` source, `.tf` and `terragrunt.hcl` alike |
| [`ansible-collections`](ansible-collections/) | Sets a collection's version in every `requirements.yml` | `ansible.set_version`, adding a missing `version` |
| [`cmake-minimum`](cmake-minimum/) | Raises `cmake_minimum_required` and moves a FetchContent tag | `cmake.set_minimum_version` keeping `...max` ranges |
| [`node-engines`](node-engines/) | Requires Node 22 in every `package.json`, never lowering a newer requirement | JSON edited as data with key order kept, values handed back to the PR |
| [`docker-base-image`](docker-base-image/) | Moves `FROM node:18` to `node:22`, keeping variants and stages | `docker.bump_base_image`, draft PRs |
| [`compose-image-bump`](compose-image-bump/) | Moves `image: redis:7` to `redis:8` in Compose files, Kubernetes manifests and Dockerfiles | `docker.bump_image` |
| [`add-dependabot`](add-dependabot/) | Writes `.github/dependabot.yml` for the ecosystems a repo uses | detecting what a repo contains with `dependabot.ecosystems` |
| [`dependabot-ecosystems`](dependabot-ecosystems/) | Adds docker and github-actions entries to existing Dependabot configs | `dependabot.add_update` and `dependabot.set_option` |

## Config files

| Campaign | What it changes | Shows |
| --- | --- | --- |
| [`ruff-config`](ruff-config/) | Sets `[tool.ruff]` options in `pyproject.toml`, adding tables where missing | `toml.set` keeping comments |
| [`renovate-preset`](renovate-preset/) | Points every `renovate.json` at a shared preset | `jsonfile.set` keeping key order and indentation |
| [`renovate-automerge`](renovate-automerge/) | Moves off `config:base` and automerges minor and patch devDependency updates | `renovate.remove_preset`, `add_preset` and `add_package_rule` |
| [`golangci-v2`](golangci-v2/) | Migrates golangci-lint configs to v2 and enables gosec | Starlark, the official `migrate` command, then Starlark again |
| [`devcontainer-image`](devcontainer-image/) | Moves Dev Containers to Go 1.23 and adds the GitHub CLI feature | `devcontainer` edits that keep JSONC comments |

## Runtimes

| Campaign | What it changes | Shows |
| --- | --- | --- |
| [`bump-go-version`](bump-go-version/) | Raises the `go` directive and workflow `go-version` pins | `exclude.repos`, `gomod.set_go_version` never lowering a version |
| [`python-version`](python-version/) | Moves `.python-version`, `requires-python` and `setup-python` to 3.12 | three helpers in one step: `runtime`, `python` and `gha` |
| [`make-go-version`](make-go-version/) | Sets `GO_VERSION` in Makefiles and raises `go.mod` to match | `make.set_var` keeping `?=`, `:=` and comments |
| [`node-version`](node-version/) | Pins Node 22 in `.nvmrc`, `.tool-versions` and `setup-node`, creating `.nvmrc` where nothing pins it | `runtime.set_version` with `create`, the version as a variable |
| [`gradle-wrapper-bump`](gradle-wrapper-bump/) | Moves the Gradle wrapper to 8.10.2, with the right checksum for `-bin` or `-all` | `gradle.set_wrapper_version`, never lowering a version |
| [`dotnet-8`](dotnet-8/) | Retargets `net6.0` projects to `net8.0` and pins the 8.0 SDK | `dotnet.set_target_framework` on single and multi-targeted projects |

## CI and security

| Campaign | What it changes | Shows |
| --- | --- | --- |
| [`upgrade-action`](upgrade-action/) | Rewrites `uses: actions/checkout@v3` to `@v4` | exact ref matching (`v3` never matches `v3.5.0`) |
| [`pin-actions`](pin-actions/) | Pins every `uses:` to a commit SHA, keeping the tag as a comment | three steps: Starlark finds refs, `gh api` resolves them, Starlark pins |
| [`workflow-permissions`](workflow-permissions/) | Adds a read-only top-level `permissions:` to workflows without one | line-based YAML edits so comments and layout survive |
| [`pre-commit-rev`](pre-commit-rev/) | Moves a hook repository to a new `rev` in `.pre-commit-config.yaml` | `precommit.set_rev`, the old rev reported in the PR |
| [`runner-label`](runner-label/) | Pins jobs on `ubuntu-latest` to `ubuntu-24.04` | `yamlpath.values` with a `*` wildcard, then `yamlpath.set` |
| [`buildkite-plugin-bump`](buildkite-plugin-bump/) | Moves the docker-compose plugin to v5.4.1 in every Buildkite pipeline | `buildkite.bump_plugin`, reporting the old version |
| [`buildkite-agents`](buildkite-agents/) | Moves steps to a new agent queue and adds `timeout_in_minutes` where missing | `buildkite.set_queue` and `buildkite.set_step_defaults`, including grouped steps |
| [`jenkins-shared-library`](jenkins-shared-library/) | Moves a Jenkins shared library and a CircleCI orb to v3 | `jenkins.bump_library` and `circleci.bump_orb`, both built on `refs.bump` |
| [`gitlab-image-bump`](gitlab-image-bump/) | Moves GitLab CI job and service images from Node 18 to 22 | `gitlab.bump_image` on `image:`, `name:` and `services` forms |
| [`gitlab-templates`](gitlab-templates/) | Moves included templates and CI/CD components to v2, and makes jobs interruptible | `gitlab.set_include_ref`, `gitlab.bump_component`, `gitlab.set_default` |
| [`circleci-orbs`](circleci-orbs/) | Moves the node orb to 5.2.0 in every CircleCI config | `circleci.bump_orb`, reporting the old version |
| [`circleci-machine-image`](circleci-machine-image/) | Moves machine jobs off retired Ubuntu 20.04 images | `circleci.set_machine_image` with a `*` match, leaving Docker images alone |
| [`jenkins-agent-image`](jenkins-agent-image/) | Moves Jenkins agents to Maven 3.9 and `tools` to JDK 21 | `jenkins.bump_image` and `jenkins.set_tool` |
| [`branch-rename`](branch-rename/) | Updates workflow branch filters and `refs/heads/` checks after `master` becomes `main` | `gha.rename_branch` |

## Deployments

| Campaign | What it changes | Shows |
| --- | --- | --- |
| [`helm-image-tag`](helm-image-tag/) | Rolls an image tag out to every Helm chart that deploys it, bumping `appVersion` and the chart version | `helm.set_image_tag`, then reading a value back with `yamlpath.get` |
| [`kustomize-image`](kustomize-image/) | Sets `newTag` for an image in every Kustomize overlay that lists it | `kustomize.set_image` |
| [`argocd-revision`](argocd-revision/) | Points Argo CD Applications at a new revision of the platform manifests | `argocd.set_target_revision` across `source` and `sources` |
| [`flux-chart-bump`](flux-chart-bump/) | Moves a Flux HelmRelease to a new chart version | `flux.set_chart_version` |
| [`k8s-api-migration`](k8s-api-migration/) | Moves manifests off API versions Kubernetes removed | `k8s.migrate_api_versions` on multi-document files |
| [`k8s-labels`](k8s-labels/) | Labels every resource and pod template with the owning team | `k8s.set_label` with a per-repo variable |

## Writing your own

Start from the example closest to your change, and check the
[library reference](https://rossedman.github.io/gh-fleet/reference/starlark-library/) before writing a helper yourself. A few habits
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
