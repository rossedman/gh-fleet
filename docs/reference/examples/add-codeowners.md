---
title: add-codeowners
---

# add-codeowners

Give every service its owning team.

**Uses:** [`codeowners.find`](../starlark/codeowners.md#find), [`codeowners.set_owners`](../starlark/codeowners.md#set_owners)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from add-codeowners --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//codeowners.star", "find", "set_owners")

    def change(repo):
        if find(repo) != None:
            return  # the repo already says who owns it
        team = "@%s/%s" % (repo.owner, repo.vars["team"])
        set_owners(repo, "*", team, header = "# Owners are requested for review on every pull request.")
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: add-codeowners
    description: Give every service its owning team

    targets:
      orgs: [acme]
      include:
        topics: [service]

    # Each repo's owning team: platform by default, web for frontends, and
    # whatever owners.csv says for the repos it lists.
    vars:
      default:
        team: platform
      match:
        - when: { topics: [frontend] }
          set: { team: web }
      file: ./owners.csv

    change:
      steps:
        # Writes .github/CODEOWNERS assigning the repo to @<org>/<team>, unless
        # the repo already has a CODEOWNERS file in any of the three places
        # GitHub reads. The team comes from vars, as repo.vars["team"].
        - name: codeowners
          starlark: ./change.star
          timeout: 1m

    pr:
      title: "chore: assign {{ .Repo.Name }} to @{{ .Repo.Owner }}/{{ .Vars.team }}"
      body: |
        Assigns {{ .Repo.Name }} to @{{ .Repo.Owner }}/{{ .Vars.team }} so pull
        requests request a review from the owning team automatically. Edit the
        file to split ownership by path.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet, "team:{{ .Vars.team }}"]
      reviewers: ["{{ .Repo.Owner }}/{{ .Vars.team }}"]
    ```

=== "owners.csv"

    ```text
    repo,team
    acme/billing,payments
    acme/ledger,payments
    acme/search,discovery
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/add-codeowners)
