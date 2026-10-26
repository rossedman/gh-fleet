---
title: k8s-api-migration
---

# k8s-api-migration

Move manifests off API versions Kubernetes removed.

**Uses:** [`k8s.MIGRATIONS`](../starlark/k8s.md), [`k8s.migrate_api_versions`](../starlark/k8s.md#migrate_api_versions), [`k8s.resources`](../starlark/k8s.md#resources)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from k8s-api-migration --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//k8s.star", "MIGRATIONS", "migrate_api_versions", "resources")

    def change(repo):
        old = ["%s/%s" % (r["kind"], r["name"]) for r in resources(repo) if "%s %s" % (r["api_version"], r["kind"]) in MIGRATIONS]
        repo.set_var("resources", ", ".join(old) or "no resources")
        migrate_api_versions(repo)
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: k8s-removed-apis
    description: Move manifests off API versions Kubernetes removed

    targets:
      orgs: [acme]
      include:
        search: "policy/v1beta1 OR batch/v1beta1 OR autoscaling/v2beta2"

    change:
      steps:
        # Rewrites apiVersion for kinds whose only change is the version
        # (PodDisruptionBudget, CronJob, HorizontalPodAutoscaler and others),
        # document by document. Moves that change the schema are not attempted.
        - name: k8s-api-migration
          starlark: ./change.star
          timeout: 1m

    pr:
      title: "deploy: move off removed Kubernetes APIs"
      body: |
        Updates {{ .Vars.resources }} in {{ .Repo.Name }} to API versions current
        Kubernetes serves. Nothing else in the manifests changes.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet, deploy]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/k8s-api-migration)
