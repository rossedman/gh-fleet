"""Manage scanner ignore files: .trivyignore, .semgrepignore and .gitleaksignore.

Each file holds one entry per line: a vulnerability ID for Trivy, a path
pattern for Semgrep, a finding fingerprint for gitleaks. Comments (`#`)
and blank lines are kept. Trivy entries can carry an expiry date
(`CVE-2024-1234 exp:2025-06-30`), after which Trivy reports the finding
again.
"""

_FILES = {
    "gitleaks": ".gitleaksignore",
    "semgrep": ".semgrepignore",
    "trivy": ".trivyignore",
}

def _path(tool):
    if tool not in _FILES:
        fail("scanignore: unknown tool %r; have %s" % (tool, ", ".join(sorted(_FILES.keys()))))
    return _FILES[tool]

def _entry(line):
    line = line.strip()
    if not line or line.startswith("#"):
        return None
    return line.split(" ")[0].split("\t")[0]

def entries(repo, tool):
    """Lists the entries in a scanner's ignore file.

    Args:
        tool: `trivy`, `semgrep` or `gitleaks`.
    Returns:
        The entries in file order, without comments or Trivy expiry dates;
        empty when the file doesn't exist.
    Example:
        before .trivyignore:
            # Not reachable: we never parse untrusted YAML.
            CVE-2024-1234 exp:2025-06-30
            GHSA-xxxx-yyyy-zzzz
        call:
            entries(repo, "trivy")
        returns:
            ["CVE-2024-1234", "GHSA-xxxx-yyyy-zzzz"]
    """
    out = []
    for line in repo.read(_path(tool), default = "").split("\n"):
        e = _entry(line)
        if e != None and e not in out:
            out.append(e)
    return out

def add(repo, tool, items, comment = None, expires = None):
    """Adds entries to a scanner's ignore file, creating it if needed.

    Entries already in the file, with or without an expiry, are skipped.
    New entries go at the end, after a blank line and `comment`.

    Args:
        tool: `trivy`, `semgrep` or `gitleaks`.
        items: the entries to add: vulnerability IDs, path patterns or
            fingerprints.
        comment: why they're ignored, written as a `#` line above them.
        expires: for Trivy, the date (`YYYY-MM-DD`) after which the entries
            stop being ignored.
    Returns:
        The list of entries that were added.
    Example:
        before .trivyignore:
            CVE-2024-1234
        call:
            add(repo, "trivy", ["CVE-2024-1234", "CVE-2025-0001"], comment = "Fixed upstream in 1.4; waiting on the base image.", expires = "2026-12-31")
        after .trivyignore:
            CVE-2024-1234

            # Fixed upstream in 1.4; waiting on the base image.
            CVE-2025-0001 exp:2026-12-31
        returns:
            ["CVE-2025-0001"]
    """
    path = _path(tool)
    if expires != None:
        if tool != "trivy":
            fail("scanignore.add: only trivy supports expires")
        if not re.matches(r"^\d{4}-\d{2}-\d{2}$", expires):
            fail("scanignore.add: expires %r is not a YYYY-MM-DD date" % expires)
    have = entries(repo, tool)
    new = []
    for item in items:
        item = item.strip()
        if item and item not in have and item not in new:
            new.append(item)
    if not new:
        return []
    content = repo.read(path, default = "")
    block = ""
    if content:
        block = "\n" if content.endswith("\n") else "\n\n"
    if comment:
        block += "# " + comment + "\n"
    suffix = " exp:" + expires if expires else ""
    repo.write(path, content + block + "".join([item + suffix + "\n" for item in new]))
    return new

def _drop(repo, path, gone):
    # Removes the lines for which gone(line) is true, along with comment
    # lines directly above them that no longer describe any entry.
    content = repo.read(path, default = None)
    if content == None:
        return []
    lines = content.split("\n")
    keep = [True] * len(lines)
    removed = []
    for i, line in enumerate(lines):
        e = _entry(line)
        if e != None and gone(line):
            keep[i] = False
            if e not in removed:
                removed.append(e)
    if not removed:
        return []

    # A comment block is dropped when every entry under it was removed.
    i = 0
    for _ in range(len(lines)):
        if i >= len(lines):
            break
        if not lines[i].strip().startswith("#"):
            i += 1
            continue
        start = i
        for j in range(i, len(lines)):
            if not lines[j].strip().startswith("#"):
                break
            i = j + 1
        under = []
        for j in range(i, len(lines)):
            if _entry(lines[j]) == None:
                break
            under.append(j)
        if under and not [j for j in under if keep[j]]:
            for j in range(start, i):
                keep[j] = False
    out = [line for i, line in enumerate(lines) if keep[i]]

    # Collapse the blank lines left behind.
    tidy = []
    for line in out:
        if not line.strip() and (not tidy or not tidy[-1].strip()):
            continue
        tidy.append(line)
    text = "\n".join(tidy).strip("\n")
    repo.write(path, text + "\n" if text else "")
    return removed

def remove(repo, tool, items):
    """Removes entries from a scanner's ignore file.

    A comment that only described removed entries goes with them.

    Args:
        tool: `trivy`, `semgrep` or `gitleaks`.
        items: the entries to remove.
    Returns:
        The list of entries that were removed.
    Example:
        before .semgrepignore:
            vendor/
            # Generated by protoc.
            gen/
            testdata/
        call:
            remove(repo, "semgrep", ["gen/", "testdata/"])
        after .semgrepignore:
            vendor/
        returns:
            ["gen/", "testdata/"]
    """
    drop = [item.strip() for item in items]
    return _drop(repo, _path(tool), lambda line: _entry(line) in drop)

def prune_expired(repo, today):
    """Removes .trivyignore entries whose expiry date has passed.

    Trivy already stops honoring them, so this only tidies the file and
    shows which findings came back. Entries without an expiry stay.

    Args:
        today: the current date as `YYYY-MM-DD`; entries expiring before it
            are removed.
    Returns:
        The list of removed vulnerability IDs.
    Example:
        before .trivyignore:
            # Waiting on the base image.
            CVE-2024-1234 exp:2025-06-30
            CVE-2025-0001 exp:2026-12-31
            CVE-2023-9999
        call:
            prune_expired(repo, "2026-01-15")
        after .trivyignore:
            # Waiting on the base image.
            CVE-2025-0001 exp:2026-12-31
            CVE-2023-9999
        returns:
            ["CVE-2024-1234"]
    """
    if not re.matches(r"^\d{4}-\d{2}-\d{2}$", today):
        fail("scanignore.prune_expired: today %r is not a YYYY-MM-DD date" % today)

    def expired(line):
        g = re.groups(r"\bexp:(\d{4}-\d{2}-\d{2})\b", line)
        return bool(g) and g[0] < today

    return _drop(repo, ".trivyignore", expired)
