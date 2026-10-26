---
title: Campaign file reference
---

# Campaign file reference

Every field of a campaign file, generated from the JSON Schema shipped in the binary. Editors that understand `yaml-language-server` validate and complete files written by `gh fleet init`.

A campaign describes one change applied across many repositories: which repos to target, what to run in each, and how to open the pull request.

Schema URL: `https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json`

## Top level

| Field | Type | Default | Description |
| --- | --- | --- | --- |
| `apiVersion` (required) | string | `fleet.gh/v1` | Schema version of the campaign file. Must be `fleet.gh/v1`. |
| `name` (required) | string |  | The campaign's identity. It derives the branch name and the PR marker, so renaming it starts a new campaign. Lowercase letters, digits and dashes. |
| `description` | string |  | Free-form description, available to the PR body template as `.Campaign.Description`. |
| `targets` (required) | object |  | Which repositories the campaign touches. Needs `orgs`, `include.repos`, or both. |
| `change` (required) | object |  | What to change in each repository: an ordered list of `steps`, each a shell script (`run`) or a Starlark script (`starlark`). Steps run in order in the same checkout, and variables a step sets are visible to the steps after it. |
| `pr` (required) | object |  | How the pull request is opened. |
| `vars` | object |  | Per-repo variables, available to PR templates as `.Vars` and to scripts as `FLEET_VAR_<NAME>`. Values are built from `default`, then every matching `match` rule in order, then the repo's row in `file`; later sources win. |
| `limits` | object |  | Safety limits for a run. |

## `targets`

Which repositories the campaign touches. Needs `orgs`, `include.repos`, or both.

| Field | Type | Default | Description |
| --- | --- | --- | --- |
| `exclude` | object |  | Repos to drop. Exclusion always wins, even over `include.repos`. |
| `include` | object |  | Filters a candidate must pass. Keys combine with AND; values within a key combine with OR. |
| `orgs` | list of string |  | Organizations (or users) whose repositories are candidates. |

## `targets.exclude`

Repos to drop. Exclusion always wins, even over `include.repos`.

| Field | Type | Default | Description |
| --- | --- | --- | --- |
| `archived` | boolean | `true` | Skip archived repos. |
| `forks` | boolean | `true` | Skip forks. |
| `properties` | map of property to value or list |  | Skip repos whose custom property has any of the listed values. |
| `repos` | list of repo or pattern |  | Repos to skip: `owner/name`, a name, or a glob pattern such as `legacy-*` or `*/legacy-*`. |
| `topics` | list of string |  | Skip repos with any of these topics. |

## `targets.include`

Filters a candidate must pass. Keys combine with AND; values within a key combine with OR.

| Field | Type | Default | Description |
| --- | --- | --- | --- |
| `files` | list of string |  | Any of these paths (file or directory) exists on the default branch. |
| `languages` | list of string |  | The repo's primary language is any of these (case-insensitive). |
| `names` | list of repo or pattern |  | Glob patterns the repo must match, such as `service-*` or `acme/api-*`. A pattern without a slash matches the repo name, one with a slash the full `owner/name`. |
| `properties` | map of property to value or list |  | Custom property values the repo must have: every listed property must match one of its values (case-insensitive). |
| `repos` | list of repo or pattern |  | Repos that are always included, bypassing the filters above: `owner/name`, or a name or glob pattern such as `*-domain` matched against the repos of `orgs`. A pattern with an owner, such as `tools/cli-*`, lists that owner. |
| `search` | string |  | A code search query; only repos with a hit match. Scoped to `orgs` automatically. |
| `topics` | list of string |  | The repo has any of these topics. |

## `change`

What to change in each repository: an ordered list of `steps`, each a shell script (`run`) or a Starlark script (`starlark`). Steps run in order in the same checkout, and variables a step sets are visible to the steps after it.

| Field | Type | Default | Description |
| --- | --- | --- | --- |
| `env` | map of string |  | Environment variables for every step. |
| `shell` | string | `bash` | Default shell for `run` steps. |
| `steps` (required) | list of object |  | The steps to run, in order. |

## `change.steps[]`

One step of the change. Set exactly one of `run` or `starlark`.

