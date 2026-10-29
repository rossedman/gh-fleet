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

With limits.waves, plan takes one wave at a time, like apply: the wave apply
would roll out now. --wave plans another wave, and --all-waves plans every
target. --repo plans the repos you name, whatever their wave.

The script sees FLEET_DRY_RUN=true.

```
gh fleet plan [campaign] [flags]
```

### Examples

```
  gh fleet plan --repo acme/api
  gh fleet plan --wave 3
  gh fleet plan --json repo,status --jq '[.[] | select(.status == "changed") | .repo]'
```

### Options

```
      --all-waves              plan every target, whatever its wave
      --concurrency int        repos cloned and run in parallel (default limits.concurrency, else 16)
      --exclude strings        leave out these repos (owner/name, repeatable)
      --exclude-file strings   leave out the repos listed in this file, one per line (- for stdin, repeatable)
  -f, --file string            campaign file (default ./fleet.yaml)
  -h, --help                   help for plan
  -q, --jq expression          filter JSON output using a jq expression
      --json fields            output JSON with the specified fields (comma-separated; --json alone lists them)
      --keep-workdirs          leave clones on disk for debugging
      --no-git-cache           clone every repo from GitHub instead of through the git cache kept between runs
      --repo strings           limit to these repos (owner/name, repeatable)
      --repo-file strings      limit to the repos listed in this file, one per line (- for stdin, repeatable)
  -t, --template string        format JSON output using a Go template; see gh help formatting
      --ungrouped              print each repo's diff as it finishes instead of grouping identical diffs
      --wave wave              plan this wave of limits.waves (default the one apply would roll out now)
      --write-rate float       GitHub API writes per second (default 1)
```

### Options inherited from parent commands

```
      --config string   config file (default $XDG_CONFIG_HOME/gh-fleet/config.yaml)
      --debug           log what gh fleet is doing and print how long each part took
```

### SEE ALSO

* [gh fleet](gh_fleet.md)	 - Apply one change across many repositories and track the PRs

