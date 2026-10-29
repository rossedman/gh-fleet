# gh fleet apply

Run the change, push branches, and open or update PRs

### Synopsis

Run the change in every target, push the campaign branch and open a PR, or
update the existing one. Running apply again never duplicates PRs: it updates
the ones that changed.

A merged PR's repo is skipped. A PR closed without merging is left alone
unless --reopen is passed. When the change is already on the base branch and
a PR is still open, --close-noop closes it.

A repo that the last plan or apply (within a day) found nothing to change in
is skipped without cloning, as long as it has no campaign PR, its base branch
hasn't moved and the campaign's files are the same. --recheck runs every repo,
for a change that depends on something outside the repo, such as the latest
version of a tool.

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
      --concurrency int        repos cloned and run in parallel (default limits.concurrency, else 16)
      --exclude strings        leave out these repos (owner/name, repeatable)
      --exclude-file strings   leave out the repos listed in this file, one per line (- for stdin, repeatable)
  -f, --file string            campaign file (default ./fleet.yaml)
  -h, --help                   help for apply
  -q, --jq expression          filter JSON output using a jq expression
      --json fields            output JSON with the specified fields (comma-separated; --json alone lists them)
      --keep-workdirs          leave clones on disk for debugging
      --no-git-cache           clone every repo from GitHub instead of through the git cache kept between runs
      --recheck                run the change even in repos the last run found nothing to change in
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
      --debug           log what gh fleet is doing and print how long each part took
```

### SEE ALSO

* [gh fleet](gh_fleet.md)	 - Apply one change across many repositories and track the PRs

