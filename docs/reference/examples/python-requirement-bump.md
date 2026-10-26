---
title: python-requirement-bump
---

# python-requirement-bump

Move every Django service to the 4.2.16 security release.

**Uses:** [`python.set_requirement`](../starlark/python.md#set_requirement)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from python-requirement-bump --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//python.star", "set_requirement")

    def change(repo):
        changed = set_requirement(repo, env["PACKAGE"], env["SPEC"])
        repo.set_var("files", ", ".join(changed))
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: bump-django
    description: Move every Django service to the 4.2.16 security release

    targets:
      orgs: [acme]
      include:
        languages: [Python]
        search: "django filename:requirements.txt"

    change:
      steps:
        # Sets the pin in requirements*.txt and pyproject.toml wherever Django is
        # listed, keeping extras, markers and comments.
        - name: requirements
          starlark: ./change.star
          timeout: 1m
      env:
        PACKAGE: django
        SPEC: "==4.2.16"

    pr:
      title: "build(deps): django 4.2.16"
      body: |
        Pins Django to 4.2.16 in {{ .Repo.Name }} ({{ .Vars.files }}). The
        release fixes several security issues; no code changes are expected.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet, dependencies, security]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/python-requirement-bump)
