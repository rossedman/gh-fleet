---
title: terragrunt-module-ref
---

# terragrunt-module-ref

Move every Terragrunt and Terraform stack to the shared modules v2.0.0.

**Uses:** [`terragrunt.set_source_ref`](../starlark/terragrunt.md#set_source_ref), [`terragrunt.sources`](../starlark/terragrunt.md#sources)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from terragrunt-module-ref --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//terragrunt.star", "set_source_ref", "sources")

    def change(repo):
        old = sorted(set([s["ref"] for s in sources(repo) if s["source"].startswith(env["SOURCE"].rstrip("*"))]))
        repo.set_var("previous", ", ".join(old))
        set_source_ref(repo, env["SOURCE"], env["REF"])
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: modules-v2
    description: Move every Terragrunt and Terraform stack to the shared modules v2.0.0

    targets:
      orgs: [acme]
      include:
        search: "acme/terraform-modules ref="

    change:
      steps:
        # Sets `?ref=` on every module source from the shared modules repo, in
        # terragrunt.hcl and .tf files alike. Sources pinned to a commit SHA, and
        # the .terragrunt-cache and .terraform directories, are left alone.
        - name: terragrunt-module-ref
          starlark: ./change.star
          timeout: 1m
      env:
        SOURCE: "git::https://github.com/acme/terraform-modules.git//*"
        REF: v2.0.0

    pr:
      title: "build(deps): terraform modules v2.0.0"
      body: |
        Moves {{ .Repo.Name }} to acme/terraform-modules v2.0.0. Review the plan
        before merging.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet, dependencies]
      draft: true
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/terragrunt-module-ref)
