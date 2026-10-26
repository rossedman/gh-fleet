---
title: node-pnpm-9
---

# node-pnpm-9

Move pnpm projects to pnpm 9 and strict engines.

**Uses:** [`npm.read_package`](../starlark/npm.md#read_package), [`npm.set_npmrc`](../starlark/npm.md#set_npmrc), [`npm.set_package_manager`](../starlark/npm.md#set_package_manager)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from node-pnpm-9 --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//npm.star", "read_package", "set_npmrc", "set_package_manager")

    def change(repo):
        current = read_package(repo).get("packageManager") or ""
        if not current.startswith("pnpm@") or semver.lowers(current, env["PNPM"]):
            return  # not pinned to pnpm, or already on a newer pnpm
        repo.set_var("pnpm", env["PNPM"])
        set_package_manager(repo, env["PNPM"])
        set_npmrc(repo, "engine-strict", True)
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: pnpm-9
    description: Move pnpm projects to pnpm 9 and strict engines

    targets:
      orgs: [acme]
      include:
        files: [pnpm-lock.yaml]

    change:
      steps:
        # Sets packageManager (used by Corepack) to pnpm 9 where it pins an
        # older pnpm, and turns on engine-strict in .npmrc so installs fail on the
        # wrong Node version. Repos on a newer pnpm are left alone.
        - name: node-pnpm-9
          starlark: ./change.star
          timeout: 1m
      env:
        PNPM: pnpm@9.12.0

    pr:
      title: "build: pnpm 9"
      body: |
        Moves {{ .Repo.Name }} to {{ .Vars.pnpm }}.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet, dependencies]
    ```

## Test cases

`gh fleet test` runs the change on each case's `before/` files and checks the result against `after/`. See [Testing a campaign](../../guides/testing.md).

### already-newer

=== "Before"

    `package.json`

    ```json
    {
      "private": true,
      "packageManager": "pnpm@12.11.2",
      "type": "module"
    }
    ```

=== "After"

    No changes: the campaign leaves this repo alone.

### pnpm-8

=== "Before"

    `package.json`

    ```json
    {
      "name": "web",
      "private": true,
      "packageManager": "pnpm@8.15.4",
      "scripts": {"build": "vite build"},
      "workspaces": ["packages/*"]
    }
    ```

=== "After"

    `.npmrc`

    ```text
    engine-strict=true
    ```

    `package.json`

    ```json
    {
      "name": "web",
      "private": true,
      "packageManager": "pnpm@9.12.0",
      "scripts": {"build": "vite build"},
      "workspaces": ["packages/*"]
    }
    ```

### yarn

=== "Before"

    `package.json`

    ```json
    {"name": "app", "packageManager": "yarn@4.5.0"}
    ```

=== "After"

    No changes: the campaign leaves this repo alone.

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/node-pnpm-9)
