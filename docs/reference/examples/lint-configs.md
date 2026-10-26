---
title: lint-configs
---

# lint-configs

Relax line-length rules and silence noisy Dockerfile checks.

**Uses:** [`lintconfig.hadolint_ignore`](../starlark/lintconfig.md#hadolint_ignore), [`lintconfig.markdownlint_rule`](../starlark/lintconfig.md#markdownlint_rule), [`lintconfig.yamllint_rule`](../starlark/lintconfig.md#yamllint_rule)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from lint-configs --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//lintconfig.star", "hadolint_ignore", "markdownlint_rule", "yamllint_rule")

    def change(repo):
        markdownlint_rule(repo, "MD013", False)
        yamllint_rule(repo, "line-length", {"max": 120})
        if repo.glob("**/Dockerfile"):
            hadolint_ignore(repo, ["DL3008", "DL3013"])
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: lint-defaults
    description: Relax line-length rules and silence noisy Dockerfile checks

    targets:
      orgs: [acme]
      exclude:
        topics: [no-fleet]

    change:
      steps:
        # Turns off markdownlint's MD013 line length and raises yamllint's to 120
        # where those tools are configured, and ignores hadolint's apt and pip pin
        # rules (DL3008, DL3013) in repos that have a Dockerfile.
        - name: lint-configs
          starlark: ./change.star
          timeout: 1m

    pr:
      title: "chore: lint rule defaults"
      body: |
        Applies the org's lint rule defaults to {{ .Repo.Name }}.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/lint-configs)
