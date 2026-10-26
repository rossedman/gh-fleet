---
title: license-year
---

# license-year

Extend the copyright year in every MIT and Apache licensed repo.

**Uses:** [`license.detect`](../starlark/license.md#detect), [`license.set_year`](../starlark/license.md#set_year)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from license-year --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//license.star", "detect", "set_year")

    def change(repo):
        spdx = detect(repo)
        repo.set_var("license", spdx or "unknown")
        if spdx in ("MIT", "Apache-2.0"):
            set_year(repo, env["YEAR"])
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: license-year-2026
    description: Extend the copyright year in every MIT and Apache licensed repo

    targets:
      orgs: [acme]
      exclude:
        topics: [no-fleet]

    change:
      steps:
        # Extends `Copyright (c) 2021 Acme` to `2021-2026` in the license file of
        # repos the license module recognizes as MIT or Apache-2.0. gh fleet has
        # no clock, so the year comes from env.
        - name: license-year
          starlark: ./change.star
          timeout: 1m
      env:
        YEAR: "2026"

    pr:
      title: "docs: copyright {{ .Vars.license }} license through 2026"
      body: |
        Extends the copyright notice in {{ .Repo.Name }}'s license to 2026.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet, docs]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/license-year)
