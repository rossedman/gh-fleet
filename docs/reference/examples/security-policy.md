---
title: security-policy
---

# security-policy

Give every repo a SECURITY.md.

**Uses:** [`files.write_if_missing`](../starlark/files.md#write_if_missing)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from security-policy --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//files.star", "write_if_missing")

    def change(repo):
        # The template ships with the campaign; campaign.read can't reach anything else.
        policy = campaign.read("SECURITY.md.tmpl").replace("{repo}", repo.full_name)
        write_if_missing(repo, "SECURITY.md", policy, also = [".github/SECURITY.md", "docs/SECURITY.md"])
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: security-policy
    description: Give every repo a SECURITY.md

    targets:
      orgs: [acme]
      exclude:
        topics: [no-fleet]

    change:
      steps:
        # Writes SECURITY.md from the template next to this file, filled in
        # for each repo, unless the repo already has a policy in one of the
        # places GitHub looks.
        - name: policy
          starlark: ./change.star
          timeout: 1m

    pr:
      title: "docs: add a security policy"
      body: |
        Adds a SECURITY.md to {{ .Repo.Name }} so GitHub shows people how to
        report a vulnerability privately.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet, security]
    ```

=== "SECURITY.md.tmpl"

    ```text
    # Security policy

    ## Reporting a vulnerability

    Please do not open a public issue. Report it privately through
    [GitHub's private vulnerability reporting](https://github.com/{repo}/security/advisories/new)
    or by email to security@acme.dev.

    We acknowledge reports within two working days and keep you updated until
    the issue is fixed.

    ## Supported versions

    Only the latest release receives security fixes.
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/security-policy)
