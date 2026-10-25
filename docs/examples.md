---
title: Examples
---

# Examples

Campaigns for the changes people make most, ready to copy from [`examples/`](https://github.com/rossedman/gh-fleet/tree/main/examples). Each directory
holds the campaign file and the script it runs (plus a PR body or source
files where needed). Change `orgs: [acme]` to your organization, then:

```sh
cd examples/sync-files
gh fleet targets
gh fleet plan --repo your-org/some-repo
```

Every script is tested in [`examples_test.go`](https://github.com/rossedman/gh-fleet/blob/main/examples/examples_test.go) and [`library_test.go`](https://github.com/rossedman/gh-fleet/blob/main/examples/library_test.go), including
that a second run changes nothing, so re-running a campaign is safe. They run
on the bash that ships with macOS (3.2) as well as Linux.

## Files and text

| Campaign | What it changes | Shows |
| --- | --- | --- |
| [`sync-files`](https://github.com/rossedman/gh-fleet/tree/main/examples/sync-files) | Copies the files under `files/` into every repo, overwriting them | shipping files with a campaign; edit once, re-apply to update every PR |
| [`find-replace`](https://github.com/rossedman/gh-fleet/tree/main/examples/find-replace) | Replaces a literal string in every tracked text file | code search targeting, `env` parameters, skipping binaries |
| [`remove-files`](https://github.com/rossedman/gh-fleet/tree/main/examples/remove-files) | Deletes retired files and directories | `files` targeting, a refusal for paths outside the repo |
| [`gitignore-entries`](https://github.com/rossedman/gh-fleet/tree/main/examples/gitignore-entries) | Appends missing patterns to `.gitignore` | multi-line `env` values, idempotent appends |
| [`add-codeowners`](https://github.com/rossedman/gh-fleet/tree/main/examples/add-codeowners) | Writes `.github/CODEOWNERS` naming each repo's team | per-repo variables from defaults, match rules and a CSV, templated reviewers and labels |

## Dependencies

| Campaign | What it changes | Shows |
| --- | --- | --- |
| [`go-module-bump`](https://github.com/rossedman/gh-fleet/tree/main/examples/go-module-bump) | `go get module@version` and `go mod tidy` in every module below that version | search plus language targeting, multi-module repos, long timeouts |
| [`npm-package-bump`](https://github.com/rossedman/gh-fleet/tree/main/examples/npm-package-bump) | Sets a package's range in every `package.json` and refreshes the lockfile | lockfile-only installs with scripts disabled |
| [`docker-base-image`](https://github.com/rossedman/gh-fleet/tree/main/examples/docker-base-image) | Moves `FROM node:18` to `node:22`, keeping variants and stages | regex rewrites that stay inside Dockerfiles, draft PRs |
| [`bump-go-version`](https://github.com/rossedman/gh-fleet/tree/main/examples/bump-go-version) | Raises the `go` directive and workflow `go-version` pins | `exclude.repos`, never lowering a version |
| [`add-dependabot`](https://github.com/rossedman/gh-fleet/tree/main/examples/add-dependabot) | Writes `.github/dependabot.yml` for the ecosystems a repo uses | detecting what a repo contains |

## CI and security

| Campaign | What it changes | Shows |
| --- | --- | --- |
| [`upgrade-action`](https://github.com/rossedman/gh-fleet/tree/main/examples/upgrade-action) | Rewrites `uses: actions/checkout@v3` to `@v4` | exact ref matching (`v3` never matches `v3.5.0`) |
| [`pin-actions`](https://github.com/rossedman/gh-fleet/tree/main/examples/pin-actions) | Pins every `uses:` to a commit SHA, keeping the tag as a comment | calling `gh api` from a script, a body file, `maxOpenPRs`, signed-off commits |
| [`workflow-permissions`](https://github.com/rossedman/gh-fleet/tree/main/examples/workflow-permissions) | Adds a read-only top-level `permissions:` to workflows without one | editing YAML with awk so comments and layout survive |

## Writing your own

The [writing changes guide](guides/writing-changes.md) covers the full script contract.

A change script runs at the repo root of a fresh clone. Exit 0 after changing
files and gh fleet commits and opens a PR; exit 0 without changes and the repo
is a no-op; exit non-zero and the repo is marked failed with the last lines of
output. The scripts here follow a few habits worth copying:

- **Make it idempotent.** Check before you change (`grep -q` before
  appending, compare versions before bumping), so a second run is a no-op.
- **Take parameters from `env`.** The same script then serves many campaigns:
  `upgrade-action` handles any action, `docker-base-image` any image.
- **Prefer literal matches.** `perl -pi -e 's/\Q$ENV{FROM}\E/.../'` and
  `grep -F` need no escaping; `sed` patterns built from variables break on
  dots and slashes.
- **Stay portable.** No `declare -A`, `mapfile` or GNU-only `sed -i`; macOS
  ships bash 3.2 and BSD tools. Use `sed -i.bak` and remove the backup.
- **Never run untrusted code.** Package managers get `--ignore-scripts`, and
  nothing is installed globally.
