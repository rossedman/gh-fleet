---
title: pin-actions
---

# pin-actions

Pin every third-party GitHub Action to a full commit SHA.

**Uses:** [`gha.pin`](../starlark/gha.md#pin), [`gha.unpinned`](../starlark/gha.md#unpinned)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from pin-actions --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//gha.star", "unpinned")

    def change(repo):
        repo.set_var("refs", unpinned(repo))
    ```

=== "pin.star"

    ```python
    load("@fleet//gha.star", "pin")

    def change(repo):
        pin(repo, repo.vars.get("shas", ""))
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: pin-actions
    description: Pin every third-party GitHub Action to a full commit SHA

    targets:
      orgs: [acme]
      include:
        files: [.github/workflows]
      exclude:
        topics: [no-fleet]

    change:
      steps:
        # 1. Starlark lists every action used at a tag or branch.
        - name: find refs
          starlark: ./change.star
          timeout: 1m
        # 2. One `gh api` call per ref resolves it to a commit SHA. This is the
        #    only part that needs the network, so it is the only shell.
        - name: resolve
          run: |
            set -e
            shas=""
            for ref in ${FLEET_VAR_REFS//,/ }; do          # ref is owner/repo@tag
              shas="$shas $ref=$(gh api "repos/${ref%@*}/commits/${ref#*@}" --jq .sha)"
            done
            echo "shas=$shas" >> "$FLEET_OUTPUT/vars"
          timeout: 5m
        # 3. Starlark rewrites each `uses:` to the SHA, keeping the tag as a
        #    comment so Dependabot can still update it.
        - name: pin
          starlark: ./pin.star
          timeout: 1m

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

=== "pr-body.md"

    ````markdown
    Pins every third-party action in `{{ .Repo.FullName }}` to a full commit SHA,
    keeping the tag as a trailing comment:

    ```yaml
    uses: actions/checkout@08eba0b27e820071cde6df949e0beb9ba4906955 # v4.3.0
    ```

    A tag can be moved to point at different code; a SHA cannot. Dependabot
    keeps pinned SHAs and their comments up to date.

    Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
    ````

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/pin-actions)
