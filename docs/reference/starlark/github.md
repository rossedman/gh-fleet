---
title: github.star
---

# github.star

Edit repository files under .github: issue template config, release note categories, labels and funding.

Edits are line-based, so comments and layout stay as they were. For
workflows, see `gha`; for CODEOWNERS, `codeowners`; for Dependabot,
`dependabot`.

```python
load("@fleet//github.star", "issue_templates", "set_blank_issues", "add_contact_link", "add_release_category", "set_label", "set_funding")
```

| Function | What it does |
| --- | --- |
| [`issue_templates`](#issue_templates) | Lists the repo's issue templates and forms. |
| [`set_blank_issues`](#set_blank_issues) | Allows or disallows issues that don't use a template. |
| [`add_contact_link`](#add_contact_link) | Adds a contact link to the new-issue page, unless one with this URL exists. |
| [`add_release_category`](#add_release_category) | Adds a category to the release notes GitHub generates, unless one with this title exists. |
| [`set_label`](#set_label) | Adds or updates a label in a label sync file, such as .github/labels.yml. |
| [`set_funding`](#set_funding) | Sets a sponsor link in .github/FUNDING.yml, such as `github: [acme]`. |

## issue_templates

```python
issue_templates(repo)
```

Lists the repo's issue templates and forms.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The sorted paths of the Markdown templates and YAML forms in
.github/ISSUE_TEMPLATE, without its config.yml.

**Example**

```python
issue_templates(repo)
# returns [".github/ISSUE_TEMPLATE/bug.yml", ".github/ISSUE_TEMPLATE/feature.md"]
```

=== "Before"

    `.github/ISSUE_TEMPLATE/bug.yml`

    ```yaml
    name: Bug
    ```

    `.github/ISSUE_TEMPLATE/feature.md`

    ```markdown
    ---
    ```

    `.github/ISSUE_TEMPLATE/config.yml`

    ```yaml
    blank_issues_enabled: false
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/github.star#L13)

## set_blank_issues

```python
set_blank_issues(repo, enabled)
```

Allows or disallows issues that don't use a template.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `enabled` | Whether blank issues are allowed. |

**Returns:** True when the file changed. The config is created if needed.

**Example**

```python
set_blank_issues(repo, False)
# returns True
```

=== "Before"

    `.github/ISSUE_TEMPLATE/config.yml`

    ```yaml
    blank_issues_enabled: true
    ```

=== "After"

    `.github/ISSUE_TEMPLATE/config.yml`

    ```yaml
    blank_issues_enabled: false
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/github.star#L33)

## add_contact_link

```python
add_contact_link(repo, name, url, about)
```

Adds a contact link to the new-issue page, unless one with this URL exists.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `name` | The link's title. |
| `url` | Where it points. |
| `about` | The line shown under it. |

**Returns:** True when the file changed. The config is created if needed.

**Example**

```python
add_contact_link(repo, "Security report", "https://github.com/acme/api/security/advisories/new", "Report a vulnerability privately.")
# returns True
```

=== "Before"

    `.github/ISSUE_TEMPLATE/config.yml`

    ```yaml
    blank_issues_enabled: false
    ```

=== "After"

    `.github/ISSUE_TEMPLATE/config.yml`

    ```yaml
    blank_issues_enabled: false
    contact_links:
      - name: Security report
        url: https://github.com/acme/api/security/advisories/new
        about: Report a vulnerability privately.
    ```

**Used in:** [`github-repo-files`](../examples/github-repo-files.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/github.star#L89)

## add_release_category

```python
add_release_category(repo, title, labels)
```

Adds a category to the release notes GitHub generates, unless one with this title exists.

Categories go in `changelog.categories` of .github/release.yml, which
is created if missing. GitHub uses the first category whose labels
match a pull request, so add catch-alls (`labels: ["*"]`) last.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `title` | The category heading, such as `Security`. |
| `labels` | The labels that put a pull request in it. |

**Returns:** True when the file changed.

**Example**

```python
add_release_category(repo, "Dependencies", ["dependencies"])
# returns True
```

=== "Before"

    `.github/release.yml`

    ```yaml
    changelog:
      categories:
        - title: Features
          labels: [feature]
    ```

=== "After"

    `.github/release.yml`

    ```yaml
    changelog:
      categories:
        - title: Features
          labels: [feature]
        - title: Dependencies
          labels: [dependencies]
    ```

**Used in:** [`github-repo-files`](../examples/github-repo-files.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/github.star#L123)

## set_label

```python
set_label(
    repo,
    name,
    color,
    description = None,
    path = ".github/labels.yml",
)
```

Adds or updates a label in a label sync file, such as .github/labels.yml.

The file is a list of labels with `name`, `color` and `description`, as
used by label sync actions and by Probot Settings under `labels:`. An
existing label keeps its place and gets the new color and description,
keeping the quotes it was written with; a new label's color is quoted.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `name` | The label name. |
| `color` | The hex color without `#`, such as `d73a4a`. |
| `description` | The label description, or None to leave it. |
| `path` | The label file. |

**Returns:** True when the file changed. The file is created if needed.

**Example**

```python
set_label(repo, "fleet", "1f6feb", description = "Opened by gh fleet")
# returns True
```

=== "Before"

    `.github/labels.yml`

    ```yaml
    - name: bug
      color: d73a4a
    - name: fleet
      color: "000000"
    ```

=== "After"

    `.github/labels.yml`

    ```yaml
    - name: bug
      color: d73a4a
    - name: fleet
      color: "1f6feb"
      description: Opened by gh fleet
    ```

**Used in:** [`github-repo-files`](../examples/github-repo-files.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/github.star#L187)

## set_funding

```python
set_funding(repo, platform, value)
```

Sets a sponsor link in .github/FUNDING.yml, such as `github: [acme]`.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `platform` | The platform key, such as `github`, `open_collective` or `custom`. |
| `value` | The account, or a list of accounts or URLs. |

**Returns:** True when the file changed. The file is created if needed.

**Example**

```python
set_funding(repo, "github", ["acme"])
set_funding(repo, "open_collective", "acme")
# returns True
```

=== "Before"

    `.github/FUNDING.yml`

    ```yaml
    github: [old-maintainer]
    ```

=== "After"

    `.github/FUNDING.yml`

    ```yaml
    github: [acme]
    open_collective: acme
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/github.star#L234)
