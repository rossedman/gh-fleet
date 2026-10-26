---
title: ruby-rails-bump
---

# ruby-rails-bump

Move every Rails app to Rails 7.2.

**Uses:** [`ruby.set_gem`](../starlark/ruby.md#set_gem)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from ruby-rails-bump --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//ruby.star", "set_gem")

    def change(repo):
        repo.set_var("changed", len(set_gem(repo, "rails", env["RAILS"])) > 0)
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: rails-7-2
    description: Move every Rails app to Rails 7.2

    targets:
      orgs: [acme]
      include:
        languages: [Ruby]
        search: "rails filename:Gemfile"

    change:
      steps:
        # Sets the rails requirement in the Gemfile, keeping its options, and
        # tells the lock step whether anything changed.
        - name: Gemfile
          starlark: ./change.star
          timeout: 1m
        # Re-resolves only rails in Gemfile.lock.
        - name: lock
          run: |
            if [ "$FLEET_VAR_CHANGED" = true ] && [ -f Gemfile.lock ]; then bundle lock --update rails; fi
          timeout: 10m
      env:
        RAILS: "~> 7.2"

    pr:
      title: "build(deps): rails 7.2"
      body: |
        Moves {{ .Repo.Name }} to Rails 7.2. Run the test suite and check the
        upgrade guide before merging.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet, dependencies]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/ruby-rails-bump)
