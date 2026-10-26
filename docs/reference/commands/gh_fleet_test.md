# gh fleet test

Run the change against local test cases, without GitHub

### Synopsis

Run the change against the test cases in tests/ next to the campaign file and
check the result. Nothing is cloned and nothing touches GitHub.

Each case is a directory:

  tests/<case>/before/     the repo's files before the change
  tests/<case>/after/      the files the change must produce (omit when the
                           change should leave the repo alone)
  tests/<case>/case.yaml   optional: repo, defaultBranch, language, topics,
                           properties, vars and env to run as, and wantVars,
                           wantTitle or wantError to check

Every step runs, shell steps included, in a git repository holding before/.
The case passes when the files match after/ exactly and a second run changes
nothing. --update writes after/ from what the change produced.

--dir runs the change on a copy of a local directory instead, such as a clone
you already have, and prints the diff, variables and PR title.

```
gh fleet test [campaign] [flags]
```

### Examples

```
  gh fleet test
  gh fleet test --run expired -v
  gh fleet test --update
  gh fleet test --dir ../billing-api --repo acme/billing-api
```

### Options

```
      --dir string        run the change on a copy of this directory and print the diff
  -f, --file string       campaign file (default ./fleet.yaml)
  -h, --help              help for test
  -q, --jq expression     filter JSON output using a jq expression
      --json fields       output JSON with the specified fields (comma-separated; --json alone lists them)
      --repo strings      with --dir, the owner/name the change runs as (default acme/<dir name>)
      --run string        only run cases whose name matches this regular expression
  -t, --template string   format JSON output using a Go template; see gh help formatting
      --update            rewrite each case's after/ from the change's result
  -v, --verbose           print each case's step output and PR title
```

### Options inherited from parent commands

```
      --config string   config file (default $XDG_CONFIG_HOME/gh-fleet/config.yaml)
```

### SEE ALSO

* [gh fleet](gh_fleet.md)	 - Apply one change across many repositories and track the PRs

