---
title: Tracking pull requests
---

# Tracking pull requests

Opening the PRs is half the job. This guide covers re-running a campaign safely, following the PRs to merged, recovering from failures and closing a campaign down.

## Idempotent by design

`apply` finds each repo's existing PR in two ways: by the head branch `fleet/<name>`, and, for any repo not found that way, by a search for the hidden marker gh fleet adds to every PR body:

```html
<!-- gh-fleet:campaign=pin-actions -->
```

What happens next depends on what it finds:

| Existing PR | `apply` does |
| --- | --- |
| none | runs the change and opens a PR |
| open | re-runs the change, pushes if the result differs, and updates the title and body if they changed. Missing labels are added. |
| merged | skips the repo |
| closed without merging | leaves it alone, unless you pass `--reopen` |

Running the same `apply` twice reports `unchanged` the second time. If GitHub refuses to reopen a closed PR, for example because its branch was recreated, gh fleet opens a new one.

When the campaign branch already exists, gh fleet rebases it onto the base branch before running your script. If the rebase conflicts, the branch is reset to the base and the script runs again, since the script is the source of truth.

### When the change has landed

If an open PR's change is already on the base branch, `apply` reports `noop` and says so. Pass `--close-noop` to close such PRs with a comment instead.

```sh
gh fleet apply --close-noop
```

## Following progress

`status` refreshes every target's PR from GitHub and prints one row per repo:

```console
$ gh fleet status -f fleet.yaml
REPO            STATUS   PR                                      CHECKS   REVIEW           NOTE
acme/api        merged   https://github.com/acme/api/pull/1      success  approved
acme/billing    pr open  https://github.com/acme/billing/pull/1  success  approved
acme/docs-site  skipped  -                                       -        -                no push access
acme/gateway    pr open  https://github.com/acme/gateway/pull/1  failure  review required
acme/web        pr open  https://github.com/acme/web/pull/1      pending  review required
acme/worker     noop     -                                       -        -

3 pr open · 1 merged · 1 noop · 1 skipped
API quota: core 4,871/5,000 · graphql 4,962/5,000
```

`CHECKS` is the rollup of the PR's status checks and `REVIEW` is its review decision. `NOTE` carries the last error for a failed repo, or the reason for a skipped, noop or closed one. The remaining API quota is printed at the end. PRs are read in batched GraphQL queries of 50 repos.

For scripts, `--json` prints a report and `--jq` filters it:

```sh
gh fleet status --json --jq '.counts'
```

## Recovering from failures

A failure in one repo never stops the run. The repo is recorded as `failed` with its error, `gh fleet` exits non-zero at the end, and `status` shows the error. Fix the script or the cause, then:

```sh
gh fleet retry
```

`retry` runs `apply` again for only the repos whose last run failed, that never ran, or that changed but were not pushed. Repos with a PR, merged or closed PRs and noops are left alone. It takes the same flags as `apply`.

### Capping open PRs

`limits.maxOpenPRs` stops new PRs once that many campaign PRs are open, counting the ones already open. Repos held back show `changed` with `maxOpenPRs reached; not pushed`. As PRs merge, `retry` opens the next ones, which makes it a simple way to roll out in waves.

## Closing a campaign

```sh
gh fleet close --comment "Superseded by #42"
```

`close` closes every open PR the campaign opened and deletes their branches. It leaves merged and already-closed PRs alone and asks for confirmation unless you pass `--yes`. Use `--keep-branches` to close the PRs but keep the branches, and `--repo` to close only some.

## Local state

Local state is a cache plus error history. GitHub is the source of truth for pull requests.

| What | Where |
| --- | --- |
| Per-campaign state | `$XDG_STATE_HOME/gh-fleet/<name>.json` |
| Script logs, one per repo | `$XDG_STATE_HOME/gh-fleet/<name>/logs/<owner>__<repo>.log` |

`XDG_STATE_HOME` falls back to `~/.local/state`. The state file records, for each repo, its status, PR number and URL, checks, review, and last error. Diffs and command output are never stored.

`gh fleet list` shows every campaign with state and where its file lives. A name from that list can stand in for `--file` in any command:

```console
$ gh fleet list
CAMPAIGN     UPDATED              REPOS                                      FILE
pin-actions  2026-10-25 17:54:33  3 pr open · 1 merged · 1 noop · 1 skipped  fleet.yaml
```

**You can delete the state file and `status` still works**, because it finds PRs by branch and marker. Only the error history is lost.
