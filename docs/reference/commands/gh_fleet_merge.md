# gh fleet merge

Merge campaign PRs that are approved and passing, and say what the rest wait on

### Synopsis

Merge every open campaign PR that is ready: GitHub reports it mergeable, its
checks pass and it is approved. The others are listed with what they wait on:
a review, running or failing checks, a conflict, or a base that moved ahead.

By default a PR needs an approving review and passing checks even where the
repo doesn't require them; --allow-unreviewed and --allow-no-checks relax
that. Merging uses the head commit gh fleet checked, so a PR that gets a new
push in the meantime is left for the next run.

--auto turns on auto-merge for PRs that aren't ready yet, so GitHub merges
each one when its requirements pass. --update-branch merges the base into
PRs that are behind it. --max rolls a campaign out in batches. Branches of
merged PRs are deleted unless --keep-branches is passed.

--dry-run shows what would happen. Asks for confirmation unless --yes is
passed.

```
gh fleet merge [campaign] [flags]
```

### Examples

```
  gh fleet merge --dry-run
  gh fleet merge --max 10 --yes
  gh fleet merge --auto --update-branch --method squash
```

### Options

```
      --allow-no-checks        merge PRs whose head has no checks
      --allow-unreviewed       merge PRs in repos that don't require a review
      --auto                   turn on auto-merge for PRs that aren't ready yet
      --concurrency int        repos processed in parallel (default limits.concurrency, else 8)
      --dry-run                show what would happen without changing anything
      --exclude strings        leave out these repos (owner/name, repeatable)
      --exclude-file strings   leave out the repos listed in this file, one per line (- for stdin, repeatable)
  -f, --file string            campaign file (default ./fleet.yaml)
  -h, --help                   help for merge
  -q, --jq expression          filter JSON output using a jq expression
      --json fields            output JSON with the specified fields (comma-separated; --json alone lists them)
      --keep-branches          keep the branches of merged PRs
      --max int                merge or auto-merge at most this many PRs (0: no limit)
      --method string          merge method: merge, squash or rebase (default "squash")
      --repo strings           limit to these repos (owner/name, repeatable)
      --repo-file strings      limit to the repos listed in this file, one per line (- for stdin, repeatable)
  -t, --template string        format JSON output using a Go template; see gh help formatting
      --update-branch          merge the base into PRs that are behind it
      --write-rate float       GitHub API writes per second (default 1)
  -y, --yes                    skip the confirmation prompt
```

### Options inherited from parent commands

```
      --config string   config file (default $XDG_CONFIG_HOME/gh-fleet/config.yaml)
      --debug           log what gh fleet is doing and print how long each part took
```

### SEE ALSO

* [gh fleet](gh_fleet.md)	 - Apply one change across many repositories and track the PRs

