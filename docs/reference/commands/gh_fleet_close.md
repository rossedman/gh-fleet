# gh fleet close

Close all open campaign PRs and delete their branches

### Synopsis

Close every open PR the campaign opened and delete its branch. Merged and
already closed PRs are left alone. Asks for confirmation unless --yes is
passed.

```
gh fleet close [campaign] [flags]
```

### Examples

```
  gh fleet close
  gh fleet close --repo acme/api --comment "Superseded by #42" --yes
```

### Options

```
      --comment string     comment to post on each PR before closing it
      --concurrency int    repos processed in parallel (default limits.concurrency, else 8)
  -f, --file string        campaign file (default ./fleet.yaml)
  -h, --help               help for close
  -q, --jq expression      filter JSON output using a jq expression
      --json fields        output JSON with the specified fields (comma-separated; --json alone lists them)
      --keep-branches      close PRs but keep their branches
      --repo strings       limit to these repos (owner/name, repeatable)
  -t, --template string    format JSON output using a Go template; see gh help formatting
      --write-rate float   GitHub API writes per second (default 1)
  -y, --yes                skip the confirmation prompt
```

### Options inherited from parent commands

```
      --config string   config file (default $XDG_CONFIG_HOME/gh-fleet/config.yaml)
```

### SEE ALSO

* [gh fleet](gh_fleet.md)	 - Apply one change across many repositories and track the PRs

