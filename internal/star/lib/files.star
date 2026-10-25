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
    """Appends each of `lines` that `path` does not already contain.

    `lines` is a list or a newline-separated string. Lines are compared
    after trimming whitespace, and blank entries are skipped. The new lines
    go at the end of the file, after a blank line and `header` (for example
    a comment) when one is given. The file is created if needed.

    Returns the lines that were added.
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
    """Removes every line of `path` equal to one of `lines` after trimming whitespace.

    A missing file is left alone. Returns the lines that were removed.
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
    """Replaces every occurrence of `old` with `new`, literally, in text files.

    `include` and `exclude` are glob patterns (`**` matches any depth).
    Binary files, which contain a NUL byte, are skipped.

    Returns the paths that changed.
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
    """Writes `path` unless it, or any path in `also`, already exists.

    Use `also` for files that can live in several places, such as
    CODEOWNERS. Returns True when the file was written.
    """
    if repo.exists(path, *also):
        return False
    repo.write(path, content)
    return True

def sync(repo, src, dest = "", exclude = []):
    """Copies every file under `src` in the campaign directory into the repo.

    Files land at the same relative path under `dest` (the repo root by
    default), replacing what is there. Files already identical are not
    touched, and files the repo has but `src` lacks are kept. New files are
    written with mode 0644; an existing file keeps its mode.

    Returns the repo paths that changed.
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
