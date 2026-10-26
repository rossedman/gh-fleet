# gh fleet retry

Run apply again for repos that failed or have no PR

### Synopsis

Run apply again for the repos whose last run failed, never ran, or changed
but were not pushed (for example because limits.maxOpenPRs was reached).
Repos with a PR, merged or closed PRs and no-op repos are left alone.

```
gh fleet retry [campaign] [flags]
```

### Examples

```
  gh fleet retry
  gh fleet retry --yes
```

### Options

```
      --close-noop             close open PRs whose change is already on the base branch
      --concurrency int        repos processed in parallel (default limits.concurrency, else 8)
      --exclude strings        leave out these repos (owner/name, repeatable)
      --exclude-file strings   leave out the repos listed in this file, one per line (- for stdin, repeatable)
  -f, --file string            campaign file (default ./fleet.yaml)
  -h, --help                   help for retry
  -q, --jq expression          filter JSON output using a jq expression
      --json fields            output JSON with the specified fields (comma-separated; --json alone lists them)
      --keep-workdirs          leave clones on disk for debugging
      --reopen                 reopen campaign PRs that were closed without merging
      --repo strings           limit to these repos (owner/name, repeatable)
      --repo-file strings      limit to the repos listed in this file, one per line (- for stdin, repeatable)
  -t, --template string        format JSON output using a Go template; see gh help formatting
      --write-rate float       GitHub API writes per second (default 1)
  -y, --yes                    skip the confirmation prompt
```

### Options inherited from parent commands

```
      --config string   config file (default $XDG_CONFIG_HOME/gh-fleet/config.yaml)
```

### SEE ALSO

* [gh fleet](gh_fleet.md)	 - Apply one change across many repositories and track the PRs

