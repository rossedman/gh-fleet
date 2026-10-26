"""Edit repository files under .github: issue template config, release note categories, labels and funding.

Edits are line-based, so comments and layout stay as they were. For
workflows, see `gha`; for CODEOWNERS, `codeowners`; for Dependabot,
`dependabot`.
"""

load("./yamlpath.star", _get = "get", _set = "set", _set_sibling = "set_sibling")

ISSUE_CONFIG = ".github/ISSUE_TEMPLATE/config.yml"
RELEASE = ".github/release.yml"

def issue_templates(repo):
    """Lists the repo's issue templates and forms.

    Returns:
        The sorted paths of the Markdown templates and YAML forms in
        .github/ISSUE_TEMPLATE, without its config.yml.
    Example:
        before .github/ISSUE_TEMPLATE/bug.yml:
            name: Bug
        before .github/ISSUE_TEMPLATE/feature.md:
            ---
        before .github/ISSUE_TEMPLATE/config.yml:
            blank_issues_enabled: false
        call:
            issue_templates(repo)
        returns:
            [".github/ISSUE_TEMPLATE/bug.yml", ".github/ISSUE_TEMPLATE/feature.md"]
    """
    return [p for p in repo.glob(".github/ISSUE_TEMPLATE/*.md", ".github/ISSUE_TEMPLATE/*.yml", ".github/ISSUE_TEMPLATE/*.yaml") if not p.endswith("/config.yml") and not p.endswith("/config.yaml")]

def set_blank_issues(repo, enabled):
    """Allows or disallows issues that don't use a template.

    Args:
        enabled: whether blank issues are allowed.
    Returns:
        True when the file changed. The config is created if needed.
    Example:
        before .github/ISSUE_TEMPLATE/config.yml:
            blank_issues_enabled: true
        call:
            set_blank_issues(repo, False)
        after .github/ISSUE_TEMPLATE/config.yml:
            blank_issues_enabled: false
        returns:
            True
    """
    return _set(repo, ISSUE_CONFIG, "blank_issues_enabled", enabled)

def _append_item(repo, path, key, item_lines, exists):
    # Appends a mapping item to the top-level list `key`, creating it.
    content = repo.read(path, default = "")
    if exists(content):
        return False
    lines = content.split("\n") if content else []
    while lines and lines[-1].strip() == "":
        lines = lines[:-1]
    start = -1
    for i, line in enumerate(lines):
        if re.matches(r"^" + re.escape(key) + r":\s*(\[\s*\])?\s*(#.*)?$", line):
            start = i
            if "[" in line:
                lines[i] = key + ":"
            break
    if start < 0:
        lines = lines + [key + ":"]
        start = len(lines) - 1
        indent = "  "
    else:
        indent = "  "
        for j in range(start + 1, len(lines)):
            if lines[j].lstrip().startswith("-"):
                indent = lines[j][:len(lines[j]) - len(lines[j].lstrip(" "))]
                break
    end = start
    for j in range(start + 1, len(lines)):
        s = lines[j]
        if s.strip() and not s.startswith(" ") and not s.startswith("-"):
            break
        if s.strip():
            end = j
    new = [indent + "- " + item_lines[0]] + [indent + "  " + l for l in item_lines[1:]]
    lines = lines[:end + 1] + new + lines[end + 1:]
    repo.write(path, "\n".join(lines) + "\n")
    return True

def add_contact_link(repo, name, url, about):
    """Adds a contact link to the new-issue page, unless one with this URL exists.

    Args:
        name: the link's title.
        url: where it points.
        about: the line shown under it.
    Returns:
        True when the file changed. The config is created if needed.
    Example:
        before .github/ISSUE_TEMPLATE/config.yml:
            blank_issues_enabled: false
        call:
            add_contact_link(repo, "Security report", "https://github.com/acme/api/security/advisories/new", "Report a vulnerability privately.")
        after .github/ISSUE_TEMPLATE/config.yml:
            blank_issues_enabled: false
            contact_links:
              - name: Security report
                url: https://github.com/acme/api/security/advisories/new
                about: Report a vulnerability privately.
        returns:
            True
    """
    return _append_item(repo, ISSUE_CONFIG, "contact_links", [
        "name: " + _plain(name),
        "url: " + _plain(url),
        "about: " + _plain(about),
    ], lambda c: re.matches(r"(?m)^\s*-?\s*url:\s*[\"']?" + re.escape(url) + r"[\"']?\s*$", c))

def _plain(s):
    if re.matches(r"^[A-Za-z0-9][\w ./:@()+,-]*[\w./)]$", s) and ": " not in s and " #" not in s:
        return s
    return json.encode(s)

