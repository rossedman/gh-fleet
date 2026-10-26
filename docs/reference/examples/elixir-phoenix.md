---
title: elixir-phoenix
---

# elixir-phoenix

Move Phoenix apps to Phoenix 1.7.14 and Elixir 1.17.

**Uses:** [`elixir.set_dep`](../starlark/elixir.md#set_dep), [`elixir.set_elixir`](../starlark/elixir.md#set_elixir), [`runtime.set_version`](../starlark/runtime.md#set_version)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from elixir-phoenix --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//elixir.star", "set_dep", "set_elixir")
    load("@fleet//runtime.star", "set_version")

    def change(repo):
        if set_dep(repo, "phoenix", env["PHOENIX"]):
            set_elixir(repo, "~> 1.17")
            set_version(repo, "elixir", "1.17.3-otp-27")
            set_version(repo, "erlang", "27.1")
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: phoenix-1-7-14
    description: Move Phoenix apps to Phoenix 1.7.14 and Elixir 1.17

    targets:
      orgs: [acme]
      include:
        languages: [Elixir]

    change:
      steps:
        # Sets the phoenix dependency in every mix.exs, raises the elixir
        # requirement, and pins Elixir 1.17.3 and OTP 27 in .tool-versions where
        # the repo uses one.
        - name: elixir-phoenix
          starlark: ./change.star
          timeout: 1m
      env:
        PHOENIX: "~> 1.7.14"

    pr:
      title: "build(deps): phoenix 1.7.14 on elixir 1.17"
      body: |
        Moves {{ .Repo.Name }} to Phoenix 1.7.14 on Elixir 1.17 and OTP 27.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet, dependencies]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/elixir-phoenix)
