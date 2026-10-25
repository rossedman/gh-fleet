---
title: Writing changes
---

# Writing changes

The change is a script. gh fleet runs it in a fresh clone of each repo and turns whatever it leaves behind into a commit and a pull request. This guide is the contract between your script and gh fleet.

## The contract

- The script runs with the **repo root as its working directory**, in a clean, shallow clone of the base branch with the campaign branch checked out.
- It changes files in place. It does not commit, push or open anything; gh fleet does that.
- The outcome comes from the exit code and the working tree:

| Script result | Outcome |
| --- | --- |
| Exit 0, files differ from the base branch | **changed**: committed, pushed and put in a PR |
| Exit 0, nothing differs | **noop**: nothing is pushed, no PR is opened |
| Non-zero exit, or the timeout is hit | **failed**: the repo is recorded with the error, others carry on |

Changes are detected with `git status`, so files matched by the repo's `.gitignore` do not count.

!!! tip "Make scripts idempotent"
    gh fleet runs the script again on every `plan` and `apply`, starting from a fresh clone of the base branch. A script that checks first and exits quietly when the work is already done produces a clean `noop` once the PR has merged.

## Environment

The script inherits your environment (so `gh` authentication works inside it), plus anything in `change.env`, plus these:

| Variable | Value |
| --- | --- |
| `FLEET_REPO` | the repo as `owner/name` |
| `FLEET_CAMPAIGN` | the campaign name |
| `FLEET_DEFAULT_BRANCH` | the repo's default branch |
| `FLEET_DRY_RUN` | `true` during `plan` and `exec`, `false` during `apply` |
| `FLEET_OUTPUT` | a temporary directory for files the script hands back to gh fleet |
| `FLEET_VAR_<NAME>` | each of the repo's [variables](../configuration.md#variables); lists are comma-joined |
| `FLEET_PROP_<NAME>` | each of the repo's custom properties |

`<NAME>` is the variable or property name in upper case, with anything other than letters, digits and underscores turned into `_`: the property `cost-center` becomes `FLEET_PROP_COST_CENTER`.

`FLEET_DRY_RUN` lets a script skip side effects other than editing files, such as calling an external service. Editing files is always safe: it happens in a throwaway clone.

## Inline or file

`change.run` is either an inline script or a path to a file:

```yaml
change:
  run: ./scripts/pin.sh      # a file next to the campaign file
```

```yaml
change:
  run: |                     # inline
    sed -i 's/old/new/g' config.yaml
```

A single-line value that names an existing file, relative to the campaign file, is run as `bash <file>`; anything else is run as `bash -c <script>`. Files do not need the executable bit. Use `shell` to pick another shell.

Keep anything longer than a few lines in a file, so it can be linted and tested on its own. The [examples](../examples.md) ship tested scripts.

## Choosing the commit message

The commit message defaults to the rendered `pr.commit.message`, which defaults to the PR title. A script can override it for one repo by writing to `$FLEET_OUTPUT/commit-message`:

```sh
echo "build: bump go to ${GO_VERSION} in ${FLEET_REPO}" > "$FLEET_OUTPUT/commit-message"
```

Set `pr.commit.signoff: true` to add a `Signed-off-by` trailer. Commit signing follows your git configuration.

## Handing values back to the PR

A script can compute values the PR text needs, such as the version it replaced, by writing `NAME=value` lines to `$FLEET_OUTPUT/vars`. They are added to the repo's variables (replacing any with the same name) before the PR title, body, commit message, labels and reviewers are rendered:

```sh
old=$(sed -n 's/^go //p' go.mod)
go mod edit -go="$GO_VERSION"
echo "old_go=$old" >> "$FLEET_OUTPUT/vars"
```

```yaml
pr:
  body: "Moves {{ .Repo.Name }} from Go {{ .Vars.old_go }} to {{ .Vars.go }}."
```

Blank lines and lines starting with `#` are ignored. Any other line that isn't `NAME=value` fails the repo, so a typo can't slip into a PR.

## Timeouts and interruption

`change.timeout` (default `5m`) limits each repo's run. When it expires the script is stopped and the repo is marked `failed`.

Press Ctrl-C once and gh fleet stops starting new repos but lets the scripts already running finish, or hit their timeout, so no repo is left half-changed. State is saved. Press it a second time to exit immediately.

## Debugging a script

- Every script's output is written to `$XDG_STATE_HOME/gh-fleet/<name>/logs/<owner>__<repo>.log`, replaced on each run. The last lines of output are also in the `NOTE` column of `gh fleet status`.
- `--keep-workdirs` leaves the clones on disk, under `gh-fleet/<campaign>/<owner>/<repo>` in your temporary directory, so you can reproduce a failure by hand.
- `gh fleet plan --repo owner/name` is the fastest loop: one repo, no writes.

## Pull request text

`pr.title`, `pr.body`, `pr.commit.message` and each entry of `pr.labels` and `pr.reviewers` are Go templates. `{{ .Repo.FullName }}`, `{{ .Repo.Name }}`, `{{ .Vars.<name> }}` and `{{ .Campaign.Name }}` are the ones used most; [variables](../configuration.md#variables) lists everything available. A longer body can live in a file next to the campaign file:

```yaml
pr:
  title: "ci: pin GitHub Actions to commit SHAs"
  body: ./pr-body.md
```
