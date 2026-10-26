---
title: go-module-bump
---

# go-module-bump

Patch golang.org/x/net to a fixed release in every Go module.

**Uses:** [`gomod.outdated`](../starlark/gomod.md#outdated)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from go-module-bump --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//gomod.star", "outdated")

    def change(repo):
        repo.set_var("dirs", outdated(repo, env["MODULE"], env["VERSION"]))
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: bump-x-net
    description: Patch golang.org/x/net to a fixed release in every Go module

    targets:
      orgs: [acme]
      include:
        languages: [Go]
        search: "golang.org/x/net filename:go.mod"

    change:
      steps:
        # Starlark finds the modules that require MODULE below VERSION.
        - name: find modules
          starlark: ./change.star
          timeout: 1m
        # The Go toolchain updates them; the step does nothing when none do.
        - name: go get
          run: |
            set -e
            for dir in ${FLEET_VAR_DIRS//,/ }; do
              (cd "$dir" && go get "$MODULE@$VERSION" && go mod tidy)
            done
          timeout: 10m
      env:
        MODULE: golang.org/x/net
        VERSION: v0.33.0
        GOFLAGS: -mod=mod

    pr:
      title: "build(deps): golang.org/x/net v0.33.0"
      body: |
        Updates golang.org/x/net in {{ .Repo.Name }} to v0.33.0, which fixes a
        known vulnerability. Check `govulncheck ./...` after merging.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet, dependencies, security]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/go-module-bump)
