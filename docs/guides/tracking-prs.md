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

### Sharing progress

`--format markdown` prints a report for people: how many repos are done, what needs attention (failed runs, failing checks, conflicts, requested changes), the open PRs with what each waits on, using the same reasons as `merge`, and the merged ones. It renders in an issue, a PR, a wiki page or chat:

```sh
gh fleet status --format markdown | gh issue comment 42 --body-file -
```

```markdown
# rails-7-2

Move every Rails app to 7.2

**2 of 7 repos done (28%)** `██░░░░░░░░`

1 merged · 4 pr open · 1 failed · 1 noop · 1 skipped

## Needs attention

| Repo | PR | Problem |
| --- | --- | --- |
| acme/d | [#4](https://github.com/acme/d/pull/4) | checks failing |
| acme/e | [#5](https://github.com/acme/e/pull/5) | merge conflict |
| acme/f | - | step lock: exit 1 |

## Open

| Repo | PR | Checks | Review | Waiting on |
| --- | --- | --- | --- | --- |
| acme/b | [#2](https://github.com/acme/b/pull/2) | success | approved | ready to merge |
| acme/c | [#3](https://github.com/acme/c/pull/3) | pending | review required | checks running |
```

`--format html` prints the same report as one self-contained page, with light and dark themes, to publish or attach:

```sh
gh fleet status --format html > rails-7-2.html
```

"Done" counts merged PRs and repos that needed no change. Skipped repos aren't counted.

## Recovering from failures

A failure in one repo never stops the run. The repo is recorded as `failed` with its error, `gh fleet` exits non-zero at the end, and `status` shows the error. Fix the script or the cause, then:

```sh
gh fleet retry
```

`retry` runs `apply` again for only the repos whose last run failed, that never ran, or that changed but were not pushed. Repos with a PR, merged or closed PRs and noops are left alone. It takes the same flags as `apply`.

### Capping open PRs

`limits.maxOpenPRs` stops new PRs once that many campaign PRs are open, counting the ones already open. Repos held back show `changed` with `maxOpenPRs reached; not pushed`. As PRs merge, `retry` opens the next ones, which makes it a simple way to roll out in waves.

## Merging

`merge` merges the campaign's open PRs that are ready, and says what each of the others waits on:

```console
$ gh fleet merge --dry-run
pr open  acme/web https://github.com/acme/web/pull/1 waiting: checks running
pr open  acme/api https://github.com/acme/api/pull/1 would merge
1 would merge · 1 waiting
$ gh fleet merge --auto --yes
pr open  acme/web https://github.com/acme/web/pull/1 auto-merge on
merged   acme/api https://github.com/acme/api/pull/1
1 merged · 1 set to auto-merge
```

A PR is merged when GitHub reports it mergeable, its checks pass and it is approved. By default both the approval and the checks are needed even in repos whose branch protection doesn't require them, so a campaign never merges something nobody looked at. `--allow-unreviewed` and `--allow-no-checks` relax that for repos you trust. The merge names the head commit gh fleet checked, so a PR that gets a new push in the meantime waits for the next run.

The others wait, with the reason:

| Reason | What to do |
| --- | --- |
| `checks running`, `needs review` | Wait, or pass `--auto` so GitHub merges it when they pass. |
| `checks failing` | Fix the change and run `apply` again. |
| `merge conflict` | Run `apply` again: it rebases the branch on the base. |
| `behind its base` | Pass `--update-branch` to merge the base into the branch. |
| `changes requested`, `draft` | Up to the repo's owners. |

Useful flags:

- `--method merge|squash|rebase`, default `squash`.
- `--auto` turns on GitHub's auto-merge for PRs that aren't ready yet. Auto-merge only waits for what branch protection requires, so it isn't used where the repo requires no review or no checks, unless you pass the `--allow-` flags.
- `--max 10` merges at most ten PRs per run, to roll out in batches. It picks the same PRs whatever order the API answers in.
- `--keep-branches` keeps the branches of merged PRs. By default they are deleted, as `close` does.
- `--dry-run` writes nothing, and `--json` prints one result per PR with its action and reason.

It asks for confirmation unless you pass `--yes`.

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
