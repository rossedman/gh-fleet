# gh fleet

Apply one change across many repositories and track the PRs

### Synopsis

gh fleet applies one change across many repositories and tracks the resulting
pull requests to completion.

A change is described once as a campaign: which repos to target, what to run in
each, and how to open the PR. Re-running a campaign updates its PRs instead of
duplicating them.

### Examples

```
  gh fleet init pin-actions
  gh fleet targets
  gh fleet plan --repo acme/api
  gh fleet apply
  gh fleet status
```

### Options

```
      --config string   config file (default $XDG_CONFIG_HOME/gh-fleet/config.yaml)
  -h, --help            help for gh fleet
```

### SEE ALSO

* [gh fleet apply](gh_fleet_apply.md)	 - Run the change, push branches, and open or update PRs
* [gh fleet close](gh_fleet_close.md)	 - Close all open campaign PRs and delete their branches
* [gh fleet exec](gh_fleet_exec.md)	 - Run a read-only command across targets and aggregate the output
* [gh fleet init](gh_fleet_init.md)	 - Scaffold a campaign file with commented defaults
* [gh fleet lib](gh_fleet_lib.md)	 - Browse the built-in Starlark library: modules, functions and examples
* [gh fleet list](gh_fleet_list.md)	 - List campaigns in local state
* [gh fleet merge](gh_fleet_merge.md)	 - Merge campaign PRs that are approved and passing, and say what the rest wait on
* [gh fleet plan](gh_fleet_plan.md)	 - Run the change in fresh clones and show per-repo diffs, without pushing
* [gh fleet retry](gh_fleet_retry.md)	 - Run apply again for repos that failed or have no PR
* [gh fleet status](gh_fleet_status.md)	 - Show every target's PR, state, checks, reviews and last error
* [gh fleet targets](gh_fleet_targets.md)	 - List the repos a campaign will touch and why each matched
* [gh fleet test](gh_fleet_test.md)	 - Run the change against local test cases, without GitHub
* [gh fleet version](gh_fleet_version.md)	 - Print the version, commit and build date

