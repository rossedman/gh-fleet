---
title: runner-label
---

# runner-label

Pin jobs on ubuntu-latest to ubuntu-24.04.

**Uses:** [`gha.workflows`](../starlark/gha.md#workflows), [`yamlpath.set`](../starlark/yamlpath.md#set), [`yamlpath.values`](../starlark/yamlpath.md#values)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from runner-label --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//gha.star", "workflows")
    load("@fleet//yamlpath.star", "set", "values")

    def change(repo):
        pinned = []
        for path in workflows(repo):
            for key, label in values(repo, path, "jobs.*.runs-on").items():
                if label == env["FROM"]:
                    set(repo, path, key, env["TO"])
                    pinned.append(key.split(".")[1])
        repo.set_var("jobs", ", ".join(pinned) or "no jobs")
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: pin-runner-image
    description: Pin jobs on ubuntu-latest to ubuntu-24.04

    targets:
      orgs: [acme]
      include:
        files: [.github/workflows]

    change:
      steps:
        # Finds every job's runs-on in every workflow and pins the ones on
        # ubuntu-latest, so a runner image upgrade happens in a reviewed PR
        # instead of underneath the build. Comments and quoting are kept.
        - name: runs-on
          starlark: ./change.star
          timeout: 1m
      env:
        FROM: ubuntu-latest
        TO: ubuntu-24.04

    pr:
      title: "ci: pin runners to ubuntu-24.04"
      body: |
        Pins {{ .Vars.jobs }} in {{ .Repo.Name }} from `ubuntu-latest` to
        `ubuntu-24.04`, so the next runner image change is an explicit PR.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/runner-label)
