---
title: Configuration
---

# Configuration

gh fleet has two kinds of settings: the campaign file, which describes one change, and CLI settings, which describe how you run gh fleet on your machine.

## The campaign file

A campaign file has five sections. Every field is documented in the [campaign file reference](reference/campaign.md), which is generated from the JSON Schema built into the binary. This page walks through how the sections fit together.

```yaml title="fleet.yaml"
# yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
apiVersion: fleet.gh/v1
name: pin-actions
description: Pin every third-party GitHub Action to a full commit SHA

targets:
  orgs: [acme]
  include:
    files: [.github/workflows]
  exclude:
    topics: [no-fleet]

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
  body: ./pr-body.md
  labels: [fleet, security]
  commit:
    signoff: true

limits:
  concurrency: 8
  maxOpenPRs: 200
```

The two Starlark steps it names sit next to it. They do the editing; the shell step between them only resolves tags to commits with `gh api`:

=== "change.star"

    ```python
    --8<-- "examples/pin-actions/change.star"
    ```

=== "pin.star"

    ```python
    --8<-- "examples/pin-actions/pin.star"
    ```

Every [example](examples.md) has a page with its campaign file, its Starlark and its test cases.

### Identity

`apiVersion` must be `fleet.gh/v1`. `name` is the campaign's identity: it derives the branch (`fleet/<name>`), the hidden marker in each PR body, and the name of the local state file. **Renaming a campaign starts a new one**, and the old PRs are no longer found. Use lowercase letters, digits and dashes.

### `targets`

Which repositories the campaign touches: the candidates from `orgs` (and any `include.repos`), narrowed by `include` and trimmed by `exclude`. Include keys combine with AND and values within a key with OR. By default archived repos and forks are excluded. See [Targeting repositories](guides/targeting.md).

### `change`

What changes in a fresh clone of each repo: `steps`, run in order in the same clone. Each step is a shell script in `run` (a path relative to the campaign file, or an inline script) or a [Starlark script](guides/starlark.md) in `starlark`, and can have a `name`. Each step has its own `timeout` (default `5m`). `shell` (default `bash`) and `env` can be set on `change` for every step and on a step for that one. Variables a step sets reach the steps after it. See [Writing changes](guides/writing-changes.md).

### `pr`

