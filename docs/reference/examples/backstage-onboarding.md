---
title: backstage-onboarding
---

# backstage-onboarding

Register every service in the Backstage catalog, owned by the team in its custom property.

**Uses:** [`backstage.add_tag`](../starlark/backstage.md#add_tag), [`backstage.ensure`](../starlark/backstage.md#ensure), [`backstage.set_annotation`](../starlark/backstage.md#set_annotation)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from backstage-onboarding --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//backstage.star", "add_tag", "ensure", "set_annotation")

    def change(repo):
        team = repo.properties.get("team")
        owner = "group:" + team if team else env["FALLBACK_OWNER"]
        ensure(repo, owner)
        if repo.exists("mkdocs.yml"):
            set_annotation(repo, "backstage.io/techdocs-ref", "dir:.", kind = "Component", only_missing = True)
        if repo.language:
            add_tag(repo, repo.language.lower().replace(" ", "-"), kind = "Component")
        repo.set_var("owner", owner)
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: backstage-onboarding
    description: Register every service in the Backstage catalog, owned by the team in its custom property

    targets:
      orgs: [acme]
      exclude:
        topics: [no-fleet]

    change:
      steps:
        # Writes catalog-info.yaml where it's missing, with the owner from the
        # repo's `team` custom property. Repos with docs get the TechDocs
        # annotation and every component gets a tag for its language.
        - name: catalog
          starlark: ./change.star
          timeout: 1m
      env:
        FALLBACK_OWNER: group:platform

    pr:
      title: "chore: register in the Backstage catalog"
      body: |
        Registers {{ .Repo.Name }} in Backstage, owned by `{{ .Vars.owner }}`.
        Check the owner, type and lifecycle in `catalog-info.yaml`; they're
        best guesses.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/backstage-onboarding)
