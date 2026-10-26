---
title: prettier-eslint
---

# prettier-eslint

Give every JavaScript repo the org's Prettier and ESLint settings.

**Uses:** [`eslint.find`](../starlark/eslint.md#find), [`eslint.set_rule`](../starlark/eslint.md#set_rule), [`prettier.set_option`](../starlark/prettier.md#set_option)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from prettier-eslint --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//eslint.star", "find", "set_rule")
    load("@fleet//prettier.star", "set_option")

    def change(repo):
        set_option(repo, "printWidth", 100)
        set_option(repo, "trailingComma", "all")
        config = find(repo)
        if config and config.startswith("eslint.config."):
            repo.set_var("eslint", "ESLint uses a flat config (%s); apply the rules there by hand." % config)
            return
        set_rule(repo, "eqeqeq", "error")
        set_rule(repo, "no-console", "error")
        repo.set_var("eslint", "")
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: js-style-defaults
    description: Give every JavaScript repo the org's Prettier and ESLint settings

    targets:
      orgs: [acme]
      include:
        languages: [TypeScript, JavaScript]

    change:
      steps:
        # Sets Prettier options in whichever config the repo uses (.prettierrc in
        # JSON or YAML, or package.json), and makes ESLint fail on `==` and
        # console calls. Repos on flat eslint.config.js are reported, not edited.
        - name: prettier-eslint
          starlark: ./change.star
          timeout: 1m

    pr:
      title: "chore: standard Prettier and ESLint settings"
      body: |
        Applies the org's Prettier and ESLint settings to {{ .Repo.Name }}.
        {{ .Vars.eslint }}

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet]
    ```

## Test cases

`gh fleet test` runs the change on each case's `before/` files and checks the result against `after/`. See [Testing a campaign](../../guides/testing.md).

### flat-config

=== "Before"

    `eslint.config.js`

    ```text
    export default []
    ```

    `package.json`

    ```json
    {"name": "web", "prettier": {"semi": false}}
    ```

=== "After"

    `eslint.config.js`

    ```text
    export default []
    ```

    `package.json`

    ```json
    {"name": "web", "prettier": {"semi": false, "printWidth": 100, "trailingComma": "all"}}
    ```

### rule-options-kept

=== "Before"

    `.eslintrc.yml`

    ```yaml
    root: true
    env:
      node: true
    rules:
      eol-last: error
      eqeqeq: [warn, allow-null]
      indent: [error, 2, { MemberExpression: "off", SwitchCase: 1 }]
    ```

    `.prettierrc.json`

    ```json
    {
      "semi": false,
      "singleQuote": true
    }
    ```

=== "After"

    `.eslintrc.yml`

    ```yaml
    root: true
    env:
      node: true
    rules:
      eol-last: error
      eqeqeq: [error, allow-null]
      indent: [error, 2, { MemberExpression: "off", SwitchCase: 1 }]
      no-console: error
    ```

    `.prettierrc.json`

    ```json
    {
      "semi": false,
      "singleQuote": true,
      "printWidth": 100,
      "trailingComma": "all"
    }
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/prettier-eslint)
