# gh fleet apply

Run the change, push branches, and open or update PRs

### Synopsis

Run the change in every target, push the campaign branch and open a PR, or
update the existing one. Running apply again never duplicates PRs: it updates
the ones that changed.

A merged PR's repo is skipped. A PR closed without merging is left alone
unless --reopen is passed. When the change is already on the base branch and
a PR is still open, --close-noop closes it.

apply asks for confirmation before pushing anything unless --yes is passed.

```
gh fleet apply [campaign] [flags]
```

### Examples

```
  gh fleet apply
  gh fleet apply --repo acme/api --yes
  gh fleet apply --close-noop
```

### Options

```
      --close-noop             close open PRs whose change is already on the base branch
      --concurrency int        repos processed in parallel (default limits.concurrency, else 8)
      --exclude strings        leave out these repos (owner/name, repeatable)
      --exclude-file strings   leave out the repos listed in this file, one per line (- for stdin, repeatable)
  -f, --file string            campaign file (default ./fleet.yaml)
  -h, --help                   help for apply
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

