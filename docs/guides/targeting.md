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
| `properties` | every listed [custom property](#custom-properties) has one of its values |
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

`exclude.properties` drops repos whose custom property has any of the listed values, for example `properties: {pci: true}`.

**Exclusion always wins**, even over `include.repos`. `archived` and `forks` both default to `true`, so archived repos and forks are skipped unless you set them to `false`.

## Custom properties

Organizations can define [custom properties](https://docs.github.com/en/organizations/managing-organization-settings/managing-custom-properties-for-repositories-in-your-organization) on their repos, such as `team`, `tier` or `pci`. They are often a better handle than topics, because the org controls them and every repo has a value. gh fleet reads them with the org listing, in one paginated call per org, and caches them with it.

```yaml
targets:
  orgs: [acme]
  include:
    properties:
      tier: [1, 2]        # any of these values
      runtime: node       # and this one
  exclude:
    properties:
      lifecycle: deprecated
```

Each listed property must match one of its values, ignoring case. A multi-select property matches when any selected value is listed. A repo without the property doesn't match. Numbers and booleans can be written bare: `tier: [1, 2]` and `tier: ["1", "2"]` are the same. `gh fleet targets` shows the property that matched, as in `property:tier=1`.

If your token can't read an org's properties, a campaign that filters on them or uses them in PR text fails with that error; any other campaign carries on as if the repos had none.

The same properties reach your script as `FLEET_PROP_<NAME>` and your PR text as `{{ .Repo.Properties.tier }}`; see [variables](../configuration.md#variables).

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

Piped, `targets` prints just the repos, tab-separated. For more, `--json targets,skipped` prints both lists, and `--jq` filters them:

```sh
gh fleet targets --json targets --jq '.targets[].repo | .owner + "/" + .name'
```

## Narrowing a run

`--repo owner/name` limits `targets`, `plan`, `apply`, `status`, `merge`, `retry` and `close` to specific repos, and repeats. A name that is not a campaign target is an error, so a typo never silently does nothing.

```sh
gh fleet apply --repo acme/api --repo acme/web
```

`--exclude owner/name` does the opposite: it leaves repos out of this run, on top of the campaign's `exclude.repos`. A repo that is both listed and excluded is left out. Status, merge and close also skip an excluded repo's PR from earlier runs.

For longer lists, `--repo-file` and `--exclude-file` read repo names from a file, one per line, and repeat. Blank lines and `#` comments are skipped. A line can be `owner/name` or a GitHub URL, and only its first field counts, so the output of `gh repo list` works as is. `-` reads stdin.

```sh
# Roll out to the pilot repos first, then everyone else.
gh fleet apply --repo-file pilot.txt
gh fleet apply --exclude-file pilot.txt

# Leave out repos with an open security freeze.
gh repo list acme --topic freeze --limit 1000 | gh fleet plan --exclude-file -
```

A bad line fails the run with its file and line number. An empty `--repo-file` is an error rather than "no limit", so a list that came out empty never runs against every target.
