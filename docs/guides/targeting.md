---
title: Targeting repositories
---

# Targeting repositories

The `targets` section decides which repositories a campaign touches. This guide covers every filter, how they combine, and how to check the result before anything runs.

## Where candidates come from

`orgs` lists organizations (or users) whose repositories are candidates. Each org is listed with one paginated GraphQL query, and the listing is cached for 10 minutes, so running `targets`, `plan` and `apply` in a row costs one listing. `include.repos` adds individual `owner/name` repos, even from other owners.

A campaign needs `orgs`, `include.repos`, or both.

## Include filters

```yaml
targets:
  orgs: [acme]
  include:
    topics: [service]
    languages: [Go]
    files: [go.mod]
```

| Filter | A repo matches when |
| --- | --- |
| `topics` | it has any of these topics |
| `languages` | its primary language is any of these (case-insensitive) |
| `files` | any of these paths, files or directories, exists on the default branch |
| `search` | a code search query has a hit in it, scoped to your `orgs` automatically |
| `repos` | always, bypassing the filters above |

**Different keys combine with AND, and the values within one key with OR.** The example above selects Go repos that have the `service` topic and a `go.mod`. With no filters at all, every non-excluded repo in the org matches.

!!! note "Code search"
    `search` uses GitHub's code search API, which has its own rate limit. gh fleet waits for it to reset instead of failing. Prefer `files` when a path check is enough: it is checked in batched queries of 50 repos.

## Exclude rules

```yaml
  exclude:
    repos: [acme/legacy-api]
    topics: [no-fleet]
    archived: true
    forks: true
```

**Exclusion always wins**, even over `include.repos`. `archived` and `forks` both default to `true`, so archived repos and forks are skipped unless you set them to `false`.

## Push access

A repo that matches but that you cannot push to is reported as **skipped** (`no push access`) instead of failing. `exec` is read-only, so it keeps those repos.

## Check before you run

`targets` lists every repo with the reason it matched:

```console
$ gh fleet targets -f fleet.yaml
REPO          MATCHED BY
acme/api      file:.github/workflows
acme/billing  file:.github/workflows
...
SKIPPED         REASON
acme/docs-site  no push access

5 repos (1 skipped)
```

Add `--json` for scripts, with `--jq` to filter:

```sh
gh fleet targets --json --jq '.targets[].repo | .owner + "/" + .name'
```

## Narrowing a run

`--repo owner/name` limits `plan`, `apply`, `status`, `retry` and `close` to specific repos, and repeats. A name that is not a campaign target is an error, so a typo never silently does nothing.

```sh
gh fleet apply --repo acme/api --repo acme/web
```
