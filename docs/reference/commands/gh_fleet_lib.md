# gh fleet lib

Browse the built-in Starlark library: modules, functions and examples

### Synopsis

Print the docs of the @fleet// Starlark library that ships in this binary,
so they always match the version you run.

With no arguments, lists the modules. With a module, such as docker, prints
what it does, its load line and its functions. With a function, such as
docker.bump_image, prints its arguments, what it returns and its example,
which the library's tests run, so it shows real output.

--search finds functions whose names or docs mention every word given.
--source prints a module's Starlark source.

```
gh fleet lib [module[.function]] [function] [flags]
```

### Examples

```
  gh fleet lib
  gh fleet lib docker
  gh fleet lib docker.bump_image
  gh fleet lib --search "never lower"
  gh fleet lib refs --source
```

### Options

```
  -h, --help              help for lib
  -q, --jq expression     filter JSON output using a jq expression
      --json fields       output JSON with the specified fields (comma-separated; --json alone lists them)
  -s, --search string     list functions whose names or docs mention these words
      --source            print the module's Starlark source
  -t, --template string   format JSON output using a Go template; see gh help formatting
```

### Options inherited from parent commands

```
      --config string   config file (default $XDG_CONFIG_HOME/gh-fleet/config.yaml)
```

### SEE ALSO

* [gh fleet](gh_fleet.md)	 - Apply one change across many repositories and track the PRs

