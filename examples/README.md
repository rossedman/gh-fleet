# Example campaigns

Three campaigns you can copy and adapt. Each directory holds the campaign
file, the change script it runs and, where the body is long, the PR body.
Change `orgs: [acme]` to your organization, then:

```sh
cd examples/pin-actions
gh fleet targets
gh fleet plan --repo your-org/some-repo
```

| Campaign | What it changes | Shows |
| --- | --- | --- |
| [`pin-actions`](pin-actions/) | Pins every `uses:` in `.github/workflows` to a commit SHA, keeping the tag as a comment | a script file, a body file, `files` targeting, `maxOpenPRs`, signed-off commits |
| [`add-dependabot`](add-dependabot/) | Writes `.github/dependabot.yml` for the ecosystems a repo uses | topic targeting, an inline body, team reviewers, no-op repos |
| [`bump-go-version`](bump-go-version/) | Raises the `go` directive in `go.mod` and `go-version` pins in workflows | language plus file targeting, `exclude.repos`, `env`, draft PRs |

The scripts are tested in `examples_test.go`.
