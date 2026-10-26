---
title: jenkins-shared-library
---

# jenkins-shared-library

Move every Jenkinsfile and CircleCI config to the shared CI library v3.

**Uses:** [`circleci.bump_orb`](../starlark/circleci.md#bump_orb), [`jenkins.bump_library`](../starlark/jenkins.md#bump_library)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from jenkins-shared-library --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//circleci.star", "bump_orb")
    load("@fleet//jenkins.star", "bump_library")

    def change(repo):
        bump_library(repo, "ci-lib", env["VERSION"])
        bump_orb(repo, "acme/ci-orb", env["ORB"])
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: ci-lib-v3
    description: Move every Jenkinsfile and CircleCI config to the shared CI library v3

    targets:
      orgs: [acme]
      include:
        search: "ci-lib@ OR acme/ci-orb@"

    change:
      steps:
        # Moves a Jenkins shared library (`@Library('ci-lib@v2')`) and a
        # CircleCI orb (`acme/ci-orb@2.1.0`) in one step. Both modules are built
        # on `refs.bump`, which works on any `name@ref` in any file.
        - name: jenkins-shared-library
          starlark: ./change.star
          timeout: 1m
      env:
        VERSION: v3.0.0
        ORB: "3.0.0"

    pr:
      title: "ci: shared CI library v3"
      body: |
        Moves {{ .Repo.Name }} to ci-lib v3.0.0 and acme/ci-orb 3.0.0.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/jenkins-shared-library)
