---
title: Performance
---

# Performance

A run spends its time in three places: finding the targets on GitHub, getting each repo onto disk and running the change in it, and writing branches and PRs back. This page explains what each costs, how gh fleet keeps that low, and what to change when a run is slow. The details live in the pages linked from each section.

## Finding out what's slow

Start here rather than guessing. `--debug` (or `GH_FLEET_DEBUG=1`) logs each step on stderr as it starts and finishes, and ends with a table of how long everything took:

```sh
gh fleet plan --debug 2> plan-debug.log
```

The end of the log looks like this (an example run over 120 repos):

```text
Timings (totals add up across parallel workers):
NAME                      COUNT  TOTAL  AVG    MAX
total: plan               1      41s    41s    41s
run change in every repo  1      36s    36s    36s
repo: total               120    9m12s  4.6s   31s
repo: clone               120    3m40s  1.8s   12s
git fetch                 120    1m2s   517ms  9s
...
```

Repos run in parallel, so per-repo, git and API rows add up across workers and can be far larger than the wall time. Compare the `AVG` and `MAX` columns, and `run change in every repo` against `total`.

| Row | What it measures | When it dominates |
| --- | --- | --- |
| `resolve targets` | finding the targets: org listings, searches, lookups, file checks | see [Targeting cheaply](#targeting-cheaply); the progress note says why an org was listed |
| `pick wave`, `index existing PRs` | looking up the campaign's PRs, in batched queries | grows with the number of targets; `--repo` limits it |
| `look up campaign branches` | where the campaign branch is in every target, 50 repos to a query | should stay small |
| `skip unchanged repos` | checking which repos the last run found nothing to change in | should stay small |
| `repo: clone` | getting one repo onto disk, from the [git cache](#the-git-cache) or GitHub | big repos; the first run, which fills the cache; `--no-git-cache` |
| `git clone` | a repo cloned straight from GitHub | appears when the cache is off or couldn't be used, which a note on stderr says |
| `git fetch` | bringing a repo's cache up to date, or fetching an existing campaign branch's history to rebase it | a slow network, or many existing campaign branches |
| `repo: step <name>` | your change script, per step | the script itself: `npm install` and the like |
| `repo: diff`, `repo: push`, `repo: sync PR` | the diff, the push and the PR's API calls | `sync PR` waits on the [write rate](#writes) |
| `cleanup (background)` | deleting a finished clone, off the worker's critical path | never blocks other repos; only the end of the run waits for it |
| `api core`, `api graphql`, `api search`, `api code_search` | each GitHub request, by API, with its method, path and status in the log | many requests: a big org listing or file checks |
| `api wait: rate limits` | time requests spent waiting on GitHub's limits or the write rate | see [Rate limits](rate-limits.md); a large value means GitHub, not gh fleet, is the bottleneck |
| `api wait: retry backoff` | backoff before retrying a `5xx` | GitHub having trouble |

## What a run costs

| Step | Cost | What keeps it down |
| --- | --- | --- |
| Find targets | API reads | caches, searches and batched lookups: see [Targeting cheaply](#targeting-cheaply) |
| Skip unchanged repos (`apply`) | one query per 50 repos | repos the last run found nothing to change in aren't cloned at all |
| Look up campaign branches | one query per 50 repos | replaces a round trip to the git server per repo |
| Get each repo onto disk | git traffic, no API quota | the [git cache](#the-git-cache): only what changed since the last run is fetched |
| Run the change | your script's time | `--concurrency` runs more repos at once |
| Push and open or update PRs (`apply`) | git push, plus a few API writes per new PR (create it, then labels and reviewers when the campaign has them), and none for a PR that's already right | GitHub's limits set the pace: see [Writes](#writes) |

## Three caches

gh fleet keeps three caches. Together they make the usual loop of `plan`, `plan` again after editing the script, then `apply` much cheaper than three full runs.

| Cache | Kept for | Saves | Turn off or refresh |
| --- | --- | --- | --- |
| Org listings | 10 minutes | listing every repo of an org again; `targets`, `plan` and `apply` in a row cost one listing | wait 10 minutes, or delete `gh-fleet/repos/` under your user cache directory |
| [Git cache](#the-git-cache) | until deleted; garbage collected weekly | downloading each repo again: later runs fetch only what changed | `--no-git-cache`, or delete `gh-fleet/git/` |
| [Unchanged repos](../how-it-works.md#skipping-unchanged-repos) | a day | `apply` skips repos the last run found nothing to change in, without cloning them | `--recheck` |

### The git cache

Every repo a run touches gets a shallow copy of its base branch under `gh-fleet/git/` in your user cache directory. A run brings that copy up to date with a fetch, which transfers only the commits since the last run, and the repo's working copy borrows its files instead of downloading them. Each working copy still has its own `.git` directory with `origin` pointing at GitHub, so your scripts, pushes and rebases behave as they would in a plain clone, and nothing a script writes reaches the cache.

The first run on a repo fills its cache, so it costs about what a clone did. After that, getting a repo onto disk takes a fraction of a full clone, and the bigger the repo, the bigger the difference. If the cache can't be used, for example when two runs fetch the same repo at the same moment, that repo is cloned from GitHub and a note says so; the cache never fails a run.

The cache holds about one shallow copy of each repo you've targeted. Deleting it is safe: the next run rebuilds it.

## Targeting cheaply

How `targets` are written decides how much of an org gets read. The cheapest form names the repos:

| Targets | What gh fleet reads |
| --- | --- |
| `include.repos` alone, with exact `owner/name` or bare names | only those repos, 50 to a query, however big the org |
| `include.names` with exact names | only those repos in each org, then the other filters on them |
| `include.topics` or `include.languages` | GitHub's repo search for the matches, without listing the org; it lists instead when a listing is cached, the search matches over 1,000 repos, or the campaign uses custom properties |
| `include.search` | code search, then only the repos with hits |
| `include.files`, `include.properties`, an `include.names` pattern, or no include filter | every repo of each org, cached for 10 minutes |
| `--repo` or `--repo-file` | only the repos named, whatever else the campaign's targets would read; its filters still apply to them |

`include.repos` beside any other include filter adds its repos to what that filter selects, so the org is still read; `targets`, `plan` and `apply` print a note when a campaign does this. When an org is listed, the progress note says which filter caused it, for example `Listing all of acme's repos: include.files is checked in every repo`. See [Targeting repositories](targeting.md) for the full rules.

## Concurrency

`--concurrency` (or `limits.concurrency`, default 16) is how many repos are cloned and run at once. Getting a repo onto disk goes through git and costs no API quota, and GitHub API requests are capped at 8 in flight however many workers there are. So raising `--concurrency` speeds up cloning and scripts without adding API load.

Raise it when `repo: clone` or your script steps dominate and your machine has cores, memory and disk to spare. Lower it when the change script is heavy, such as a full dependency install, and the machine is struggling.

## Writes

Writes are paced at 1 per second across all workers by default, which is GitHub's own advice for mutating requests, and new PRs are also kept within GitHub's limit of 80 a minute and 500 an hour. gh fleet already sends as few writes as it can: labels and reviewers only when the campaign has them, and an existing PR only when its title, body or labels differ, so re-running `apply` on repos whose PRs are already right writes nothing.

When `apply` opens thousands of PRs, GitHub's limits set the pace, not gh fleet. [Waves](tracking-prs.md#rolling-out-in-waves) and `retry` let such a rollout span several sessions. See [Rate limits](rate-limits.md) for the details, and why raising `--write-rate` rarely helps.

## Checklist for a slow run

1. Run it with `--debug` and look at the table's largest `AVG` and `MAX` rows.
2. If `resolve targets` is large, check the progress note for why an org was listed, and see [Targeting cheaply](#targeting-cheaply).
3. If `repo: clone` is large, run again: the first run fills the git cache. If `git clone` appears, read the note on stderr for why the cache wasn't used.
4. If your script's steps dominate, make the script faster or raise `--concurrency`.
5. If `api wait: rate limits` is large, GitHub is pacing the run; see [Rate limits](rate-limits.md).
6. To iterate on one repo, use `--repo owner/name`: everything above shrinks to that repo.
