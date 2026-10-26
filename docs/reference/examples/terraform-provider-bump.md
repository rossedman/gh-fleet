---
title: terraform-provider-bump
---

# terraform-provider-bump

Move every Terraform root to the AWS provider 5.x.

**Uses:** [`terraform.set_provider_version`](../starlark/terraform.md#set_provider_version), [`terraform.set_required_version`](../starlark/terraform.md#set_required_version)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from terraform-provider-bump --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//terraform.star", "set_provider_version", "set_required_version")

    def change(repo):
        if set_provider_version(repo, env["PROVIDER"], env["CONSTRAINT"]):
            set_required_version(repo, env["TERRAFORM"])
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: aws-provider-5
    description: Move every Terraform root to the AWS provider 5.x

    targets:
      orgs: [acme]
      include:
        languages: [HCL]
        search: "hashicorp/aws extension:tf"

    change:
      steps:
        # Sets the version constraint of the AWS provider in required_providers
        # and raises required_version, keeping comments and alignment.
        - name: terraform
          starlark: ./change.star
          timeout: 1m
      env:
        PROVIDER: hashicorp/aws
        CONSTRAINT: "~> 5.0"
        TERRAFORM: ">= 1.9"

    pr:
      title: "build(deps): AWS provider ~> 5.0"
      body: |
        Moves {{ .Repo.Name }} to the AWS provider 5.x and Terraform 1.9.
        Run `terraform init -upgrade` and review the plan before merging.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      draft: true
      labels: [fleet, dependencies]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/terraform-provider-bump)
