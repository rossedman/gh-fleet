---
name: steward
description: How to drive a gh-fleet pull request to mergeable - what each CI check runs and how to reproduce it locally, which failures are infrastructure rather than the PR, and the repo's rules for commits, docs and tests. Use when fixing CI or answering review on a gh-fleet PR.
---

# Stewarding a gh-fleet pull request

A PR here is done when every required check is green on the current head,
there is no merge conflict, and a code owner has approved. PRs are
squash-merged, so the branch's own history doesn't matter; the PR title
becomes the commit on `main`.

## Reproduce a failure before fixing it

Every CI job maps to a make target. Run the one that failed, watch it fail,
fix, and run it again before pushing.

| Check | Job | Run locally |
| --- | --- | --- |
| `ci / lint` | `go mod tidy` is clean | `go mod tidy && git diff --exit-code -- go.mod go.sum` |
| | golangci-lint, check-cmds, doccheck | `make lint` |
| | generated docs are current | `make docs-check` (fix with `make docs` and commit the result) |
| `ci / test (ubuntu-latest)` | race tests, 90% coverage of hand-written code | `make cover` |
| | benchmark smoke run | `make bench-smoke` |
| `ci / test (macos-latest)`, `ci / test-windows` | plain tests on other OSes | `make test`; read the log for path, line-ending or process differences, since you can't run those OSes here |
| `ci / fuzz (<target>)` | 30s per fuzz target | `go test -run='^$' -fuzz='^FuzzX$' -fuzztime=30s ./pkg`; keep a failing input under `testdata/fuzz/` as a regression test |
| `ci / vuln` | govulncheck | `make vuln` |
| `ci / release-check` | goreleaser check and snapshot | `make release-check` |
| `guardrails / gate` | build, race tests, 90% coverage (90% of changed lines), lint, govulncheck on touched packages | `guardrails gate --base origin/main --explain`, as CI runs it, with the release `guardrails.yml` pins and golangci-lint and govulncheck built by `GOTOOLCHAIN=$(go env GOVERSION) go install` first on `PATH`. `make cover lint vuln` does **not** check coverage of changed lines, the gate's most common failure; the gate's report names the uncovered `file:line`s |
| `Docs / build` | `go run ./tools/gendocs`, then `mkdocs build --strict` | `make docs-build` (needs `uv`) |

Tooling notes:

- golangci-lint and govulncheck must be built with this module's Go version;
  a released or preinstalled binary refuses to load the module with "the Go
  language version ... is lower than the targeted Go version". The Makefile
  already does this. To run golangci-lint directly:
  `GOTOOLCHAIN="$(go env GOVERSION)" go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0 run ./...`
- `make lint` needs `shellcheck`. If it's missing, use
  `make lint SHELLCHECK="uvx --from shellcheck-py shellcheck"`. Otherwise run
  the remaining steps yourself (`tools/check-cmds.sh`,
  `go run ./tools/doccheck ./...`) and say that shellcheck was skipped.
- `Docs` only runs when a PR touches `docs/`, `cmd/`, `tools/`, `mkdocs.yml`,
  the campaign schema, `CONTRIBUTING.md` or `SECURITY.md`. Any new or changed
  flag in `cmd/` changes `docs/reference/`, so run `make docs` and commit the
  result whenever `cmd/` changes.

## Failures that aren't the PR's

Check for these before assuming the change broke something. When one
matches, re-run the failed job once and leave one comment on the PR naming
the check and the cause. If the re-run fails again, treat it as a real
failure.

- **Module downloads.** Errors from `proxy.golang.org` while fetching
  modules (`stream error ... INTERNAL_ERROR`, `unexpected EOF`, TLS or 5xx on
  a `.zip` or `.mod`), before any build or test step ran. The Docs job runs
  with `cache: false`, so it downloads every module and hits this most often.
- **Sigstore during release.** Signing timing out against Sigstore's
  transparency log; `tools/sign.sh` already retries it. Release doesn't run
  on PRs, and `ci / release-check` skips signing.
- **sarifix** is disabled (`vars.SARIFIX_ENABLED`), so a skipped sarifix run
  is expected and blocks nothing.

Anything else in code the PR touches is the PR's to fix. A failure on `main`
as well is not; check the latest `main` run of the same workflow before
deciding.

## Rules for changes

- **Never** edit `.guardrails.yaml`, `.golangci.yml`, `go.sum` by hand, or
  generated files; add `//nolint`; or `t.Skip` a test to get green.
  `.claude/settings.json` denies force pushes, `git reset --hard` and
  `git commit --no-verify`. Bring a base branch in by merging, not rebasing.
- **Commits and PR titles** follow Conventional Commits (`feat:`, `fix:`,
  `perf:`, `docs:`, `ci:`, `build(deps):`, optionally scoped like
  `fix(target):`). The release changelog groups by type, so pick the one
  that fits.
- **Tests:** table tests with `t.Run` and `t.Parallel()`. GitHub access only
  through the fake in `internal/ghapi/fake`, and git only against local bare
  repos from `internal/testutil`. No test talks to the real GitHub API. A new
  GraphQL query is checked against the schema in `internal/ghapi/testdata`,
  so a schema failure means the query is wrong, not the schema.
- **Coverage:** guardrails also requires 90% of changed lines to be covered.
  When it lists uncovered `file:line`s, add a test that asserts behavior on
  those lines rather than one that only runs them.
- **Doc comments** on every package, function and method, starting with its
  name (`go run ./tools/doccheck ./...`).
- **cmd/** holds one cobra command per file, with flag parsing only; logic
  belongs in `internal/app`.
- **Critical paths** (the table in `CONTRIBUTING.md`) need a benchmark, and a
  fuzz target where they take outside input. A reviewer asking for one on
  such a path is asking for a small, local change.
- **Output** goes through `internal/ui` and `internal/render`: results on
  stdout, everything else on stderr, nothing styled when piped or with
  `--json`.

## Before pushing a fix

Run the target that failed, then `make test` and `make lint` for anything
touching Go, and `make docs-check` for anything touching `cmd/` or `docs/`.
Re-read the diff for anything the table above would reject. Push once,
after everything passes.
