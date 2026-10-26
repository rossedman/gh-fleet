# gh fleet targets

List the repos a campaign will touch and why each matched

### Synopsis

Resolve the campaign's targets and list every repo it will touch, with the
filters it matched. Repos without push access are listed as skipped.

Org listings are cached for 10 minutes.

```
gh fleet targets [campaign] [flags]
```

### Examples

```
  gh fleet targets
  gh fleet targets --json targets --jq '.targets[].repo | .owner + "/" + .name'
```

### Options

```
      --concurrency int        repos processed in parallel (default limits.concurrency, else 8)
      --exclude strings        leave out these repos (owner/name, repeatable)
      --exclude-file strings   leave out the repos listed in this file, one per line (- for stdin, repeatable)
  -f, --file string            campaign file (default ./fleet.yaml)
  -h, --help                   help for targets
  -q, --jq expression          filter JSON output using a jq expression
      --json fields            output JSON with the specified fields (comma-separated; --json alone lists them)
      --repo strings           limit to these repos (owner/name, repeatable)
      --repo-file strings      limit to the repos listed in this file, one per line (- for stdin, repeatable)
  -t, --template string        format JSON output using a Go template; see gh help formatting
      --write-rate float       GitHub API writes per second (default 1)
```

### Options inherited from parent commands

```
      --config string   config file (default $XDG_CONFIG_HOME/gh-fleet/config.yaml)
```

### SEE ALSO

* [gh fleet](gh_fleet.md)	 - Apply one change across many repositories and track the PRs

