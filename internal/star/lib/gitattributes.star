"""Set attributes for path patterns in .gitattributes, such as line endings and Git LFS tracking."""

PATH = ".gitattributes"

def _name(attr):
    return attr.lstrip("-!").split("=")[0]

def set(repo, pattern, attributes):
    """Gives a pattern attributes, merging with any it already has.

    An attribute already on the pattern's line is replaced by the new one
    with the same name, so `eol=crlf` becomes `eol=lf` and `text` becomes
    `-text`; other attributes stay. A pattern with no line gets one at the
    end. The file is created if needed.

    Args:
        pattern: the path pattern, such as `*.sh` or `*`.
        attributes: the attributes, as a space-separated string or a list,
            such as `"text eol=lf"`.
    Returns:
        True when the file changed.
    Example:
        before .gitattributes:
            * text=auto
            *.sh text eol=crlf
        call:
            set(repo, "*.sh", "eol=lf")
            set(repo, "*.png", "binary")
        after .gitattributes:
            * text=auto
            *.sh text eol=lf
            *.png binary
        returns:
            True
    """
    if type(attributes) == "string":
        attributes = attributes.split()
    content = repo.read(PATH, default = "")
    lines = content.split("\n") if content else []
    if lines and lines[-1] == "":
        lines = lines[:-1]
    for i, line in enumerate(lines):
        fields = line.split()
        if not fields or fields[0] != pattern:
            continue
        attrs = fields[1:]
        for a in attributes:
            same = [j for j, x in enumerate(attrs) if _name(x) == _name(a)]
            if same:
                attrs[same[0]] = a
            else:
                attrs.append(a)
        lines[i] = " ".join([pattern] + attrs)
        return _save(repo, content, lines)
    lines.append(" ".join([pattern] + attributes))
    return _save(repo, content, lines)

def track_lfs(repo, pattern):
    """Tracks a pattern with Git LFS, as `git lfs track` would.

    This only edits .gitattributes; files already committed stay in Git
    until they are migrated.

    Args:
        pattern: the path pattern, such as `*.psd`.
    Returns:
        True when the file changed.
    Example:
        before .gitattributes:
            * text=auto
        call:
            track_lfs(repo, "*.psd")
        after .gitattributes:
            * text=auto
            *.psd filter=lfs diff=lfs merge=lfs -text
        returns:
            True
    """
    return set(repo, pattern, "filter=lfs diff=lfs merge=lfs -text")

def _save(repo, content, lines):
    updated = "\n".join(lines) + "\n"
    if updated == content:
        return False
    repo.write(PATH, updated)
    return True
