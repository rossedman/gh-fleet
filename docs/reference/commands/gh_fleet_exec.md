# gh fleet exec

Run a read-only command across targets and aggregate the output

### Synopsis

Clone every target, run a command in it and tally the output lines across
repos. Nothing is pushed and no PRs are opened.

Targets come from ad-hoc flags (--org, --topic, --language, --file, --search,
--repo, --repo-file) when any is given, otherwise from the campaign (--campaign, default
./fleet.yaml). Repos you can read but not push to are included. The command
runs directly, without a shell; wrap it in sh -c for pipes.

```
gh fleet exec -- <command> [args...] [flags]
```

### Examples

```
  gh fleet exec --org acme --file Dockerfile -- grep -h '^FROM' Dockerfile
  gh fleet exec --org acme --language Go -- sh -c "grep '^go ' go.mod"
  gh fleet exec --raw -- git log -1 --format=%cs
```

### Options

```
      --campaign string        take targets from this campaign file or name (default ./fleet.yaml)
      --concurrency int        repos cloned and run in parallel (default 16)
      --exclude strings        leave out these repos (owner/name, repeatable)
      --exclude-file strings   leave out the repos listed in this file, one per line (- for stdin, repeatable)
      --file strings           only repos where any of these paths exist
  -h, --help                   help for exec
  -q, --jq expression          filter JSON output using a jq expression
      --json fields            output JSON with the specified fields (comma-separated; --json alone lists them)
      --keep-workdirs          leave clones on disk for debugging
      --language strings       only repos with any of these primary languages
      --no-git-cache           clone every repo from GitHub instead of through the git cache kept between runs
      --org strings            target this org's repos (repeatable)
      --raw                    print each repo's output instead of a tally
      --repo strings           target these repos (owner/name, repeatable); with a campaign, limit to them
      --repo-file strings      target the repos listed in this file, one per line (- for stdin, repeatable); with a campaign, limit to them
      --search string          only repos with a hit for this code search query
  -t, --template string        format JSON output using a Go template; see gh help formatting
      --timeout duration       time limit per repo (default 5m0s)
      --topic strings          only repos with any of these topics
```

### Options inherited from parent commands

```
      --config string   config file (default $XDG_CONFIG_HOME/gh-fleet/config.yaml)
      --debug           log what gh fleet is doing and print how long each part took
```

### SEE ALSO

* [gh fleet](gh_fleet.md)	 - Apply one change across many repositories and track the PRs

