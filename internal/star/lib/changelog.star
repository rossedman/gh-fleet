"""Add entries to a Keep a Changelog style CHANGELOG.md.

A campaign can add a step that records its own change, so the repo's
changelog stays complete without anyone editing it by hand.
"""

NAMES = ["CHANGELOG.md", "CHANGELOG", "CHANGES.md", "HISTORY.md", "docs/CHANGELOG.md"]

def find(repo):
    """Finds the repo's changelog.

    Returns:
        The path of the first of CHANGELOG.md, CHANGELOG, CHANGES.md,
        HISTORY.md and docs/CHANGELOG.md that exists, or None.
    Example:
        before CHANGES.md:
            # Changes
        call:
            find(repo)
        returns:
            "CHANGES.md"
    """
    for name in NAMES:
        if repo.exists(name):
            return name
    return None

def add_entry(repo, text, section = "Changed", path = None):
    """Adds a bullet under the Unreleased release, in a section such as Changed or Security.

    The `## [Unreleased]` heading is added above the first release when
    missing, and the `### <section>` heading inside it. An identical entry
    already there is not added again. A repo without a changelog is left
    alone.

    Args:
        text: the entry, without the leading `- `.
        section: the Keep a Changelog section: Added, Changed, Deprecated,
            Removed, Fixed or Security.
        path: the changelog to edit; found with `find` by default.
    Returns:
        True when the file changed.
    Example:
        before CHANGELOG.md:
            # Changelog

            ## [1.2.0] - 2026-09-01

            ### Added

            - Dark mode.
        call:
            add_entry(repo, "Bump golang.org/x/net to v0.33.0.", section = "Security")
        after CHANGELOG.md:
            # Changelog

            ## [Unreleased]

            ### Security

            - Bump golang.org/x/net to v0.33.0.

            ## [1.2.0] - 2026-09-01

            ### Added

            - Dark mode.
        returns:
            True
    """
    path = path or find(repo)
    if path == None:
        return False
    content = repo.read(path)
    lines = content.split("\n")
    entry = "- " + text
    start = _find_heading(lines, r"(?i)^##\s+\[?unreleased\]?")
    if start < 0:
        # Add Unreleased above the first release, or at the end.
        first = _find_heading(lines, r"^##\s")
        if first < 0:
            while lines and lines[-1].strip() == "":
                lines = lines[:-1]
            lines = lines + ["", "## [Unreleased]", ""]
            start = len(lines) - 2
        else:
            lines = lines[:first] + ["## [Unreleased]", ""] + lines[first:]
            start = first
    end = len(lines)
    for i in range(start + 1, len(lines)):
        if re.matches(r"^##\s", lines[i]):
            end = i
            break
    if entry in [l.strip() for l in lines[start:end]]:
        return False
    sec = -1
    for i in range(start + 1, end):
        if re.matches(r"(?i)^###\s+" + re.escape(section) + r"\s*$", lines[i]):
            sec = i
            break
    if sec < 0:
        lines = lines[:start + 1] + ["", "### " + section, "", entry] + lines[start + 1:]
    else:
        at = sec + 1
        while at < end and lines[at].strip() == "":
            at += 1
        while at < end and lines[at].startswith("- "):
            at += 1
        lines = lines[:at] + [entry] + lines[at:]
    repo.write(path, "\n".join(lines))
    return True

def _find_heading(lines, pattern):
    for i, line in enumerate(lines):
        if re.matches(pattern, line):
            return i
    return -1
