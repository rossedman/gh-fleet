---
title: buildkite-agents
---

# buildkite-agents

Move pipelines to the new agent queue and give every step a timeout.

**Uses:** [`buildkite.set_queue`](../starlark/buildkite.md#set_queue), [`buildkite.set_step_defaults`](../starlark/buildkite.md#set_step_defaults)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from buildkite-agents --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//buildkite.star", "set_queue", "set_step_defaults")

    def change(repo):
        set_queue(repo, env["TO"], from_queue = env["FROM"])
        set_step_defaults(repo, {"timeout_in_minutes": int(env["TIMEOUT"])})
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: buildkite-queues
    description: Move pipelines to the new agent queue and give every step a timeout

    targets:
      orgs: [acme]
      include:
        files: [.buildkite]

    change:
      steps:
        # Moves steps on the retired `default` queue to `linux-small`, and adds
        # timeout_in_minutes to every command step that has none, so a hung step
        # can't hold an agent forever. Steps that set their own timeout keep it.
        - name: buildkite-agents
          starlark: ./change.star
          timeout: 1m
      env:
        FROM: default
        TO: linux-small
        TIMEOUT: "30"

    pr:
      title: "ci: new agent queue and step timeouts"
      body: |
        Moves {{ .Repo.Name }} off the retired `default` agent queue and gives
        every command step a 30-minute timeout.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/buildkite-agents)
