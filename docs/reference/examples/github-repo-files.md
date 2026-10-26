---
title: github-repo-files
---

# github-repo-files

Standard release notes, labels and issue links for every repo.

**Uses:** [`github.add_contact_link`](../starlark/github.md#add_contact_link), [`github.add_release_category`](../starlark/github.md#add_release_category), [`github.set_label`](../starlark/github.md#set_label)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from github-repo-files --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//github.star", "add_contact_link", "add_release_category", "set_label")

    def change(repo):
        add_release_category(repo, "Security", ["security"])
        add_release_category(repo, "Dependencies", ["dependencies"])
        add_release_category(repo, "Other changes", ["*"])
        set_label(repo, "fleet", "1f6feb", description = "Opened by gh fleet")
        add_contact_link(
            repo,
            "Report a vulnerability",
            "https://github.com/%s/security/advisories/new" % repo.full_name,
            "Please report security issues privately.",
        )
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: github-repo-defaults
    description: Standard release notes, labels and issue links for every repo

    targets:
      orgs: [acme]
      exclude:
        topics: [no-fleet]

    change:
      steps:
        # Adds release note categories (dependencies and security before the
        # catch-all), a fleet label, and a security contact link on the new issue
        # page, keeping whatever each repo already has.
        - name: github-repo-files
          starlark: ./change.star
          timeout: 1m

    pr:
      title: "chore: release notes, labels and issue links"
      body: |
        Adds the org's standard release note categories, labels and new-issue
        links to {{ .Repo.Name }}.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/github-repo-files)
