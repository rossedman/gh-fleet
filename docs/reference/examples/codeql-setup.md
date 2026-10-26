---
title: codeql-setup
---

# codeql-setup

Turn on CodeQL code scanning, including GitHub Actions workflows.

**Uses:** [`codeql.add_language`](../starlark/codeql.md#add_language), [`codeql.add_workflow`](../starlark/codeql.md#add_workflow), [`codeql.languages`](../starlark/codeql.md#languages), [`codeql.set_version`](../starlark/codeql.md#set_version)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from codeql-setup --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//codeql.star", "add_language", "add_workflow", "languages", "set_version")

    # GitHub's name for a repo's main language, mapped to CodeQL's.
    CODEQL = {
        "C#": "csharp",
        "C++": "c-cpp",
        "Go": "go",
        "Java": "java-kotlin",
        "JavaScript": "javascript-typescript",
        "Kotlin": "java-kotlin",
        "Python": "python",
        "Ruby": "ruby",
        "Swift": "swift",
        "TypeScript": "javascript-typescript",
    }

    def change(repo):
        main = CODEQL.get(repo.language)
        added = add_workflow(repo, ([main] if main else []) + ["actions"])
        if not added:
            add_language(repo, "actions")
            set_version(repo, "v4")
        repo.set_var("added", added)
        repo.set_var("languages", ", ".join(languages(repo)))
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: codeql-everywhere
    description: Turn on CodeQL code scanning, including GitHub Actions workflows

    targets:
      orgs: [acme]
      include:
        files: [.github/workflows]
      exclude:
        topics: [no-fleet]

    change:
      steps:
        # Repos without CodeQL get a workflow for their main language and their
        # Actions workflows. Repos that already run it get `actions` added to
        # the matrix and codeql-action moved to v4.
        - name: codeql
          starlark: ./change.star
          timeout: 1m

    pr:
      title: "ci: CodeQL code scanning"
      body: |
        {{ if .Vars.added -}}
        Adds a CodeQL workflow scanning {{ .Vars.languages }} on pushes, pull
        requests and weekly.
        {{- else -}}
        Adds `actions` to {{ .Repo.Name }}'s CodeQL matrix, so workflow files
        are scanned too, and moves `github/codeql-action` to v4.
        {{- end }}

        Results show up under Security → Code scanning.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet, security]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/codeql-setup)
