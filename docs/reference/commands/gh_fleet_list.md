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
  gh fleet list --json --jq '.[].name'
```

### Options

```
  -h, --help        help for list
      --jq string   filter JSON output with a jq expression
      --json        print JSON instead of a table
```

### Options inherited from parent commands

```
      --config string   config file (default $XDG_CONFIG_HOME/gh-fleet/config.yaml)
```

### SEE ALSO

* [gh fleet](gh_fleet.md)	 - Apply one change across many repositories and track the PRs

