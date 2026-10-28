# gh fleet list

List campaigns in local state

### Synopsis

List every campaign with local state under $XDG_STATE_HOME/gh-fleet, with its
last update, per-status repo counts and campaign file. A listed name can be
passed to any command in place of --file.

```
gh fleet list [flags]
```

### Examples

```
  gh fleet list
  gh fleet list --json name --jq '.[].name'
```

### Options

```
  -h, --help              help for list
  -q, --jq expression     filter JSON output using a jq expression
      --json fields       output JSON with the specified fields (comma-separated; --json alone lists them)
  -t, --template string   format JSON output using a Go template; see gh help formatting
```

### Options inherited from parent commands

```
      --config string   config file (default $XDG_CONFIG_HOME/gh-fleet/config.yaml)
      --debug           log what gh fleet is doing and print how long each part took
```

### SEE ALSO

* [gh fleet](gh_fleet.md)	 - Apply one change across many repositories and track the PRs

