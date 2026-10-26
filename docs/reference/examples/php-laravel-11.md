---
title: php-laravel-11
---

# php-laravel-11

Move Laravel apps to Laravel 11 and PHP 8.2.

**Uses:** [`php.set_php`](../starlark/php.md#set_php), [`php.set_require`](../starlark/php.md#set_require)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from php-laravel-11 --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//php.star", "set_php", "set_require")

    def change(repo):
        changed = set_require(repo, "laravel/framework", env["LARAVEL"])
        set_php(repo, env["PHP"])
        repo.set_var("changed", len(changed) > 0)
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: laravel-11
    description: Move Laravel apps to Laravel 11 and PHP 8.2

    targets:
      orgs: [acme]
      include:
        languages: [PHP]
        search: "laravel/framework filename:composer.json"

    change:
      steps:
        # Sets laravel/framework to ^11.0 and require.php to ^8.2 in every
        # composer.json that lists them, then updates the lockfile for laravel only.
        - name: php-laravel-11
          starlark: ./change.star
          timeout: 1m
        # Resolves the new constraint without installing or running scripts.
        - name: lock
          run: |
            if [ "$FLEET_VAR_CHANGED" = true ] && [ -f composer.lock ]; then composer update laravel/framework --with-all-dependencies --no-install --no-scripts; fi
          timeout: 10m
      env:
        LARAVEL: "^11.0"
        PHP: "^8.2"

    pr:
      title: "build(deps): laravel 11"
      body: |
        Moves {{ .Repo.Name }} to Laravel 11, which needs PHP 8.2. Follow the
        upgrade guide and run the test suite before merging.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet, dependencies]
    ```

## Test cases

`gh fleet test` runs the change on each case's `before/` files and checks the result against `after/`. See [Testing a campaign](../../guides/testing.md).

### already-newer

=== "Before"

    `composer.json`

    ```json
    {
        "name": "laravel/laravel",
        "keywords": ["laravel", "framework"],
        "require": {
            "php": "^8.3",
            "laravel/framework": "^13.17"
        }
    }
    ```

    `composer.lock`

    ```text
    {}
    ```

=== "After"

    No changes: the campaign leaves this repo alone.

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/php-laravel-11)
