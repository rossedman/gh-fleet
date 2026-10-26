---
title: k8s-labels
---

# k8s-labels

Label every Kubernetes resource with its owning team.

**Uses:** [`k8s.set_label`](../starlark/k8s.md#set_label)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from k8s-labels --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//k8s.star", "set_label")

    def change(repo):
        set_label(repo, "app.kubernetes.io/part-of", repo.name)
        set_label(repo, "acme.dev/team", repo.vars["team"])
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: label-owning-team
    description: Label every Kubernetes resource with its owning team

    targets:
      orgs: [acme]
      include:
        topics: [service]

    change:
      steps:
        # Sets app.kubernetes.io/part-of and a team label on every resource and
        # the pods workloads create. The team comes from vars, so each repo gets
        # its own.
        - name: k8s-labels
          starlark: ./change.star
          timeout: 1m

    pr:
      title: "deploy: label resources with team {{ .Vars.team }}"
      body: |
        Labels every resource in {{ .Repo.Name }} with `team: {{ .Vars.team }}`,
        so costs and alerts can be grouped by owner.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet, deploy]

    vars:
      default:
        team: platform
      match:
        - when: { topics: [payments] }
          set: { team: payments }
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/k8s-labels)
