"""Add or replace sections and badges in Markdown files such as README.md."""

def _heading(line):
    # Returns (level, text) for an ATX heading line, or None.
    g = re.groups(r"^(#{1,6})\s+(.*?)\s*#*\s*$", line)
    return (len(g[0]), g[1]) if g else None

def _find(lines, heading):
    # Returns (start, end, level) of the section titled heading, or None.
    fence = False
    for i, line in enumerate(lines):
        if line.lstrip().startswith("```"):
            fence = not fence
        h = None if fence else _heading(line)
        if h and h[1].lower() == heading.lower():
            end = len(lines)
            inner = False
            for j in range(i + 1, len(lines)):
                if lines[j].lstrip().startswith("```"):
                    inner = not inner
                h2 = None if inner else _heading(lines[j])
                if h2 and h2[0] <= h[0]:
                    end = j
                    break
            return (i, end, h[0])
    return None

def has_section(repo, path, heading):
    """Reports whether `path` has a heading with this text, at any level, ignoring case."""
    return _find(repo.read(path, default = "").split("\n"), heading) != None

def set_section(repo, path, heading, body, level = 2, replace = True):
    """Makes the section titled `heading` contain `body`.

    An existing section, found by heading text at any level, has its
    content replaced up to the next heading of the same or a higher level;
    with `replace = False` it is left as it is. A missing section is
    appended at the end of the file as a level-`level` heading. Headings
    inside code fences are ignored.

    Returns True when the file changed.
    """
    content = repo.read(path, default = "")
    lines = content.split("\n")
    found = _find(lines, heading)
    body = body.strip("\n")
    if found:
        if not replace:
            return False
        start, end, _ = found
        updated = "\n".join(lines[:start + 1] + ["", body, ""] + lines[end:])
    else:
        section = "#" * level + " " + heading + "\n\n" + body + "\n"
        updated = content.rstrip("\n") + "\n\n" + section if content.strip() else section
    if updated == content:
        return False
    repo.write(path, updated)
    return True

def ensure_badge(repo, path, badge):
    """Adds a badge (a line of Markdown such as `[![CI](...)](...)`) under the first heading.

    A file that already contains the badge's text anywhere is left alone,
    and a file with no heading gets the badge at the top. Returns True when
    the file changed.
    """
    content = repo.read(path, default = None)
    if content == None or badge in content:
        return False
    lines = content.split("\n")
    at = -1
    for i, line in enumerate(lines):
        if _heading(line):
            at = i + 1
            break
    if at < 0:
        lines = [badge, ""] + lines
    else:
        # Keep badges together: after a blank line, behind any already there.
        j = at + 1 if at < len(lines) and lines[at] == "" else at
        k = j
        while k < len(lines) and lines[k].startswith(("[![", "![")):
            k += 1
        if k > j:
            lines = lines[:k] + [badge] + lines[k:]
        elif at < len(lines) and lines[at] != "":
            lines = lines[:at] + ["", badge, ""] + lines[at:]
        else:
            lines = lines[:at] + ["", badge] + lines[at:]
    repo.write(path, "\n".join(lines))
    return True
