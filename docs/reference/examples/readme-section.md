---
title: readme-section
---

# readme-section

Give every README a Security section and a CI badge.

**Uses:** [`markdown.ensure_badge`](../starlark/markdown.md#ensure_badge), [`markdown.set_section`](../starlark/markdown.md#set_section)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from readme-section --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//markdown.star", "ensure_badge", "set_section")

    SECURITY = """Please report vulnerabilities privately to security@acme.dev, or through
    [GitHub's private reporting](https://github.com/%s/security/advisories/new).
    Do not open a public issue."""

    def change(repo):
        set_section(repo, "README.md", "Security", SECURITY % repo.full_name)
        url = "https://github.com/%s/actions/workflows/ci.yml" % repo.full_name
        ensure_badge(repo, "README.md", "[![CI](%s/badge.svg)](%s)" % (url, url))
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: readme-security
    description: Give every README a Security section and a CI badge

    targets:
      orgs: [acme]
      include:
        files: [README.md]

    change:
      steps:
        # Writes the Security section (replacing an old one) and adds a CI
        # badge under the title, using the repo's own name in the badge URL.
        - name: readme
          starlark: ./change.star
          timeout: 1m

    pr:
      title: "docs: security contact and CI badge in README"
      body: |
        Adds the standard Security section to {{ .Repo.Name }}'s README so
        people know where to report vulnerabilities, and a CI status badge.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet, docs]
    ```

## Test cases

`gh fleet test` runs the change on each case's `before/` files and checks the result against `after/`. See [Testing a campaign](../../guides/testing.md).

### logo-and-badges

=== "Before"

    `README.md`

    ```markdown
    ![Logo](docs/logo.png)

    The open-source platform for monitoring

    [![License](https://img.shields.io/github/license/acme/api)](LICENSE)

    Intro.

    ## Get started

    - [Get it](https://acme.dev/get)

    ## License

    AGPL-3.0
    ```

=== "After"

    `README.md`

    ```markdown
    ![Logo](docs/logo.png)

    The open-source platform for monitoring

    [![License](https://img.shields.io/github/license/acme/api)](LICENSE)
    [![CI](https://github.com/acme/api/actions/workflows/ci.yml/badge.svg)](https://github.com/acme/api/actions/workflows/ci.yml)

    Intro.

    ## Get started

    - [Get it](https://acme.dev/get)

    ## License

    AGPL-3.0

    ## Security

    Please report vulnerabilities privately to security@acme.dev, or through
    [GitHub's private reporting](https://github.com/acme/api/security/advisories/new).
    Do not open a public issue.
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/readme-section)
