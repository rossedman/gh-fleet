---
title: Getting started
---

# Getting started

Install the extension, scaffold a campaign and open your first pull requests across an organization, with a preview step before anything is pushed.

## Install

gh fleet is a [GitHub CLI](https://cli.github.com/) extension.

```sh
gh extension install rossedman/gh-fleet
gh fleet version
```

Upgrade later with `gh extension upgrade fleet`.

### What you need

- The GitHub CLI, signed in with `gh auth login`. gh fleet uses gh's authentication for the API and never asks for a token of its own.
- `git` on your `PATH`. Clones, commits and pushes shell out to your git, so your credential helpers, commit signing and proxy settings apply.
- Push access to the repositories you want to change. Repos you can read but not push to are listed as **skipped** and left alone.

!!! note "One-time git credential setup"
    Clones and pushes use your own `git`, not the GitHub API. If git has no GitHub credentials yet, run `gh auth setup-git` once so it can use gh's token.

## Your first campaign

A campaign is one YAML file. Scaffold one with every option shown and commented:

```sh
gh fleet init pin-actions --org acme
```

This writes `fleet.yaml` in the current directory, with a `yaml-language-server` header so editors validate and complete it. The name defaults to the directory's name, and `--file` writes it elsewhere.

Or start from one of the [80 tested examples](examples.md), with its scripts and test cases:

```sh
gh fleet init --list
gh fleet init node-22 --from node-version --org acme
gh fleet test        # the copied test cases pass before you change anything
```

Open it and edit the targets and the change. The examples in this guide use the [`pin-actions` example](examples.md), which pins every `uses:` in `.github/workflows` to a commit SHA:

```yaml title="fleet.yaml"
apiVersion: fleet.gh/v1
name: pin-actions
description: Pin every third-party GitHub Action to a full commit SHA

targets:
  orgs: [acme]
  include:
    files: [.github/workflows]

change:
  steps:
    - name: find refs
      starlark: ./change.star    # load("@fleet//gha.star", "unpinned")
    - name: resolve              # shell only where a tool is needed
      run: |
        set -e
        shas=""
        for ref in ${FLEET_VAR_REFS//,/ }; do
          shas="$shas $ref=$(gh api "repos/${ref%@*}/commits/${ref#*@}" --jq .sha)"
        done
        echo "shas=$shas" >> "$FLEET_OUTPUT/vars"
    - name: pin
      starlark: ./pin.star       # load("@fleet//gha.star", "pin")

pr:
  title: "ci: pin GitHub Actions to commit SHAs"
  labels: [fleet, security]
```

The [configuration page](configuration.md) walks through every section.

## 1. See what it will touch

```console
$ gh fleet targets -f fleet.yaml
REPO          MATCHED BY
acme/api      file:.github/workflows
acme/billing  file:.github/workflows
acme/gateway  file:.github/workflows
acme/web      file:.github/workflows
acme/worker   file:.github/workflows

SKIPPED         REASON
acme/docs-site  no push access

5 repos (1 skipped)
```

Every matched repo shows the filter that matched it. Nothing is cloned yet.

!!! tip "You can leave off `-f`"
    Commands read `./fleet.yaml` by default. You can also pass a campaign name from `gh fleet list`, or a path.

## 2. Preview the change

`plan` clones each target, runs your steps, and prints the diff each repo would get. Nothing is pushed and nothing is written to GitHub.

```console
$ gh fleet plan -f fleet.yaml --repo acme/api
changed  acme/api
── acme/api
diff --git a/.github/workflows/ci.yml b/.github/workflows/ci.yml
index e9179e8..4b6957d 100644
--- a/.github/workflows/ci.yml
+++ b/.github/workflows/ci.yml
@@ -1,5 +1,5 @@
 jobs:
   build:
     steps:
-      - uses: actions/checkout@v4
-      - uses: actions/setup-go@v5
+      - uses: actions/checkout@08eba0b27e820071cde6df949e0beb9ba4906955 # v4
+      - uses: actions/setup-go@0aaccfd150d50ccaeb58ebd88d36e91967a5f35b # v5

1 changed · 1 skipped
```

Start with `--repo` on one repo, then drop it to preview all of them. A repo where the steps change nothing is reported as `noop`. Across many repos, each distinct diff is printed once, under the repos that get it (`── 212 repos: acme/api, acme/billing, … and 202 more`), so the same change in 300 repos is read once. `--ungrouped` prints every repo's diff instead.

!!! tip "Test without GitHub"
    `gh fleet test --dir ../api` runs the change on a clone you already have. Once it works, keep sample repos in `tests/` next to `fleet.yaml` and `gh fleet test` checks them on every edit. See [Testing a campaign](guides/testing.md).

## 3. Open the pull requests

```console
$ gh fleet apply -f fleet.yaml --yes
pr open  acme/api https://github.com/acme/api/pull/1 created
pr open  acme/billing https://github.com/acme/billing/pull/1 created
pr open  acme/gateway https://github.com/acme/gateway/pull/1 created
pr open  acme/web https://github.com/acme/web/pull/1 created
noop     acme/worker
4 pr open · 1 noop · 1 skipped
```

Without `--yes`, `apply` asks for confirmation before it pushes anything, and in a non-interactive shell it refuses to run rather than hang.

Run the same command again and nothing is duplicated:

```console
$ gh fleet apply -f fleet.yaml --yes
pr open  acme/api https://github.com/acme/api/pull/1 unchanged
...
```

## 4. Follow them to merged

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

Fix a script, or change the campaign, and run `apply` again to update every open PR.

When PRs are approved and green, `merge` merges them and says what the rest wait on. `--auto` hands the ones still running checks to GitHub's auto-merge:

```console
$ gh fleet merge --auto
```

See [Tracking pull requests](guides/tracking-prs.md) for `merge`, `retry`, `close` and what the local state holds.

## Where next

- [How it works](how-it-works.md): the path each repo takes.
- [Writing changes](guides/writing-changes.md): how steps run, and the shell contract.
- [Starlark library](reference/starlark-library.md): every built-in helper. `gh fleet lib` prints the same docs in the terminal.
- [Targeting repositories](guides/targeting.md): topics, languages, files and code search.
- [Examples](examples.md): eighty tested campaigns for common changes.
