# gh fleet init

Scaffold a campaign file with commented defaults

### Synopsis

Write a new campaign file with every option shown and commented.

The name defaults to the current directory's name. The file starts with a
yaml-language-server schema header, so editors validate and complete it.

```
gh fleet init [name] [flags]
```

### Examples

```
  gh fleet init pin-actions
  gh fleet init bump-go --org acme --file campaigns/bump-go.yaml
```

### Options

```
  -f, --file string   where to write the campaign (default ./fleet.yaml)
      --force         overwrite an existing file
  -h, --help          help for init
      --org string    organization to target
```

### Options inherited from parent commands

```
      --config string   config file (default $XDG_CONFIG_HOME/gh-fleet/config.yaml)
```

### SEE ALSO

* [gh fleet](gh_fleet.md)	 - Apply one change across many repositories and track the PRs

