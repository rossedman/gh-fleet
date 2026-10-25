# Example campaigns

Campaigns for the changes people make most, ready to copy. Each directory
holds the campaign file and what its steps run: a `change.star` for
[Starlark](https://rossedman.github.io/gh-fleet/guides/starlark/) steps, a script under `scripts/` for shell steps,
plus a PR body or source files where needed. Change `orgs: [acme]` to your
organization, then:

```sh
cd examples/sync-files
gh fleet targets
gh fleet plan --repo your-org/some-repo
```

Most examples are a few lines of Starlark on top of the
[built-in library](https://rossedman.github.io/gh-fleet/reference/starlark-library/) (`load("@fleet//gha.star", ...)`): no shell
quoting, no GNU-versus-BSD differences, and no access to anything outside the
clone. Three use shell steps where they need a tool: `pin-actions` calls
`gh api`, `go-module-bump` runs `go get`, and `npm-package-bump` follows its
Starlark step with `npm install --package-lock-only`. Every example is tested
in `examples_test.go`, `library_test.go` and `starlark_test.go`, including that a second run changes nothing, so re-running a
campaign is safe.

## Files and text

| Campaign | What it changes | Shows |
| --- | --- | --- |
| [`sync-files`](sync-files/) | Copies the files under `files/` into every repo, replacing them | `files.sync` and the read-only `campaign` object; edit once, re-apply to update every PR |
| [`find-replace`](find-replace/) | Replaces a literal string in every text file | code search targeting, `env` parameters, `files.replace_all` skipping binaries |
| [`remove-files`](remove-files/) | Deletes retired files and directories | `repo.delete` with paths that cannot leave the repo |
| [`gitignore-entries`](gitignore-entries/) | Appends missing patterns to `.gitignore` | multi-line `env` values, `files.ensure_lines` |
| [`add-codeowners`](add-codeowners/) | Writes `.github/CODEOWNERS` naming each repo's team | per-repo variables from defaults, match rules and a CSV, read as `repo.vars` |

## Dependencies

| Campaign | What it changes | Shows |
| --- | --- | --- |
| [`go-module-bump`](go-module-bump/) | `go get module@version` and `go mod tidy` in every module below that version | a shell step, search plus language targeting, long timeouts |
| [`npm-package-bump`](npm-package-bump/) | Sets a package's range in every `package.json` and refreshes the lockfile | two steps: Starlark edits, then a shell step reads `FLEET_VAR_DIRS` from it |
| [`node-engines`](node-engines/) | Requires Node 22 in every `package.json`, never lowering a newer requirement | JSON edited as data with key order kept, values handed back to the PR |
| [`docker-base-image`](docker-base-image/) | Moves `FROM node:18` to `node:22`, keeping variants and stages | `docker.bump_base_image`, draft PRs |
| [`bump-go-version`](bump-go-version/) | Raises the `go` directive and workflow `go-version` pins | `exclude.repos`, `gomod.set_go_version` never lowering a version |
| [`add-dependabot`](add-dependabot/) | Writes `.github/dependabot.yml` for the ecosystems a repo uses | detecting what a repo contains with `dependabot.ecosystems` |

## CI and security

| Campaign | What it changes | Shows |
| --- | --- | --- |
| [`upgrade-action`](upgrade-action/) | Rewrites `uses: actions/checkout@v3` to `@v4` | exact ref matching (`v3` never matches `v3.5.0`) |
| [`pin-actions`](pin-actions/) | Pins every `uses:` to a commit SHA, keeping the tag as a comment | a shell step calling `gh api`, a body file, `maxOpenPRs`, signed-off commits |
| [`workflow-permissions`](workflow-permissions/) | Adds a read-only top-level `permissions:` to workflows without one | line-based YAML edits so comments and layout survive |

## Writing your own

Start from the example closest to your change. A few habits make
changes safe to run across hundreds of repos:

- **Make it idempotent.** Check before you change, so a second run is a
  no-op. Every `@fleet` helper already does.
- **Take parameters from `env`.** The same script then serves many
  campaigns: `upgrade-action` handles any action, `docker-base-image` any
  image.
- **Prefer Starlark.** Reach for a shell step only when you need a tool such
  as a package manager. Shell scripts must run on macOS's bash 3.2 and BSD
  tools as well as Linux.
- **Never run untrusted code.** Package managers get `--ignore-scripts`, and
  nothing is installed globally.
