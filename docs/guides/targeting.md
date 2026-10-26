---
title: Targeting repositories
---

# Targeting repositories

The `targets` section decides which repositories a campaign touches. This guide covers every filter, how they combine, and how to check the result before anything runs.

## Where candidates come from

`orgs` lists organizations (or users) whose repositories are candidates. Each org is listed with one paginated GraphQL query, 100 repos per page, and the listing is cached for 10 minutes, so running `targets`, `plan` and `apply` in a row costs one listing. `include.repos` adds individual repos, even from other owners.

A campaign needs `orgs`, `include.repos`, or both.

### Just some repos

To change a handful of repos, list them in `include.repos` with no other include filter. The campaign then targets only those repos, however big the org is:

```yaml
targets:
  orgs: [acme]          # scopes the bare names below
  include:
    repos: [api, web, tools/cli]
```

Only what you name is fetched:
- **`owner/name`:** that repo is looked up, with no listing.
- **A bare name:** looked up in each org, and skipped where it doesn't exist.
- **A pattern:** its owner, or `orgs` for a bare pattern like `*-domain`, is listed once and only the matches are kept.

No org listing runs for exact names. Lookups go 50 repos to a query, so a list of 500 repos costs 10 queries even in an org of thousands. Add any include filter, such as `topics`, and `include.repos` goes back to adding its repos to the org's matches.

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
| `names` | its name matches any of these [patterns](#name-patterns), such as `service-*` |
| `properties` | every listed [custom property](#custom-properties) has one of its values |
| `repos` | always, bypassing the filters above; names or [patterns](#name-patterns). [Alone, it's the whole selection.](#just-some-repos) |

**Different keys combine with AND, and the values within one key with OR.** The example above selects Go repos that have the `service` topic and a `go.mod`. With no filters at all, every non-excluded repo in the org matches, unless `include.repos` is set: then only the repos it names do.

### Searching a big org

With `search` and `orgs`, gh fleet runs the search first and looks up only the repos with hits, 50 to a query. The org is never listed, so finding a few matches in an org of 5,000 repos takes a few requests rather than one per 100 repos. The other filters then apply to those repos only.

GitHub returns at most 1,000 matching files for one search, and counts every file, so a few busy repos can use up that limit before quieter ones appear. When a search matches more files than that, gh fleet splits it by file size until each part fits, so no repo is dropped. Each part is a request, and code search allows 10 a minute, so a broad query takes longer, and gh fleet says when it is waiting.

If GitHub still can't return everything, for example when over 1,000 matching files have the same size or your query sets its own `size:`, gh fleet prints a warning naming the query, rather than quietly missing repos.

!!! note "What code search can't see"
    GitHub's code search only covers each repo's default branch and files under 384 KB, and leaves out forks with fewer stars than their parent. When you know the path, `files` is exact: it checks every candidate repo, 50 repos to a query and 4 queries at a time.

## Exclude rules

```yaml
  exclude:
    repos: [acme/legacy-api]
    topics: [no-fleet]
    archived: true
    forks: true
```

`exclude.properties` drops repos whose custom property has any of the listed values, for example `properties: {pci: true}`.

**Exclusion always wins**, even over `include.repos`. `archived` and `forks` both default to `true`, so archived repos and forks are skipped unless you set them to `false`. Empty repos, which have no default branch, are always skipped.

## Name patterns

Anywhere a campaign names repos, a glob pattern can stand for many:

```yaml
targets:
  orgs: [acme]
  include:
    names: [service-*, "api-v[0-9]"]   # only these names
    repos: [tools/cli-*]               # plus every cli-* repo in tools
  exclude:
    repos: ["*/legacy-*", acme/sandbox-?]
```

| Pattern | Matches |
| --- | --- |
| `*` | any run of characters, except `/` |
| `?` | any one character |
| `[a-c]`, `[0-9]` | one character in the range |
| `[^a-c]` | one character not in the range |

Matching ignores case. A pattern without a slash matches the repo's name, and one with a slash matches `owner/name`, so `*-domain` and `acme/*-domain` both work. This holds everywhere repos are listed: `include.names`, `include.repos`, `exclude.repos`, [`vars.match[].when.repos`](../configuration.md#variables) and [`limits.waves[].repos`](tracking-prs.md#rolling-out-in-waves).

`include.names` is a filter like `topics`, so it combines with the other filters with AND: `names: [service-*]` and `languages: [Go]` select Go repos named `service-*`. A pattern in `include.repos` instead adds every match and bypasses the filters, as a plain name does. With an owner, the owner doesn't need to be in `orgs`: `tools/cli-*` lists `tools` once and keeps the matches. Without one, as in `*-domain`, or with a pattern for the owner, as in `*/cli-*`, it only matches the repos of `orgs`, so the campaign needs `orgs`.

A pattern that doesn't parse, such as `acme/[ab`, is an error when the campaign loads, with its line and column.

On the command line, `--exclude` and `--exclude-file` take patterns too (quote them so the shell leaves them alone); `--repo` and `--repo-file` take exact names, so a typo is an error rather than an empty run.

!!! note "Custom properties cost a call per 100 repos"
    gh fleet only reads custom properties when the campaign uses them, in filters, `vars` rules, PR templates, `FLEET_PROP_*` in a script, or `repo.properties` in Starlark. A campaign that doesn't use them never pays for them.

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

`--repo owner/name` limits `targets`, `plan`, `apply`, `status`, `merge`, `retry` and `close` to specific repos, and repeats. A name that is not a campaign target is an error, so a typo never silently does nothing. Only the named repos are looked up, 50 to a query, and checked against the filters, so `--repo` and `--repo-file` are quick even when the campaign targets thousands.

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
