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

## Dependencies

| Campaign | What it changes | Shows |
| --- | --- | --- |
| [`go-module-bump`](go-module-bump/) | `go get module@version` and `go mod tidy` in every module below that version | Starlark picks the modules (`gomod.outdated`), a three-line `run` step updates them |
| [`npm-package-bump`](npm-package-bump/) | Sets a package's range in every `package.json` and refreshes the lockfile | `npm.set_dependency`, then a short `run` step reads `FLEET_VAR_DIRS` |
| [`python-requirement-bump`](python-requirement-bump/) | Pins a package in `requirements*.txt` and `pyproject.toml` | `python.set_requirement` keeping extras, markers and comments |
| [`cargo-dependency-bump`](cargo-dependency-bump/) | Sets a crate's version in every `Cargo.toml` and refreshes `Cargo.lock` | `cargo.set_dependency` for plain and table forms |
| [`terraform-provider-bump`](terraform-provider-bump/) | Moves the AWS provider to `~> 5.0` and raises `required_version` | `terraform.set_provider_version` on block and one-line entries |
| [`node-engines`](node-engines/) | Requires Node 22 in every `package.json`, never lowering a newer requirement | JSON edited as data with key order kept, values handed back to the PR |
| [`docker-base-image`](docker-base-image/) | Moves `FROM node:18` to `node:22`, keeping variants and stages | `docker.bump_base_image`, draft PRs |
| [`compose-image-bump`](compose-image-bump/) | Moves `image: redis:7` to `redis:8` in Compose files, Kubernetes manifests and Dockerfiles | `docker.bump_image` |
| [`add-dependabot`](add-dependabot/) | Writes `.github/dependabot.yml` for the ecosystems a repo uses | detecting what a repo contains with `dependabot.ecosystems` |

## Runtimes

| Campaign | What it changes | Shows |
| --- | --- | --- |
| [`bump-go-version`](bump-go-version/) | Raises the `go` directive and workflow `go-version` pins | `exclude.repos`, `gomod.set_go_version` never lowering a version |
| [`python-version`](python-version/) | Moves `.python-version`, `requires-python` and `setup-python` to 3.12 | three helpers in one step: `runtime`, `python` and `gha` |
| [`node-version`](node-version/) | Pins Node 22 in `.nvmrc`, `.tool-versions` and `setup-node`, creating `.nvmrc` where nothing pins it | `runtime.set_version` with `create`, the version as a variable |

## CI and security

| Campaign | What it changes | Shows |
| --- | --- | --- |
| [`upgrade-action`](upgrade-action/) | Rewrites `uses: actions/checkout@v3` to `@v4` | exact ref matching (`v3` never matches `v3.5.0`) |
| [`pin-actions`](pin-actions/) | Pins every `uses:` to a commit SHA, keeping the tag as a comment | three steps: Starlark finds refs, `gh api` resolves them, Starlark pins |
| [`workflow-permissions`](workflow-permissions/) | Adds a read-only top-level `permissions:` to workflows without one | line-based YAML edits so comments and layout survive |
| [`pre-commit-rev`](pre-commit-rev/) | Moves a hook repository to a new `rev` in `.pre-commit-config.yaml` | `precommit.set_rev`, the old rev reported in the PR |

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
