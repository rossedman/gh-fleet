"""Find and edit CODEOWNERS files."""

LOCATIONS = [".github/CODEOWNERS", "CODEOWNERS", "docs/CODEOWNERS"]

def find(repo):
    """Finds the CODEOWNERS file GitHub uses.

    GitHub reads the first of .github/CODEOWNERS, CODEOWNERS and
    docs/CODEOWNERS that exists.

    Returns:
        Its path, or None when the repo has none.
    Example:
        before docs/CODEOWNERS:
            * @acme/docs
        call:
            find(repo)
        returns:
            "docs/CODEOWNERS"
    """
    for path in LOCATIONS:
        if repo.exists(path):
            return path
    return None

def _owners(owners):
    if type(owners) == "string":
        owners = owners.split()
    return [o if o.startswith("@") or "@" in o else "@" + o for o in owners]

def owners(repo, pattern = "*"):
    """Reads the owners of a CODEOWNERS pattern.

    Args:
        pattern: the pattern exactly as written, such as `*` or `/docs/`.
    Returns:
        The owners of the last rule for the pattern, or an empty list.
    Example:
        before .github/CODEOWNERS:
            * @acme/platform
            /docs/ @acme/writers
        call:
            owners(repo, "/docs/")
        returns:
            ["@acme/writers"]
    """
    path = find(repo)
    if path == None:
        return []
    found = []
    for line in repo.read(path).splitlines():
        fields = line.split("#")[0].split()
        if fields and fields[0] == pattern:
            found = fields[1:]
    return found

def set_owners(repo, pattern, owners, header = None):
    """Makes a list of users or teams the owners of a pattern.

    The last existing rule for the pattern is replaced; otherwise a rule is
    added at the end. With no CODEOWNERS file, one is created at
    .github/CODEOWNERS.

    Args:
        pattern: the pattern, such as `*` or `/docs/`.
        owners: a list or a space-separated string of users or teams; a
            leading `@` is added when missing.
        header: a line, such as a comment, to start a new file with.
    Returns:
        True when the file changed.
    Example:
        before .github/CODEOWNERS:
            * @acme/platform
            /docs/ @acme/writers
        call:
            set_owners(repo, "*", "acme/payments")
        after .github/CODEOWNERS:
            * @acme/payments
            /docs/ @acme/writers
        returns:
            True
    """
    rule = pattern + " " + " ".join(_owners(owners))
    path = find(repo)
    if path == None:
        text = (header + "\n" if header else "") + rule + "\n"
        repo.write(".github/CODEOWNERS", text)
        return True
    content = repo.read(path)
    lines = content.split("\n")
    last = -1
    for i, line in enumerate(lines):
        fields = line.split("#")[0].split()
        if fields and fields[0] == pattern:
            last = i
    if last >= 0:
        lines[last] = rule
        updated = "\n".join(lines)
    else:
        updated = content + ("" if content.endswith("\n") or not content else "\n") + rule + "\n"
    if updated == content:
        return False
    repo.write(path, updated)
    return True
