---
title: changelog.star
---

# changelog.star

Add entries to a Keep a Changelog style CHANGELOG.md.

A campaign can add a step that records its own change, so the repo's
changelog stays complete without anyone editing it by hand.

```python
load("@fleet//changelog.star", "find", "add_entry")
```

| Function | What it does |
| --- | --- |
| [`find`](#find) | Finds the repo's changelog. |
| [`add_entry`](#add_entry) | Adds a bullet under the Unreleased release, in a section such as Changed or Security. |

## find

```python
find(repo)
```

Finds the repo's changelog.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The path of the first of CHANGELOG.md, CHANGELOG, CHANGES.md,
HISTORY.md and docs/CHANGELOG.md that exists, or None.

**Example**

```python
find(repo)
# returns "CHANGES.md"
```

=== "Before"

    `CHANGES.md`

    ```markdown
    # Changes
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/changelog.star#L9)

## add_entry

```python
add_entry(repo, text, section = "Changed", path = None)
```

Adds a bullet under the Unreleased release, in a section such as Changed or Security.

The `## [Unreleased]` heading is added above the first release when
missing, and the `### <section>` heading inside it. An identical entry
already there is not added again. A repo without a changelog is left
alone.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `text` | The entry, without the leading `- `. |
| `section` | The Keep a Changelog section: Added, Changed, Deprecated, Removed, Fixed or Security. |
| `path` | The changelog to edit; found with `find` by default. |

**Returns:** True when the file changed.

**Example**

```python
add_entry(repo, "Bump golang.org/x/net to v0.33.0.", section = "Security")
# returns True
```

=== "Before"

    `CHANGELOG.md`

    ```markdown
    # Changelog

    ## [1.2.0] - 2026-09-01

    ### Added

    - Dark mode.
    ```

=== "After"

    `CHANGELOG.md`

    ```markdown
    # Changelog

    ## [Unreleased]

    ### Security

    - Bump golang.org/x/net to v0.33.0.

    ## [1.2.0] - 2026-09-01

    ### Added

    - Dark mode.
    ```

**Used in:** [`changelog-entry`](https://github.com/rossedman/gh-fleet/tree/main/examples/changelog-entry)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/changelog.star#L28)
