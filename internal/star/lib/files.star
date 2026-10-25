"""Keep lines in a file, replace text across the repo, and copy files from the campaign directory.

Every helper is idempotent: run it on a repo that already has the change
and it writes nothing, so the repo comes out as a no-op.
"""

def _lines(value):
    if type(value) == "string":
        value = value.splitlines()
    out = []
    for line in value:
        line = line.strip()
        if line and line not in out:
            out.append(line)
    return out

def ensure_lines(repo, path, lines, header = None):
    """Appends each of `lines` that a file does not already contain.

    Lines are compared after trimming whitespace, and blank entries are
    skipped. New lines go at the end of the file, after a blank line and
    `header` when one is given. The file is created if needed.

    Args:
        path: the file to edit, relative to the repo root.
        lines: a list of lines, or one string with a line per line.
        header: a line, such as a comment, written above the new lines when
            any are added.
    Returns:
        The list of lines that were added; empty when the file had them all.
    Example:
        before .gitignore:
            node_modules
            .env
        call:
            ensure_lines(repo, ".gitignore", [".env", ".env.local"], header = "# Local")
        after .gitignore:
            node_modules
            .env

            # Local
            .env.local
        returns:
            [".env.local"]
    """
    content = repo.read(path, default = "")
    have = [line.strip() for line in content.splitlines()]
    missing = [line for line in _lines(lines) if line not in have]
    if not missing:
        return []
    block = ""
    if content:
        block = "\n" if content.endswith("\n") else "\n\n"
    if header:
        block += header + "\n"
    repo.write(path, content + block + "\n".join(missing) + "\n")
    return missing

def remove_lines(repo, path, lines):
    """Removes every line equal to one of `lines`, after trimming whitespace.

    A missing file is left alone.

    Args:
        path: the file to edit, relative to the repo root.
        lines: a list of lines, or one string with a line per line.
    Returns:
        The list of lines that were removed.
    Example:
        before .gitignore:
            node_modules
            .travis/
            build/
        call:
            remove_lines(repo, ".gitignore", [".travis/"])
        after .gitignore:
            node_modules
            build/
        returns:
            [".travis/"]
    """
    content = repo.read(path, default = None)
    if content == None:
        return []
    drop = _lines(lines)
    kept, removed = [], []
    for line in content.splitlines():
        if line.strip() in drop:
            if line.strip() not in removed:
                removed.append(line.strip())
        else:
            kept.append(line)
    if removed:
        out = "\n".join(kept)
        repo.write(path, out + "\n" if out else "")
    return removed

def replace_all(repo, old, new, include = ["**"], exclude = []):
    """Replaces every occurrence of a string with another, literally, in text files.

    No regular expressions: dots, slashes and brackets mean themselves. Files
    containing a NUL byte are treated as binary and skipped.

    Args:
        old: the text to find; must not be empty.
        new: the text to put in its place.
        include: glob patterns of the files to search; `**` matches any depth.
        exclude: glob patterns of files to skip, such as `["**/vendor/**"]`.
    Returns:
        The list of paths that changed.
    Example:
        before README.md:
            Docs: https://docs.acme.io/start
        before src/app.ts:
            const docs = "https://docs.acme.io";
        call:
            replace_all(repo, "https://docs.acme.io", "https://docs.acme.dev")
        after README.md:
            Docs: https://docs.acme.dev/start
        after src/app.ts:
            const docs = "https://docs.acme.dev";
        returns:
            ["README.md", "src/app.ts"]
    """
    if not old:
        fail("replace_all: old must not be empty")
    changed = []
    for path in repo.glob(exclude = exclude, *include):
        content = repo.read(path)
        if old not in content or "\0" in content:
            continue
        repo.write(path, content.replace(old, new))
        changed.append(path)
    return changed

def write_if_missing(repo, path, content, also = []):
    r"""Writes a file unless it, or any of the alternative paths, already exists.

    Use `also` for files that can live in several places, such as CODEOWNERS
    or SECURITY.md, so a repo that already has one elsewhere is left alone.

    Args:
        path: the file to write, relative to the repo root.
        content: the text to write.
        also: other paths that count as the file already existing.
    Returns:
        True when the file was written, False when one already existed.
    Example:
        call:
            write_if_missing(repo, ".github/CODEOWNERS", "* @acme/web\n", also = ["CODEOWNERS", "docs/CODEOWNERS"])
        after .github/CODEOWNERS:
            * @acme/web
        returns:
            True
    """
    if repo.exists(path, *also):
        return False
    repo.write(path, content)
    return True

def sync(repo, src, dest = "", exclude = []):
    """Copies every file under a directory next to the campaign file into the repo.

    Files land at the same relative path under `dest`, replacing what is
    there. Files that already match are not touched, and files the repo has
    but `src` lacks are kept. New files get mode 0644; an existing file keeps
    its mode.

    Args:
        src: a directory next to the campaign file, such as `files`.
        dest: the directory in the repo to copy into; the root by default.
        exclude: glob patterns, relative to the campaign directory, of files
            not to copy.
    Returns:
        The list of repo paths that changed.
    Example:
        campaign files/.editorconfig:
            root = true
        campaign files/.github/pull_request_template.md:
            ## What and why
        before .editorconfig:
            root = false
        call:
            sync(repo, "files")
        after .editorconfig:
            root = true
        after .github/pull_request_template.md:
            ## What and why
        returns:
            [".editorconfig", ".github/pull_request_template.md"]
    """
    src = src.removeprefix("./").rstrip("/")
    if src in ("", "."):
        fail("sync: src must be a directory below the campaign file, such as files")
    dest = dest.removeprefix("./").rstrip("/")
    changed = []
    files = campaign.glob(src + "/**", exclude = exclude)
    if not files:
        fail("sync: no files under %s in the campaign directory" % src)
    for path in files:
        rel = path[len(src) + 1:]
        target = dest + "/" + rel if dest else rel
        content = campaign.read(path)
        if repo.read(target, default = None) != content:
            repo.write(target, content)
            changed.append(target)
    return changed
