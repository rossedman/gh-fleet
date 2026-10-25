---
title: dependabot.star
---

# dependabot.star

Detect package ecosystems and write a Dependabot configuration.

```python
load("@fleet//dependabot.star", "find", "ecosystems", "config", "write_if_missing")
```

| Function | What it does |
| --- | --- |
| [`find`](#find) | Finds the repo's Dependabot configuration. |
| [`ecosystems`](#ecosystems) | Detects the package ecosystems a repo uses, from files at its root. |
| [`config`](#config) | Builds the text of a dependabot.yml with one update entry per ecosystem. |
| [`write_if_missing`](#write_if_missing) | Writes .github/dependabot.yml for the detected ecosystems unless a config exists. |

## find

```python
find(repo)
```

Finds the repo's Dependabot configuration.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The path of .github/dependabot.yml or .github/dependabot.yaml, or
None when the repo has neither.

**Example**

```python
find(repo)
# returns ".github/dependabot.yaml"
```

=== "Before"

    `.github/dependabot.yaml`

    ```yaml
    version: 2
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/dependabot.star#L17)

## ecosystems

```python
ecosystems(repo)
```

Detects the package ecosystems a repo uses, from files at its root.

Recognizes gomod, npm, pip, cargo, bundler, composer, maven, gradle,
docker and github-actions.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The Dependabot ecosystem names, in a fixed order.

**Example**

```python
ecosystems(repo)
# returns ["gomod", "npm", "github-actions"]
```

=== "Before"

    `go.mod`

    ```text
    module acme/api
    ```

    `package.json`

    ```json
    {}
    ```

    `.github/workflows/ci.yml`

    ```yaml
    on: push
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/dependabot.star#L36)

## config

```python
config(ecosystems, interval = "weekly", group = True, prefixes = {})
```

Builds the text of a dependabot.yml with one update entry per ecosystem.

**Arguments**

| Name | Description |
| --- | --- |
| `ecosystems` | The ecosystem names, such as from `ecosystems(repo)`. |
| `interval` | How often each ecosystem is checked. |
| `group` | When True, minor and patch updates come as one PR per ecosystem. |
| `prefixes` | Commit message prefixes by ecosystem; the default is `ci(deps)` for github-actions and `build(deps)` for the rest. |

**Returns:** The YAML text.

**Example**

```python
text = config(["gomod"], interval = "daily", group = False)
repo.write(".github/dependabot.yml", text)
```

=== "After"

    `.github/dependabot.yml`

    ```yaml
    version: 2
    updates:
      - package-ecosystem: gomod
        directory: /
        schedule:
          interval: daily
        commit-message:
          prefix: "build(deps)"
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/dependabot.star#L58)

## write_if_missing

```python
write_if_missing(repo, interval = "weekly", group = True, prefixes = {})
```

Writes .github/dependabot.yml for the detected ecosystems unless a config exists.

Repos with a config, or with no ecosystem detected, are left alone.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `interval` | Passed to `config`. |
| `group` | Passed to `config`. |
| `prefixes` | Passed to `config`. |

**Returns:** The ecosystems written, or an empty list.

**Example**

```python
write_if_missing(repo, group = False)
# returns ["gomod"]
```

=== "Before"

    `go.mod`

    ```text
    module acme/api
    ```

=== "After"

    `.github/dependabot.yml`

    ```yaml
    version: 2
    updates:
      - package-ecosystem: gomod
        directory: /
        schedule:
          interval: weekly
        commit-message:
          prefix: "build(deps)"
    ```

**Used in:** [`add-dependabot`](https://github.com/rossedman/gh-fleet/tree/main/examples/add-dependabot)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/dependabot.star#L95)
