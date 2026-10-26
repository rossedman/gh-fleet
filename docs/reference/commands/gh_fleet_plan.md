# gh fleet plan

Run the change in fresh clones and show per-repo diffs, without pushing

### Synopsis

Clone every target, run the change script and print the unified diff each repo
would get. Nothing is pushed and nothing is written to GitHub. Repos where the
script changes nothing are reported as noop.

Diffs are printed at the end, each distinct diff once with the repos that
get it, so a change made the same way in 300 repos is read once. Blob hashes
and hunk line numbers don't count as differences. --ungrouped prints each
repo's diff as it finishes instead.

The script sees FLEET_DRY_RUN=true.

```
gh fleet plan [campaign] [flags]
```

### Examples

```
  gh fleet plan --repo acme/api
  gh fleet plan --json repo,status --jq '[.[] | select(.status == "changed") | .repo]'
```

### Options

```
      --concurrency int        repos processed in parallel (default limits.concurrency, else 8)
      --exclude strings        leave out these repos (owner/name, repeatable)
      --exclude-file strings   leave out the repos listed in this file, one per line (- for stdin, repeatable)
  -f, --file string            campaign file (default ./fleet.yaml)
  -h, --help                   help for plan
  -q, --jq expression          filter JSON output using a jq expression
      --json fields            output JSON with the specified fields (comma-separated; --json alone lists them)
      --keep-workdirs          leave clones on disk for debugging
      --repo strings           limit to these repos (owner/name, repeatable)
      --repo-file strings      limit to the repos listed in this file, one per line (- for stdin, repeatable)
  -t, --template string        format JSON output using a Go template; see gh help formatting
      --ungrouped              print each repo's diff as it finishes instead of grouping identical diffs
      --write-rate float       GitHub API writes per second (default 1)
```

### Options inherited from parent commands

```
      --config string   config file (default $XDG_CONFIG_HOME/gh-fleet/config.yaml)
```

### SEE ALSO

* [gh fleet](gh_fleet.md)	 - Apply one change across many repositories and track the PRs

