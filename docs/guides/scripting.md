---
title: Scripting and CI
---

# Scripting and CI

gh fleet follows gh's conventions for scripts. If you've scripted `gh pr list`, the same habits work here.

## Output goes where scripts expect

- **stdout carries results.** Status lines, summaries, hints, warnings and progress go to stderr, so a pipe gets only the data.
- **Piped tables are tab-separated, with no header row.** That's how gh prints them, so `cut`, `awk` and `sort` work directly:

```sh
gh fleet status | awk -F'\t' '$2 == "failed" { print $1 }'
```

- **Styling is for terminals only.** Nothing is colored when output is piped, or when `NO_COLOR` or `CLICOLOR=0` is set. `CLICOLOR_FORCE=1` forces color, and `GH_FORCE_TTY` makes gh fleet lay output out as if it were writing to a terminal.
- **Long runs report progress on stderr.** `plan`, `apply`, `retry`, `merge`, `close` and `exec` first say what they're resolving (listing each org, checking files, searching code, looking up PRs). Then they list each repo as it finishes. In a terminal, a status line at the bottom shows how many repos are done, how many failed, the elapsed time and an estimate of what's left. In a log, the same line prints every 30 seconds. When GitHub's rate limit pauses every request, gh fleet says so and for how long. With `--json` and stderr not a terminal, only the heartbeat and rate limit lines print.
- **Errors are one plain line on stderr.** They read the same in a terminal, a CI log or a file.

## JSON, jq and templates

Commands that print results take the same three flags as gh:

| Flag | Does |
| --- | --- |
| `--json <fields>` | Prints JSON with only these comma-separated fields. `--json` alone lists the fields the command has. |
| `--jq <expression>`, `-q` | Filters the JSON with a jq expression. No jq install needed. |
| `--template <template>`, `-t` | Formats the JSON with a Go template. Use `join`, `pluck`, `truncate`, `timefmt` and `json` there. |

```sh
gh fleet list --json
# Specify one or more comma-separated fields for `--json`:
#   file
#   name
#   ...

gh fleet plan --json repo,status --jq '.[] | select(.status == "changed") | .repo'
gh fleet merge --dry-run --json repo,action,reason -t '{{range .}}{{.repo}}: {{.action}} {{.reason}}{{"\n"}}{{end}}'
```

**How fields work.** For commands that print a list, `--json` picks fields from each item. For commands that print one object, `--json` picks its top-level fields. `status` prints `campaign`, `repos`, `counts` and `quota`, and `targets` prints `targets` and `skipped`. Unlike gh, `--jq` and `--template` also work without `--json`, and then they see every field.

## Exit codes

| Code | Meaning |
| --- | --- |
| 0 | Success |
| 1 | An error, or at least one repo failed (the others still ran) |
| 2 | You declined the confirmation prompt |
| 4 | gh isn't signed in to the host; run `gh auth login` or set `GH_TOKEN` |
| 8 | `status --exit-status` only: a repo still needs work |
| 130 | Stopped with Ctrl-C |

`status --exit-status` makes a campaign something a script can wait on:

```sh
until gh fleet status pin-actions --exit-status >/dev/null; do
  [ $? -eq 8 ] || exit 1   # a repo failed
  sleep 600
done
echo "every PR is merged"
```

It exits 0 once every repo is merged, closed, a no-op or skipped, 1 if any failed, and 8 otherwise.

## Prompts and authentication

- **Confirmation.** Commands that write to GitHub ask first in a terminal. Without one, they fail and point you at `--yes` rather than hang. `GH_PROMPT_DISABLED=1` turns prompts off in a terminal too.
- **Authentication.** gh fleet uses gh's: whatever `gh auth login` stored, or `GH_TOKEN`, `GH_HOST` and `GH_ENTERPRISE_TOKEN` for GitHub Enterprise.
