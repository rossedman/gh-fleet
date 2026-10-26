# gh fleet status

Show every target's PR, state, checks, reviews and last error

### Synopsis

Refresh every target's PR from GitHub, in one batched GraphQL query per 50
repos, and print a table with the PR link, state, check rollup, review decision
and the last error recorded locally.

GitHub is the source of truth: PRs are found by branch and by the hidden
campaign marker, so status is correct even after the local state file is
deleted. The remaining API quota is shown at the end.

--format markdown prints a progress report to paste into an issue, a PR or
chat: what needs attention, open PRs and what each waits on, and merged
PRs. --format html prints the same report as a self-contained page.

```
gh fleet status [campaign] [flags]
```

### Examples

```
  gh fleet status
  gh fleet status pin-actions --json --jq '.counts'
  gh fleet status --format markdown | gh issue comment 42 --body-file -
  gh fleet status --format html > report.html
```

### Options

```
      --concurrency int        repos processed in parallel (default limits.concurrency, else 8)
      --exclude strings        leave out these repos (owner/name, repeatable)
      --exclude-file strings   leave out the repos listed in this file, one per line (- for stdin, repeatable)
  -f, --file string            campaign file (default ./fleet.yaml)
      --format string          table, or a report to share: markdown or html (default "table")
  -h, --help                   help for status
      --jq string              filter JSON output with a jq expression
      --json                   print JSON instead of a table
      --repo strings           limit to these repos (owner/name, repeatable)
      --repo-file strings      limit to the repos listed in this file, one per line (- for stdin, repeatable)
      --write-rate float       GitHub API writes per second (default 1)
```

### Options inherited from parent commands

```
      --config string   config file (default $XDG_CONFIG_HOME/gh-fleet/config.yaml)
```

### SEE ALSO

* [gh fleet](gh_fleet.md)	 - Apply one change across many repositories and track the PRs

