---
title: Examples
---

# Examples

Three complete campaigns you can copy and adapt. Each directory in [`examples/`](https://github.com/rossedman/gh-fleet/tree/main/examples) holds the campaign file, the script it runs and, where the body is long, the PR body. The scripts are tested in `examples_test.go`.

Change `orgs: [acme]` to your organization, then:

```sh
cd examples/pin-actions
gh fleet targets
gh fleet plan --repo your-org/some-repo
```

| Campaign | What it changes | Shows |
| --- | --- | --- |
| [`pin-actions`](https://github.com/rossedman/gh-fleet/tree/main/examples/pin-actions) | Pins every `uses:` in `.github/workflows` to a commit SHA, keeping the tag as a comment | a script file, a body file, `files` targeting, `maxOpenPRs`, signed-off commits |
| [`add-dependabot`](https://github.com/rossedman/gh-fleet/tree/main/examples/add-dependabot) | Writes `.github/dependabot.yml` for the ecosystems a repo uses | topic targeting, an inline body, team reviewers, no-op repos |
| [`bump-go-version`](https://github.com/rossedman/gh-fleet/tree/main/examples/bump-go-version) | Raises the `go` directive in `go.mod` and `go-version` pins in workflows | language plus file targeting, `exclude.repos`, `env`, draft PRs |

## pin-actions

Resolves each `uses: owner/repo@ref` to a commit SHA with `gh api` and keeps the ref as a comment, so Dependabot can still update it.

```yaml
targets:
  orgs: [acme]
  include:
    files: [.github/workflows]
  exclude:
    topics: [no-fleet]

change:
  run: ./scripts/pin.sh
  timeout: 5m

pr:
  title: "ci: pin GitHub Actions to commit SHAs"
  body: ./pr-body.md
  labels: [fleet, security]
  commit:
    signoff: true

limits:
  concurrency: 8
  maxOpenPRs: 200
```

## add-dependabot

Writes a Dependabot config for the ecosystems it detects. Repos that already have one are left alone, so they come out as `noop`.

```yaml
targets:
  orgs: [acme]
  include:
    topics: [service]

change:
  run: ./scripts/dependabot.sh
  timeout: 1m

pr:
  title: "build: add Dependabot config"
  labels: [fleet, dependencies]
  reviewers: [acme/platform]
```

## bump-go-version

Raises the `go` directive and the toolchain line (never lowers them), and bumps `go-version` in workflows that pin one. The target version comes from `change.env`.

```yaml
targets:
  orgs: [acme]
  include:
    languages: [Go]
    files: [go.mod]
  exclude:
    repos: [acme/legacy-api]

change:
  run: ./scripts/bump-go.sh
  env:
    GO_VERSION: "1.27.3"
  timeout: 2m

pr:
  title: "build: go 1.27"
  draft: true
  labels: [fleet]

limits:
  concurrency: 16
```
