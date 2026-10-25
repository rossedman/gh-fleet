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
  run: ./scripts/pin.sh
  timeout: 5m

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

### Identity

`apiVersion` must be `fleet.gh/v1`. `name` is the campaign's identity: it derives the branch (`fleet/<name>`), the hidden marker in each PR body, and the name of the local state file. **Renaming a campaign starts a new one**, and the old PRs are no longer found. Use lowercase letters, digits and dashes.

### `targets`

Which repositories the campaign touches: the candidates from `orgs` (and any `include.repos`), narrowed by `include` and trimmed by `exclude`. Include keys combine with AND and values within a key with OR. By default archived repos and forks are excluded. See [Targeting repositories](guides/targeting.md).

### `change`

The script run in a fresh clone of each repo. `run` is either a path relative to the campaign file or an inline script; `shell` defaults to `bash`, `timeout` to `5m`, and `env` adds variables. See [Writing changes](guides/writing-changes.md).

### `pr`

How each pull request is opened. `title` is required, and `title`, `body` and the commit message are Go templates with `.Repo` (`.Owner`, `.Name`, `.FullName`, `.DefaultBranch`) and `.Campaign` (for example `.Campaign.Name`). A template that refers to a missing field is an error instead of rendering as empty text.

`body` is inline text or a file relative to the campaign file. gh fleet always appends the hidden marker to the rendered body, so keep that out of your templates.

| Field | Default |
| --- | --- |
| `branch` | `fleet/<name>` |
| `base` | the repo's default branch |
| `draft` | `false` |
| `commit.message` | the rendered PR title |
| `commit.signoff` | `false` |

`reviewers` takes users or `org/team` slugs and is requested when the PR is created. On later runs `apply` updates an open PR's title and body when they differ and adds any labels it is missing; it never removes labels, re-requests reviewers or changes `draft`.

### `limits`

`concurrency` is how many repos are processed in parallel (default 8, between 1 and 256); `--concurrency` overrides it. `maxOpenPRs` stops opening new PRs once that many campaign PRs are open (default 0, no limit). Repos held back this way are marked `changed` and `retry` picks them up later.

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
| Clones | `gh-fleet/<campaign>/<owner>/<repo>` under `$TMPDIR`, removed when a repo finishes unless `--keep-workdirs` is set |

`XDG_STATE_HOME` falls back to `~/.local/state`.
