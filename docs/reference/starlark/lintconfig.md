---
title: lintconfig.star
---

# lintconfig.star

Set rules in markdownlint, yamllint and hadolint configs.

Each helper edits the config file the tool reads by default, keeping
comments in YAML files and key order in JSON files.

```python
load("@fleet//lintconfig.star", "markdownlint_rule", "yamllint_rule", "hadolint_ignore")
```

| Function | What it does |
| --- | --- |
| [`markdownlint_rule`](#markdownlint_rule) | Sets a markdownlint rule, such as `MD013` (line length). |
| [`yamllint_rule`](#yamllint_rule) | Sets a yamllint rule, such as turning off `document-start`. |
| [`hadolint_ignore`](#hadolint_ignore) | Adds rules to the `ignored` list of .hadolint.yaml, creating the file if needed. |

## markdownlint_rule

```python
markdownlint_rule(repo, rule, value)
```

Sets a markdownlint rule, such as `MD013` (line length).

Edits .markdownlint.json, .markdownlint.jsonc (without comments),
.markdownlint.yaml or .markdownlint.yml.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `rule` | The rule ID or alias, such as `MD013` or `line-length`. |
| `value` | True or False to turn the rule on or off, or a dict of its options. |

**Returns:** True when the file changed; False when the repo has no config.

**Example**

```python
markdownlint_rule(repo, "MD013", False)
# returns True
```

=== "Before"

    `.markdownlint.json`

    ```json
    {
      "default": true
    }
    ```

=== "After"

    `.markdownlint.json`

    ```json
    {
      "default": true,
      "MD013": false
    }
    ```

**Used in:** [`lint-configs`](../examples/lint-configs.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/lintconfig.star#L16)

## yamllint_rule

```python
yamllint_rule(repo, rule, value)
```

Sets a yamllint rule, such as turning off `document-start`.

Edits .yamllint, .yamllint.yaml or .yamllint.yml.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `rule` | The rule, such as `line-length`. |
| `value` | `disable` or `enable`, or an option of the rule as `{"max": 120}`, set key by key. |

**Returns:** True when the file changed; False when the repo has no config.

**Example**

```python
yamllint_rule(repo, "line-length", {"max": 120})
yamllint_rule(repo, "document-start", "disable")
# returns True
```

=== "Before"

    `.yamllint.yml`

    ```yaml
    extends: default
    rules:
      line-length:
        max: 80  # strict
    ```

=== "After"

    `.yamllint.yml`

    ```yaml
    extends: default
    rules:
      line-length:
        max: 120  # strict
      document-start: disable
    ```

**Used in:** [`lint-configs`](../examples/lint-configs.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/lintconfig.star#L52)

## hadolint_ignore

```python
hadolint_ignore(repo, rules)
```

Adds rules to the `ignored` list of .hadolint.yaml, creating the file if needed.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `rules` | The rule IDs, such as `["DL3008", "DL3013"]`. |

**Returns:** True when the file changed.

**Example**

```python
hadolint_ignore(repo, ["DL3013", "DL3008"])
# returns True
```

=== "Before"

    `.hadolint.yaml`

    ```yaml
    ignored:
      - DL3008  # apt pins
    ```

=== "After"

    `.hadolint.yaml`

    ```yaml
    ignored:
      - DL3008  # apt pins
      - DL3013
    ```

**Used in:** [`lint-configs`](../examples/lint-configs.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/lintconfig.star#L92)
