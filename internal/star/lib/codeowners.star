"""Find and edit CODEOWNERS files."""

LOCATIONS = [".github/CODEOWNERS", "CODEOWNERS", "docs/CODEOWNERS"]

def find(repo):
    """Returns the path of the CODEOWNERS file GitHub uses, or None.

    GitHub reads the first of .github/CODEOWNERS, CODEOWNERS and
    docs/CODEOWNERS that exists.
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
    """Returns the owners of the last rule for exactly `pattern`, or an empty list."""
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
    """Makes `owners` the owners of `pattern`, editing the existing rule or adding one.

    `owners` is a list or a space-separated string of users or teams; a
    leading `@` is added when missing. With no CODEOWNERS file, one is
    created at .github/CODEOWNERS, starting with `header` when given.

    Returns True when the file changed.
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
