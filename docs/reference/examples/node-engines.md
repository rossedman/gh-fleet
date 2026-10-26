---
title: node-engines
---

# node-engines

Require Node 22 in every package.json.

**Uses:** [`npm.packages`](../starlark/npm.md#packages), [`npm.read_package`](../starlark/npm.md#read_package), [`npm.write_package`](../starlark/npm.md#write_package)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from node-engines --org your-org
```

## Campaign

=== "change.star"

    ```python
    # Require Node >= env["NODE"] in every package.json outside node_modules.
    load("@fleet//npm.star", "packages", "read_package", "write_package")

    def required_major(spec):
        """Returns the lowest major version a range like ">=18" or "^20.1" allows, or None."""
        m = re.groups(r"(\d+)", spec)
        return int(m[0]) if m else None

    def change(repo):
        want = int(env["NODE"])
        repo.set_var("node", want)
        previous = []
        for path in packages(repo):
            pkg = read_package(repo, path)
            engines = pkg.get("engines", {})
            current = engines.get("node", "")
            have = required_major(current)
            if have != None and have >= want:
                continue  # already requires this version or newer
            previous.append(current or "unset")
            engines["node"] = ">=%d" % want
            pkg["engines"] = engines
            write_package(repo, path, pkg)
        repo.set_var("previous", ", ".join(previous) or "already set")
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: node-engines
    description: Require Node 22 in every package.json

    targets:
      orgs: [acme]
      include:
        files: [package.json]

    change:
      steps:
        # A Starlark change: it edits package.json as data, keeps its key order,
        # never lowers a newer requirement, and tells the PR what it replaced.
        # No shell, no network: it can only touch files in the clone.
        - name: engines
          starlark: ./change.star
          timeout: 1m
      env:
        NODE: "22"

    pr:
      title: "build: require Node {{ .Vars.node }}"
      body: |
        Sets `engines.node` to `>={{ .Vars.node }}` in {{ .Repo.Name }}
        (was `{{ .Vars.previous }}`), so installs on older Node fail early with a
        clear message instead of at runtime.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet, dependencies]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/node-engines)
