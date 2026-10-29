---
title: How it works
---

# How it works

Every target repo goes through the same short pipeline, and each repo can end in a different state without affecting the others.

<figure class="diagram" markdown>
![A campaign file resolves to target repos. Each repo is cloned, the script runs, and the result is a noop, a failure, or a commit pushed to fleet/name with a pull request opened or updated. Status reads the pull requests back from GitHub.](assets/how-it-works.png)
</figure>

## The pipeline

1. **Resolve targets.** The campaign's `orgs` are listed 100 repos per page, 4 pages at a time (cached for 10 minutes), unless a code `search` has already named the only repos that can match. Cheap metadata filters run first, then `files` checks in batched queries of 50 repos, 4 at a time, split in half whenever GitHub times out. Exclusion always wins.
2. **Look up existing PRs.** For `apply`, gh fleet finds PRs by the head branch (`fleet/<name>`), in batched queries of 50 repos, and falls back to a search for the hidden `<!-- gh-fleet:campaign=<name> -->` marker in the PR body.
3. **Skip what hasn't changed.** For `apply`, a repo the last `plan` or `apply` found nothing to change in, within the last day, is skipped without cloning, as long as it has no campaign PR, its base branch is on the same commit (checked in batched queries of 50 repos) and the change is the same. See [Skipping unchanged repos](#skipping-unchanged-repos).
4. **Prepare a clone.** A shallow, blobless, single-branch clone of the base branch, with the campaign branch checked out. Where the campaign branch is in every target is looked up before the run, 50 repos to a GraphQL query, rather than asked of each repo's git server. If the branch already exists on GitHub it is rebased onto the base; on a conflict it is reset to the base, because the script, not the old branch, is the source of truth.
5. **Run the script.** At the repo root, with the `FLEET_*` environment and a timeout. See [Writing changes](guides/writing-changes.md).
6. **Decide.** No difference from the base is a `noop`. A non-zero exit is `failed`. Otherwise the change is committed, pushed, and the PR is opened or updated.

## Skipping unchanged repos

Most repos in a big campaign need nothing, and running the change in each of them means a clone and a script run. So when `plan` or `apply` finds nothing to change in a repo, gh fleet records the commit the base branch was on and a fingerprint of the change, and the next `apply` skips the repo while all of these hold:

- it was run less than a day ago (a skip doesn't count, so a repo is always rechecked daily);
- it has no campaign PR;
- its base branch is still on the same commit;
- the change is the same: every file in the campaign's directory (hidden directories aside), any script or Starlark file it uses outside it, the repo's variables, topics and properties, and the gh fleet version.

Running `plan` and then `apply` therefore clones only the repos the plan found changes in. A skipped repo shows as `noop` with the detail `unchanged since <time>`.

A change that depends on something outside the repo, such as the latest release of a tool fetched while it runs, can need doing even though nothing above moved. Pass `--recheck` to run every repo. A Starlark file that loads a remote module at a branch rather than a version or commit, or a campaign directory too big to be a campaign (over 2,000 files or 50 MB), turns skipping off.

## Where each repo ends up

| Status | Meaning |
| --- | --- |
| `pr open` | A PR exists and is open. Its detail says `created`, `unchanged` or what was updated. |
| `merged` | The PR was merged. Later runs skip the repo. |
| `closed` | The PR was closed without merging. Later runs leave it alone unless you pass `--reopen`. |
| `noop` | The script produced no difference from the base branch. |
| `changed` | The script made a change but it was not pushed, for example because `limits.maxOpenPRs` was reached. `retry` picks it up. |
| `failed` | The script, git or the API failed. The error is kept and `retry` runs the repo again. |
| `skipped` | The repo matched but you cannot push to it. |
| `pending` | Known to the campaign but not yet processed. |

## Source of truth

Local state under `$XDG_STATE_HOME/gh-fleet/` is a cache plus error history. `status` re-reads every PR from GitHub by branch and marker, so it stays correct if the state file is deleted, and a PR recorded locally that no longer exists on GitHub is reset to pending.