| Field | Type | Default | Description |
| --- | --- | --- | --- |
| `env` | map of string |  | Environment variables for this step, added to `change.env`. |
| `name` | string |  | A label for the step, used in logs and errors. |
| `run` | string |  | A shell script path relative to the campaign file, or an inline script. Runs with the repo root as working directory. |
| `shell` | string |  | Shell for this `run` step. Defaults to `change.shell`. |
| `starlark` | string |  | A Starlark file relative to the campaign file that defines `change(repo)`. It edits files through `repo` with no shell or network access. |
| `timeout` | string | `5m` | Maximum run time of this step per repo, as a Go duration (`30s`, `5m`). |

## `pr`

How the pull request is opened.

| Field | Type | Default | Description |
| --- | --- | --- | --- |
| `base` | string | "" | Base branch. Empty means the repo's default branch. |
| `body` | string |  | PR body: a file relative to the campaign file, or inline text. A Go template with `.Repo` and `.Campaign`. |
| `branch` | string |  | Head branch name. Defaults to `fleet/<name>`. |
| `commit` | object |  | The commit made in each repo. |
| `draft` | boolean | `false` | Open PRs as drafts. |
| `labels` | list of string |  | Labels to add to each PR. |
| `reviewers` | list of string |  | Users or `org/team` slugs to request review from. |
| `reviewersFrom` | list of - |  | Also request review from each repo's own owners: `codeowners` for the CODEOWNERS owners of the files the change touches, `catalog` for the owners in its Backstage catalog-info.yaml (a group becomes the `org/group` team). An owner GitHub can't request, such as the PR's author or a team without access, is skipped. |
| `title` (required) | string |  | PR title. A Go template with `.Repo` and `.Campaign`. |

## `pr.commit`

The commit made in each repo.

| Field | Type | Default | Description |
| --- | --- | --- | --- |
| `message` | string |  | Commit message. Defaults to the PR title. A script can override it per repo by writing `$FLEET_OUTPUT/commit-message`. |
| `signoff` | boolean | `false` | Add a `Signed-off-by` trailer. |

## `vars`

Per-repo variables, available to PR templates as `.Vars` and to scripts as `FLEET_VAR_<NAME>`. Values are built from `default`, then every matching `match` rule in order, then the repo's row in `file`; later sources win.

| Field | Type | Default | Description |
| --- | --- | --- | --- |
| `default` | map of name to value or list |  | Values every repo starts with. |
| `file` | string |  | A `.csv`, `.json`, `.yaml` or `.yml` file of per-repo values, relative to the campaign file. A CSV needs a `repo` column and has one column per variable; JSON and YAML map `owner/name` to an object of variables. |
| `match` | list of object |  | Rules that set values on the repos they match, applied in order. |

## `vars.match[]`

Each entry of `vars.match`.

| Field | Type | Default | Description |
| --- | --- | --- | --- |
| `set` (required) | map of name to value or list |  | Values to set. |
| `when` (required) | object |  | Which repos the rule applies to. Every key that is set must match; values within a key are alternatives. An empty `when` matches every repo. |

## `vars.match[].when`

Which repos the rule applies to. Every key that is set must match; values within a key are alternatives. An empty `when` matches every repo.

| Field | Type | Default | Description |
| --- | --- | --- | --- |
| `languages` | list of string |  | The repo's primary language is any of these. |
| `properties` | map of property to value or list |  | Custom property values, as in `targets.include.properties`. |
| `repos` | list of repo or pattern |  | Repos: `owner/name`, a name, or a glob pattern such as `payments-*`. |
| `topics` | list of string |  | The repo has any of these topics. |

## `limits`

Safety limits for a run.

| Field | Type | Default | Description |
| --- | --- | --- | --- |
| `concurrency` | integer | `8` | Repos processed in parallel. `--concurrency` overrides it. Between 1 and 256. |
| `maxOpenPRs` | integer | `0` | Stop opening new PRs once this many campaign PRs are open. 0 means no limit. At least 0. |
| `waves` | list of object |  | Roll the change out in order. `apply` opens PRs for a wave only once every repo in the waves before it is merged or needed no change, so a problem stops at the canaries. Targets left after the last wave form one final wave. `--repo` bypasses waves. |

## `limits.waves[]`

Each entry of `limits.waves`.

| Field | Type | Default | Description |
| --- | --- | --- | --- |
| `repos` | list of repo or pattern |  | Repos in this wave, such as canaries: `owner/name`, a name, or a glob pattern such as `canary-*`. Repos that aren't targets are ignored. |
| `size` | integer or string |  | How many more targets join this wave, in name order: a number, or a percentage of all targets such as `25%`. At least 1. |

