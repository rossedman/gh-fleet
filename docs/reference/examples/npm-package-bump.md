---
title: npm-package-bump
---

# npm-package-bump

Move axios to 1.x in every package that depends on it.

**Uses:** [`npm.set_dependency`](../starlark/npm.md#set_dependency)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from npm-package-bump --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//npm.star", "set_dependency")

    def change(repo):
        changed = set_dependency(repo, env["PACKAGE"], env["RANGE"])
        dirs = [path.removesuffix("package.json").rstrip("/") or "." for path in changed]
        repo.set_var("dirs", [d for d in dirs if repo.exists(d + "/package-lock.json")])
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: bump-axios
    description: Move axios to 1.x in every package that depends on it

    targets:
      orgs: [acme]
      include:
        files: [package.json]
        search: "axios filename:package.json"

    change:
      steps:
        # Sets PACKAGE to RANGE in every package.json that already lists it,
        # keeping key order and indentation, and records which directories
        # have a lockfile to refresh.
        - name: package.json
          starlark: ./change.star
          timeout: 1m
        # Refreshes those lockfiles without installing or running scripts.
        - name: lockfiles
          run: |
            set -e
            for dir in ${FLEET_VAR_DIRS//,/ }; do
              (cd "$dir" && npm install --package-lock-only --ignore-scripts --no-audit --no-fund)
            done
          timeout: 10m
      env:
        PACKAGE: axios
        RANGE: "^1.7.9"

    pr:
      title: "build(deps): axios ^1.7.9"
      body: |
        Updates axios in {{ .Repo.Name }} to ^1.7.9. Lockfiles were refreshed
        with `npm install --package-lock-only --ignore-scripts`.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet, dependencies]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/npm-package-bump)
