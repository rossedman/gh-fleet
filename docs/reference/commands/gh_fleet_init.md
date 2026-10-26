# gh fleet init

Scaffold a campaign file with commented defaults

### Synopsis

Write a new campaign file with every option shown and commented.

The name defaults to the current directory's name. The file starts with a
yaml-language-server schema header, so editors validate and complete it.

--from copies one of the tested example campaigns instead: its fleet.yaml,
scripts and test cases, into the directory of --file. A name and --org, when
given, replace the example's name and target org. --list shows the examples.

```
gh fleet init [name] [flags]
```

### Examples

```
  gh fleet init pin-actions
  gh fleet init bump-go --org acme --file campaigns/bump-go.yaml
  gh fleet init --list
  gh fleet init node-22 --from node-version --org acme --file campaigns/node-22/fleet.yaml
```

### Options

```
  -f, --file string   where to write the campaign (default ./fleet.yaml)
      --force         overwrite existing files
      --from string   copy this example campaign, with its scripts and test cases
  -h, --help          help for init
      --list          list the example campaigns --from can copy
      --org string    organization to target
```

### Options inherited from parent commands

```
      --config string   config file (default $XDG_CONFIG_HOME/gh-fleet/config.yaml)
```

### SEE ALSO

* [gh fleet](gh_fleet.md)	 - Apply one change across many repositories and track the PRs

