---
title: Exploring with exec
---

# Exploring with exec

`gh fleet exec` runs a read-only command in a fresh clone of every target and tallies the output lines. It is how you find out what a campaign should change before you write it.

```console
$ gh fleet exec --org acme --file .github/workflows -- grep -ho uses: [^@]* .github/workflows/ci.yml
failed   acme/docs-site exited with status 2: grep: .github/workflows/ci.yml: No such file or directory
REPOS  OUTPUT
6      uses: actions/checkout
5      uses: actions/setup-go

6 done · 1 failed
```

Each distinct output line is counted once per repo, most common first. Nothing is pushed and no PRs are opened.

## Choosing targets

With any ad-hoc flag (`--org`, `--topic`, `--language`, `--file`, `--search`, `--repo`, `--repo-file`) exec builds its targets from those flags. Otherwise it uses the campaign (`--campaign`, default `./fleet.yaml`). The ad-hoc flags use the same rules as [campaign targeting](targeting.md), with one difference: repos you can read but not push to are included, as the command only reads. `--exclude` and `--exclude-file` leave repos out either way.

```sh
gh fleet exec --org acme --file Dockerfile -- grep -h '^FROM' Dockerfile
gh fleet exec --org acme --language Go -- sh -c "grep '^go ' go.mod"
```

## The command

The command runs directly, **without a shell**. Wrap it in `sh -c` when you need pipes or redirects. It runs in the repo root of a shallow clone of the default branch, with the `FLEET_*` variables set (`FLEET_DRY_RUN` is `true`).

## Output

- `--raw` prints each repo's output, prefixed with the repo name, instead of the tally.
- `--json` prints every repo's result and the tally; `--jq` filters it.
- `--timeout` limits each repo (default 5 minutes), and `--concurrency` sets the parallelism (default 8).

A repo where the command exits non-zero is reported as failed, and `exec` exits non-zero at the end.