How each pull request is opened. `title` is required. `title`, `body`, the commit message and every entry of `labels` and `reviewers` are Go templates with `.Repo`, `.Vars` and `.Campaign`; see [variables](#variables). A template that refers to a missing field is an error for that repo instead of rendering as empty text.

`body` is inline text or a file relative to the campaign file. gh fleet always appends the hidden marker to the rendered body, so keep that out of your templates.

| Field | Default |
| --- | --- |
| `branch` | `fleet/<name>` |
| `base` | the repo's default branch |
| `draft` | `false` |
| `commit.message` | the rendered PR title |
| `commit.signoff` | `false` |

`reviewers` takes users or `org/team` slugs and is requested when the PR is created. On later runs `apply` updates an open PR's title and body when they differ and adds any labels it is missing; it never removes labels, re-requests reviewers or changes `draft`.

`reviewersFrom` also asks each repo's own owners, so a fleet-wide change reaches the people who know each repo:

```yaml
pr:
  reviewers: [acme/platform]
  reviewersFrom: [codeowners, catalog]
```

- `codeowners` reads the repo's CODEOWNERS (`.github/`, the root or `docs/`) and requests the owners of the files the change touched, the last matching rule winning as on GitHub. Email owners are skipped.
- `catalog` reads `catalog-info.yaml` and requests each entity's `spec.owner`: a group (`payments`, `group:payments` or `group:default/payments`) becomes the team `<org>/payments`, a user ref becomes that user.

Owners are requested after `reviewers`, without duplicates. An owner GitHub refuses, such as the PR's author or a team without access to the repo, is skipped and noted in the repo's log instead of failing it.

### Variables

Variables let one campaign say different things to different repos: a different owning team, a different on-call channel, a version that varies by repo. Each repo's variables are built from three sources in the `vars` block, later ones winning:

```yaml
vars:
  default:                         # 1. every repo starts with these
    team: platform
    node: 22
    labels: [fleet, node]
  match:                           # 2. then each rule whose `when` matches, in order
    - when: { topics: [frontend] }
      set: { team: web }
    - when: { repos: [acme/legacy-api] }
      set: { node: 20 }
    - when: { properties: { tier: 1 } }
      set: { team: sre }
  file: ./owners.csv               # 3. then the repo's row in a data file
```

`when` takes `repos`, `topics`, `languages` and `properties`, with the same meaning as in `targets.include`: every key that is set must match, and values within a key are alternatives. An empty `when` matches every repo.

`file` is a `.csv`, `.json`, `.yaml` or `.yml` file next to the campaign file. A CSV needs a `repo` column and has one column per variable; an empty cell leaves that variable unset. JSON and YAML map `owner/name` to an object:

```csv
repo,team,oncall
acme/api,payments,#payments-oncall
acme/web,web,#web-oncall
```

Variable names use letters, digits and underscores, and values are strings, numbers, booleans or lists of those.

**Where they show up:**

| In | Variables | Custom properties | Repo |
| --- | --- | --- | --- |
| Scripts | `FLEET_VAR_TEAM` | `FLEET_PROP_TIER` | `FLEET_REPO`, `FLEET_DEFAULT_BRANCH` |
| PR text | `{{ .Vars.team }}` | `{{ .Repo.Properties.tier }}` | `{{ .Repo.Name }}`, `.Owner`, `.FullName`, `.DefaultBranch`, `.Language`, `.Topics` |

Lists are comma-joined in scripts. In templates, `join` joins a list with any separator (`{{ join .Vars.labels ", " }}`), and `lower` and `upper` change case. A `labels` or `reviewers` entry that renders to a comma-separated list becomes several entries, so `reviewers: ["{{ join .Vars.reviewers \",\" }}"]` requests each of them.

A script can add or override variables for its repo by writing `NAME=value` lines to `$FLEET_OUTPUT/vars`; see [handing values back](guides/writing-changes.md#handing-values-back-to-the-pr).

**Check before you run.** `gh fleet targets` shows every repo's resolved variables, and `--json` includes them along with the custom properties. It also warns when the data file names a repo that isn't a target, which is usually a typo. A template that uses a variable a repo doesn't have fails that repo with `missing variable "oncall" for acme/web`, in `plan` as well as `apply` and before anything is pushed, so nothing goes out half-filled.

!!! warning "Variables are not secrets"
    Variables end up in PR bodies, logs and the local state file. A secret a script needs belongs in your own environment, which scripts inherit.

### `limits`

`concurrency` is how many repos are cloned and run in parallel (default 16, between 1 and 256); `--concurrency` overrides it. GitHub API requests are capped separately, so this doesn't add API load ([rate limits](guides/rate-limits.md#tuning)). `maxOpenPRs` stops opening new PRs once that many campaign PRs are open (default 0, no limit). Repos held back this way are marked `changed` and `retry` picks them up later.

`waves` rolls a change out in stages; see [Rolling out in waves](guides/tracking-prs.md#rolling-out-in-waves).

### Editor support

Files written by `gh fleet init` start with a `yaml-language-server` header pointing at the schema, so editors that support it validate and complete the file. Errors from the CLI carry the line and column they refer to.

## CLI settings

Flags can also come from the environment and from a config file. The order of precedence is:

1. A command-line flag
2. A `GH_FLEET_*` environment variable
3. The config file

### Environment variables

The variable for a flag is `GH_FLEET_` plus the flag name in upper case, with dashes turned into underscores.

```sh
export GH_FLEET_WRITE_RATE=0.5
export GH_FLEET_CONCURRENCY=4
gh fleet apply
```

### Config file

gh fleet reads `$XDG_CONFIG_HOME/gh-fleet/config.yaml` (`~/.config/gh-fleet/config.yaml` when `XDG_CONFIG_HOME` is unset). It also looks for a `config.yaml` in your home directory and the current directory. Use `--config` to name a file explicitly; a file you name must exist and parse.

Keys are flag names:

```yaml title="~/.config/gh-fleet/config.yaml"
write-rate: 0.5
concurrency: 4
```

### Where gh fleet keeps files

| What | Where |
| --- | --- |
| Campaign state | `$XDG_STATE_HOME/gh-fleet/<name>.json` |
| Per-repo script logs | `$XDG_STATE_HOME/gh-fleet/<name>/logs/` |
| Org listing cache (10 minutes) | `gh-fleet/repos/` under your user cache directory |
| Clones | `gh-fleet/<campaign>/<owner>/<repo>` under `$TMPDIR`, moved aside when a repo finishes and deleted in the background, all before the command exits, unless `--keep-workdirs` is set |

`XDG_STATE_HOME` falls back to `~/.local/state`.

### Finding out what's slow

`--debug` (or `GH_FLEET_DEBUG=1`) logs each step on stderr as it starts and finishes, and ends with a table of how long everything took: each phase of the command, each stage of each repo (clone, every change step, diff, cleanup, push, PR sync), every git command, every GitHub request by API, and the time requests spent waiting on rate limits and retries.

```sh
gh fleet plan --debug 2> plan-debug.log
```

Repos run in parallel, so per-repo, git and API totals add up across workers and can be larger than the wall time; compare the `AVG` and `MAX` columns, and `run change in every repo` against `total`, to see where the time goes.
