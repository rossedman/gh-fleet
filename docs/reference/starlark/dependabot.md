---
title: dependabot.star
---

# dependabot.star

Detect package ecosystems, write a Dependabot configuration, and edit existing ones.

```python
load("@fleet//dependabot.star", "find", "ecosystems", "config", "write_if_missing", "add_update", "set_option")
```

| Function | What it does |
| --- | --- |
| [`find`](#find) | Finds the repo's Dependabot configuration. |
| [`ecosystems`](#ecosystems) | Detects the package ecosystems a repo uses, from files at its root. |
| [`config`](#config) | Builds the text of a dependabot.yml with one update entry per ecosystem. |
| [`write_if_missing`](#write_if_missing) | Writes .github/dependabot.yml for the detected ecosystems unless a config exists. |
| [`add_update`](#add_update) | Adds an update entry to an existing Dependabot config, unless one covers that ecosystem and directory. |
| [`set_option`](#set_option) | Sets an option on update entries, such as `open-pull-requests-limit` or `rebase-strategy`. |

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

**Used in:** [`dependabot-ecosystems`](https://github.com/rossedman/gh-fleet/tree/main/examples/dependabot-ecosystems)

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

## add_update

```python
add_update(
    repo,
    ecosystem,
    directory = "/",
    interval = "weekly",
    prefix = None,
)
```

Adds an update entry to an existing Dependabot config, unless one covers that ecosystem and directory.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `ecosystem` | The package ecosystem, such as `docker` or `gomod`. |
| `directory` | The directory Dependabot looks in. |
| `interval` | How often to check. |
| `prefix` | The commit message prefix; `ci(deps)` for github-actions and `build(deps)` otherwise when None. |

**Returns:** True when the file changed; False when the entry exists or the repo
has no config (use `write_if_missing` to create one).

**Example**

```python
add_update(repo, "docker")
# returns True
```

=== "Before"

    `.github/dependabot.yml`

    ```yaml
    version: 2
    updates:
      - package-ecosystem: gomod
        directory: /
        schedule:
          interval: weekly
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
      - package-ecosystem: docker
        directory: /
        schedule:
          interval: weekly
        commit-message:
          prefix: "build(deps)"
    ```

**Used in:** [`dependabot-ecosystems`](https://github.com/rossedman/gh-fleet/tree/main/examples/dependabot-ecosystems)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/dependabot.star#L153)

## set_option

```python
set_option(repo, key, value, ecosystem = None)
```

Sets an option on update entries, such as `open-pull-requests-limit` or `rebase-strategy`.

An existing option is replaced; a missing one is added to the entry.
Use `schedule.interval` for the check interval.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `key` | The option, dotted for nested ones, such as `schedule.interval`. |
| `value` | A string, number or boolean. |
| `ecosystem` | Only entries for this ecosystem change; every entry when None. |

**Returns:** True when the file changed.

**Example**

```python
set_option(repo, "schedule.interval", "weekly", ecosystem = "npm")
set_option(repo, "open-pull-requests-limit", 5, ecosystem = "npm")
# returns True
```

=== "Before"

    `.github/dependabot.yml`

    ```yaml
    version: 2
    updates:
      - package-ecosystem: npm
        directory: /
        schedule:
          interval: daily  # too noisy
      - package-ecosystem: github-actions
        directory: /
        schedule:
          interval: weekly
    ```

=== "After"

    `.github/dependabot.yml`

    ```yaml
    version: 2
    updates:
      - package-ecosystem: npm
        directory: /
        schedule:
          interval: weekly  # too noisy
        open-pull-requests-limit: 5
      - package-ecosystem: github-actions
        directory: /
        schedule:
          interval: weekly
    ```

**Used in:** [`dependabot-ecosystems`](https://github.com/rossedman/gh-fleet/tree/main/examples/dependabot-ecosystems)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/dependabot.star#L222)