def add_release_category(repo, title, labels):
    """Adds a category to the release notes GitHub generates, unless one with this title exists.

    Categories go in `changelog.categories` of .github/release.yml, which
    is created if missing. GitHub uses the first category whose labels
    match a pull request, so add catch-alls (`labels: ["*"]`) last.

    Args:
        title: the category heading, such as `Security`.
        labels: the labels that put a pull request in it.
    Returns:
        True when the file changed.
    Example:
        before .github/release.yml:
            changelog:
              categories:
                - title: Features
                  labels: [feature]
        call:
            add_release_category(repo, "Dependencies", ["dependencies"])
        after .github/release.yml:
            changelog:
              categories:
                - title: Features
                  labels: [feature]
                - title: Dependencies
                  labels: [dependencies]
        returns:
            True
    """
    content = repo.read(RELEASE, default = "")
    if re.matches(r"(?m)^\s*-?\s*title:\s*[\"']?" + re.escape(title) + r"[\"']?\s*$", content):
        return False
    item = ["title: " + _plain(title), "labels: [" + ", ".join([_plain(l) for l in labels]) + "]"]
    lines = content.split("\n") if content else []
    at = -1
    indent = None
    for i, line in enumerate(lines):
        if re.matches(r"^\s+categories:\s*$", line):
            at = i
            base = len(line) - len(line.lstrip(" "))
            indent = " " * (base + 2)
            for j in range(i + 1, len(lines)):
                s = lines[j]
                if s.strip() and len(s) - len(s.lstrip(" ")) <= base:
                    break
                if s.strip():
                    at = j
                    if s.lstrip().startswith("-") and indent == " " * (base + 2):
                        indent = s[:len(s) - len(s.lstrip(" "))]
            break
    if at < 0:
        while lines and lines[-1].strip() == "":
            lines = lines[:-1]
        if not re.matches(r"(?m)^changelog:", content):
            lines = lines + ["changelog:"]
        lines = lines + ["  categories:"]
        at, indent = len(lines) - 1, "    "
    new = [indent + "- " + item[0], indent + "  " + item[1]]
    lines = lines[:at + 1] + new + lines[at + 1:]
    text = "\n".join(lines)
    repo.write(RELEASE, text if text.endswith("\n") else text + "\n")
    return True

def set_label(repo, name, color, description = None, path = ".github/labels.yml"):
    """Adds or updates a label in a label sync file, such as .github/labels.yml.

    The file is a list of labels with `name`, `color` and `description`, as
    used by label sync actions and by Probot Settings under `labels:`. An
    existing label keeps its place and gets the new color and description,
    keeping the quotes it was written with; a new label's color is quoted.

    Args:
        name: the label name.
        color: the hex color without `#`, such as `d73a4a`.
        description: the label description, or None to leave it.
        path: the label file.
    Returns:
        True when the file changed. The file is created if needed.
    Example:
        before .github/labels.yml:
            - name: bug
              color: d73a4a
            - name: fleet
              color: "000000"
        call:
            set_label(repo, "fleet", "1f6feb", description = "Opened by gh fleet")
        after .github/labels.yml:
            - name: bug
              color: d73a4a
            - name: fleet
              color: "1f6feb"
              description: Opened by gh fleet
        returns:
            True
    """
    content = repo.read(path, default = None)
    if content != None and re.matches(r"(?m)^\s*-?\s*name:\s*[\"']?" + re.escape(name) + r"[\"']?\s*$", content):
        a = _set_sibling(repo, path, "name", name, "color", color)
        b = description != None and _set_sibling(repo, path, "name", name, "description", description)
        return a or b
    # Always quote the color: 000000 or 1e10 would otherwise read as a number.
    item = ["- name: " + _plain(name), "  color: " + json.encode(color)]
    if description != None:
        item.append("  description: " + _plain(description))
    lines = (content or "").split("\n")
    while lines and lines[-1].strip() == "":
        lines = lines[:-1]
    repo.write(path, "\n".join(lines + item) + "\n")
    return True

def set_funding(repo, platform, value):
    """Sets a sponsor link in .github/FUNDING.yml, such as `github: [acme]`.

    Args:
        platform: the platform key, such as `github`, `open_collective` or
            `custom`.
        value: the account, or a list of accounts or URLs.
    Returns:
        True when the file changed. The file is created if needed.
    Example:
        before .github/FUNDING.yml:
            github: [old-maintainer]
        call:
            set_funding(repo, "github", ["acme"])
            set_funding(repo, "open_collective", "acme")
        after .github/FUNDING.yml:
            github: [acme]
            open_collective: acme
        returns:
            True
    """
    return _set(repo, ".github/FUNDING.yml", platform, value)
