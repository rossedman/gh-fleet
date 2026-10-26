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
    """Reports whether a Markdown file has a heading with this text.

    Headings of any level match, case is ignored, and headings inside code
    fences don't count.

    Args:
        path: the Markdown file.
        heading: the heading text, without `#`.
    Returns:
        True when the section exists.
    Example:
        before README.md:
            # api

            ## Usage

            Run it.
        call:
            has_section(repo, "README.md", "usage")
        returns:
            True
    """
    return _find(repo.read(path, default = "").split("\n"), heading) != None

def set_section(repo, path, heading, body, level = 2, replace = True):
    """Makes the section under a heading contain the given text.

    An existing section, found by heading text at any level, has its content
    replaced up to the next heading of the same or a higher level. A missing
    section is appended at the end of the file. Headings inside code fences
    are ignored.

    Args:
        path: the Markdown file; created if needed.
        heading: the heading text, without `#`.
        body: the section's content.
        level: the heading level for a new section.
        replace: when False, an existing section is left as it is.
    Returns:
        True when the file changed.
    Example:
        before README.md:
            # api

            Intro.

            ## License

            MIT
        call:
            set_section(repo, "README.md", "Security", "Report issues to security@acme.dev.")
        after README.md:
            # api

            Intro.

            ## License

            MIT

            ## Security

            Report issues to security@acme.dev.
        returns:
            True
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
    """Adds a badge to a Markdown file, next to the badges it already shows.

    The badge joins the first run of badges before the first `##` section,
    wherever it is: under the title, or under a logo. Without one, it goes
    under a `#` title, or at the top. A file that already contains the
    badge's text is left alone.

    Args:
        path: the Markdown file.
        badge: one line of Markdown, such as `[![CI](.../badge.svg)](...)`.
    Returns:
        True when the file changed; False when the badge was there or the
        file does not exist.
    Example:
        before README.md:
            # api

            Intro.
        call:
            ensure_badge(repo, "README.md", "[![CI](https://github.com/acme/api/actions/workflows/ci.yml/badge.svg)](https://github.com/acme/api/actions)")
        after README.md:
            # api

            [![CI](https://github.com/acme/api/actions/workflows/ci.yml/badge.svg)](https://github.com/acme/api/actions)

            Intro.
        returns:
            True
    """
    content = repo.read(path, default = None)
    if content == None or badge in content:
        return False
    lines = content.split("\n")
    is_badge = lambda line: line.startswith(("[![", "![")) and "](" in line and ("badge" in line or "shields.io" in line or "[![" in line)

    # The preamble ends at the first section heading below the title.
    end = len(lines)
    for i, line in enumerate(lines):
        h = _heading(line)
        if h and h[0] > 1:
            end = i
            break
    run = -1
    for i in range(end):
        if is_badge(lines[i]):
            run = i
            break
    if run >= 0:
        k = run
        for _ in range(len(lines)):
            if k < len(lines) and is_badge(lines[k]):
                k += 1
            else:
                break
        lines = lines[:k] + [badge] + lines[k:]
    else:
        title = -1
        for i in range(end):
            h = _heading(lines[i])
            if h and h[0] == 1:
                title = i
                break
        if title < 0:
            lines = [badge, ""] + lines
        else:
            at = title + 1
            if at < len(lines) and lines[at] != "":
                lines = lines[:at] + ["", badge, ""] + lines[at:]
            else:
                lines = lines[:at] + ["", badge] + lines[at:]
    repo.write(path, "\n".join(lines))
    return True
