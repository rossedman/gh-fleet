---
title: markdown.star
---

# markdown.star

Add or replace sections and badges in Markdown files such as README.md.

```python
load("@fleet//markdown.star", "has_section", "set_section", "ensure_badge")
```

| Function | What it does |
| --- | --- |
| [`has_section`](#has_section) | Reports whether a Markdown file has a heading with this text. |
| [`set_section`](#set_section) | Makes the section under a heading contain the given text. |
| [`ensure_badge`](#ensure_badge) | Adds a badge to a Markdown file, next to the badges it already shows. |

## has_section

```python
has_section(repo, path, heading)
```

Reports whether a Markdown file has a heading with this text.

Headings of any level match, case is ignored, and headings inside code
fences don't count.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `path` | The Markdown file. |
| `heading` | The heading text, without `#`. |

**Returns:** True when the section exists.

**Example**

```python
has_section(repo, "README.md", "usage")
# returns True
```

=== "Before"

    `README.md`

    ```markdown
    # api

    ## Usage

    Run it.
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/markdown.star#L28)

## set_section

```python
set_section(repo, path, heading, body, level = 2, replace = True)
```

Makes the section under a heading contain the given text.

An existing section, found by heading text at any level, has its content
replaced up to the next heading of the same or a higher level. A missing
section is appended at the end of the file. Headings inside code fences
are ignored.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `path` | The Markdown file; created if needed. |
| `heading` | The heading text, without `#`. |
| `body` | The section's content. |
| `level` | The heading level for a new section. |
| `replace` | When False, an existing section is left as it is. |

**Returns:** True when the file changed.

**Example**

```python
set_section(repo, "README.md", "Security", "Report issues to security@acme.dev.")
# returns True
```

=== "Before"

    `README.md`

    ```markdown
    # api

    Intro.

    ## License

    MIT
    ```

=== "After"

    `README.md`

    ```markdown
    # api

    Intro.

    ## License

    MIT

    ## Security

    Report issues to security@acme.dev.
    ```

**Used in:** [`readme-section`](../examples/readme-section.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/markdown.star#L53)

## ensure_badge

```python
ensure_badge(repo, path, badge)
```

Adds a badge to a Markdown file, next to the badges it already shows.

The badge joins the first run of badges before the first `##` section,
wherever it is: under the title, or under a logo. Without one, it goes
under a `#` title, or at the top. A file that already contains the
badge's text is left alone.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `path` | The Markdown file. |
| `badge` | One line of Markdown, such as `[![CI](.../badge.svg)](...)`. |

**Returns:** True when the file changed; False when the badge was there or the
file does not exist.

**Example**

```python
ensure_badge(repo, "README.md", "[![CI](https://github.com/acme/api/actions/workflows/ci.yml/badge.svg)](https://github.com/acme/api/actions)")
# returns True
```

=== "Before"

    `README.md`

    ```markdown
    # api

    Intro.
    ```

=== "After"

    `README.md`

    ```markdown
    # api

    [![CI](https://github.com/acme/api/actions/workflows/ci.yml/badge.svg)](https://github.com/acme/api/actions)

    Intro.
    ```

**Used in:** [`readme-section`](../examples/readme-section.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/markdown.star#L112)
