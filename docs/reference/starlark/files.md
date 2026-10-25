---
title: files.star
---

# files.star

Keep lines in a file, replace text across the repo, and copy files from the campaign directory.

Every helper is idempotent: run it on a repo that already has the change
and it writes nothing, so the repo comes out as a no-op.

```python
load("@fleet//files.star", "ensure_lines", "remove_lines", "replace_all", "write_if_missing", "sync")
```

| Function | What it does |
| --- | --- |
| [`ensure_lines`](#ensure_lines) | Appends each of `lines` that a file does not already contain. |
| [`remove_lines`](#remove_lines) | Removes every line equal to one of `lines`, after trimming whitespace. |
| [`replace_all`](#replace_all) | Replaces every occurrence of a string with another, literally, in text files. |
| [`write_if_missing`](#write_if_missing) | Writes a file unless it, or any of the alternative paths, already exists. |
| [`sync`](#sync) | Copies every file under a directory next to the campaign file into the repo. |

## ensure_lines

```python
ensure_lines(repo, path, lines, header = None)
```

Appends each of `lines` that a file does not already contain.

Lines are compared after trimming whitespace, and blank entries are
skipped. New lines go at the end of the file, after a blank line and
`header` when one is given. The file is created if needed.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `path` | The file to edit, relative to the repo root. |
| `lines` | A list of lines, or one string with a line per line. |
| `header` | A line, such as a comment, written above the new lines when any are added. |

**Returns:** The list of lines that were added; empty when the file had them all.

**Example**

```python
ensure_lines(repo, ".gitignore", [".env", ".env.local"], header = "# Local")
# returns [".env.local"]
```

=== "Before"

    `.gitignore`

    ```text
    node_modules
    .env
    ```

=== "After"

    `.gitignore`

    ```text
    node_modules
    .env

    # Local
    .env.local
    ```

**Used in:** [`gitignore-entries`](https://github.com/rossedman/gh-fleet/tree/main/examples/gitignore-entries)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/files.star#L17)

## remove_lines

```python
remove_lines(repo, path, lines)
```

Removes every line equal to one of `lines`, after trimming whitespace.

A missing file is left alone.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `path` | The file to edit, relative to the repo root. |
| `lines` | A list of lines, or one string with a line per line. |

**Returns:** The list of lines that were removed.

**Example**

```python
remove_lines(repo, ".gitignore", [".travis/"])
# returns [".travis/"]
```

=== "Before"

    `.gitignore`

    ```text
    node_modules
    .travis/
    build/
    ```

=== "After"

    `.gitignore`

    ```text
    node_modules
    build/
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/files.star#L59)

## replace_all

```python
replace_all(repo, old, new, include = ["**"], exclude = [])
```

Replaces every occurrence of a string with another, literally, in text files.

No regular expressions: dots, slashes and brackets mean themselves. Files
containing a NUL byte are treated as binary and skipped.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `old` | The text to find; must not be empty. |
| `new` | The text to put in its place. |
| `include` | Glob patterns of the files to search; `**` matches any depth. |
| `exclude` | Glob patterns of files to skip, such as `["**/vendor/**"]`. |

**Returns:** The list of paths that changed.

**Example**

```python
replace_all(repo, "https://docs.acme.io", "https://docs.acme.dev")
# returns ["README.md", "src/app.ts"]
```

=== "Before"

    `README.md`

    ```markdown
    Docs: https://docs.acme.io/start
    ```

    `src/app.ts`

    ```typescript
    const docs = "https://docs.acme.io";
    ```

=== "After"

    `README.md`

    ```markdown
    Docs: https://docs.acme.dev/start
    ```

    `src/app.ts`

    ```typescript
    const docs = "https://docs.acme.dev";
    ```

**Used in:** [`find-replace`](https://github.com/rossedman/gh-fleet/tree/main/examples/find-replace)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/files.star#L98)

## write_if_missing

```python
write_if_missing(repo, path, content, also = [])
```

Writes a file unless it, or any of the alternative paths, already exists.

Use `also` for files that can live in several places, such as CODEOWNERS
or SECURITY.md, so a repo that already has one elsewhere is left alone.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `path` | The file to write, relative to the repo root. |
| `content` | The text to write. |
| `also` | Other paths that count as the file already existing. |

**Returns:** True when the file was written, False when one already existed.

**Example**

```python
write_if_missing(repo, ".github/CODEOWNERS", "* @acme/web\n", also = ["CODEOWNERS", "docs/CODEOWNERS"])
# returns True
```

=== "After"

    `.github/CODEOWNERS`

    ```text
    * @acme/web
    ```

**Used in:** [`security-policy`](https://github.com/rossedman/gh-fleet/tree/main/examples/security-policy)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/files.star#L136)

## sync

```python
sync(repo, src, dest = "", exclude = [])
```

Copies every file under a directory next to the campaign file into the repo.

Files land at the same relative path under `dest`, replacing what is
there. Files that already match are not touched, and files the repo has
but `src` lacks are kept. New files get mode 0644; an existing file keeps
its mode.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `src` | A directory next to the campaign file, such as `files`. |
| `dest` | The directory in the repo to copy into; the root by default. |
| `exclude` | Glob patterns, relative to the campaign directory, of files not to copy. |

**Returns:** The list of repo paths that changed.

**Example**

```python
sync(repo, "files")
# returns [".editorconfig", ".github/pull_request_template.md"]
```

=== "Before"

    `files/.editorconfig` next to the campaign file

    ```text
    root = true
    ```

    `files/.github/pull_request_template.md` next to the campaign file

    ```markdown
    ## What and why
    ```

    `.editorconfig`

    ```text
    root = false
    ```

=== "After"

    `.editorconfig`

    ```text
    root = true
    ```

    `.github/pull_request_template.md`

    ```markdown
    ## What and why
    ```

**Used in:** [`sync-files`](https://github.com/rossedman/gh-fleet/tree/main/examples/sync-files)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/files.star#L161)
