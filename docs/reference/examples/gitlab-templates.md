---
title: gitlab-templates
---

# gitlab-templates

Move every project to the shared CI templates and components v2.

**Uses:** [`gitlab.bump_component`](../starlark/gitlab.md#bump_component), [`gitlab.set_default`](../starlark/gitlab.md#set_default), [`gitlab.set_include_ref`](../starlark/gitlab.md#set_include_ref)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from gitlab-templates --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//gitlab.star", "bump_component", "set_default", "set_include_ref")

    def change(repo):
        set_include_ref(repo, env["TEMPLATES"], env["REF"])
        bump_component(repo, env["COMPONENTS"], env["VERSION"])
        set_default(repo, "interruptible", True)
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: gitlab-ci-templates-v2
    description: Move every project to the shared CI templates and components v2

    targets:
      orgs: [acme]
      include:
        files: [.gitlab-ci.yml]

    change:
      steps:
        # Sets the ref of `include: project: acme/ci-templates`, moves every
        # acme CI/CD component to 2.0.0, and makes jobs interruptible by default
        # so new pushes cancel stale pipelines.
        - name: gitlab-templates
          starlark: ./change.star
          timeout: 1m
      env:
        TEMPLATES: acme/ci-templates
        REF: v2.0.0
        COMPONENTS: "gitlab.com/acme/components/*"
        VERSION: "2.0.0"

    pr:
      title: "ci: shared templates and components v2"
      body: |
        Moves {{ .Repo.Name }} to the v2 CI templates and components, and makes
        jobs interruptible by default.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet]
    ```

## Test cases

`gh fleet test` runs the change on each case's `before/` files and checks the result against `after/`. See [Testing a campaign](../../guides/testing.md).

### spec-header

=== "Before"

    `.gitlab-ci.yml`

    ```yaml
    spec:
      inputs:
        release:
          type: boolean
          default: false
    ---
    variables:
      GO_VERSION: "1.23"
    test:
      script: go test ./...
    ```

=== "After"

    `.gitlab-ci.yml`

    ```yaml
    spec:
      inputs:
        release:
          type: boolean
          default: false
    ---
    variables:
      GO_VERSION: "1.23"
    test:
      script: go test ./...
    default:
      interruptible: true
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/gitlab-templates)
