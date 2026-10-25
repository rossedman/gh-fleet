"""Find GitHub Actions workflows, bump action versions, set inputs and add permissions.

Edits are line-based rather than a YAML round trip, so comments, quoting
and formatting stay exactly as they were.
"""

def _lit(s):
    # Escapes $ for use in a re.sub replacement.
    return s.replace("$", "$$")

def workflows(repo):
    """Returns the paths of the repo's workflow files under .github/workflows."""
    return repo.glob(".github/workflows/*.yml", ".github/workflows/*.yaml")

def action_files(repo):
    """Returns the workflows plus every local action's action.yml or action.yaml."""
    return sorted(set(workflows(repo) + repo.glob("**/action.yml", "**/action.yaml")))

def bump_action(repo, action, to, from_ref = None):
    """Rewrites `uses: <action>@<ref>` to `uses: <action>@<to>` in workflows and local actions.

    `action` is `owner/repo`; steps that use a sub-path (`owner/repo/path`)
    match too. Refs pinned to a full commit SHA are left alone. With
    `from_ref`, only that exact ref is replaced, so `v3` does not match
    `v3.1`.

    Returns the paths that changed.
    """
    pattern = r"""(uses:\s*["']?)""" + re.escape(action) + r"""((?:/[^@\s"']*)?)@([^\s"'#]+)"""
    changed = []
    for path in action_files(repo):
        content = repo.read(path)
        out = []
        dirty = False
        for line in content.split("\n"):
            g = re.groups(pattern, line)
            if g and g[2] != to and not re.matches(r"^[0-9a-f]{40}$", g[2]) and (from_ref == None or g[2] == from_ref):
                line = re.sub(pattern, "${1}" + _lit(action) + "${2}@" + _lit(to), line, count = 1)
                dirty = True
            out.append(line)
        if dirty:
            repo.write(path, "\n".join(out))
            changed.append(path)
    return changed

def set_input(repo, key, value, match = None):
    """Sets every `<key>: <value>` line in workflows, such as `go-version` or `node-version`.

    Quotes around the old value are kept. With `match`, a regular
    expression, only current values it matches are replaced; for example
    `match = r"^\\d"` leaves `go-version: stable` alone.

    Returns the paths that changed.
    """
    pattern = r"""^(\s*-?\s*""" + re.escape(key) + r""":\s*["']?)([^"'\s#]+)"""
    changed = []
    for path in workflows(repo):
        content = repo.read(path)
        out = []
        for line in content.split("\n"):
            g = re.groups(pattern, line)
            if g and g[1] != value and (match == None or re.matches(match, g[1])):
                line = re.sub(pattern, "${1}" + _lit(value), line, count = 1)
            out.append(line)
        updated = "\n".join(out)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed

def ensure_permissions(repo, permissions = {"contents": "read"}):
    """Adds a top-level `permissions` block to each workflow that has none.

    The block goes right above the top-level `jobs:` key. Workflows that
    already set top-level permissions, even broader ones, are left alone.

    Returns the paths that changed.
    """
    block = "permissions:\n" + "".join(["  %s: %s\n" % (k, v) for k, v in permissions.items()]) + "\n"
    changed = []
    for path in workflows(repo):
        content = repo.read(path)
        if re.matches(r"(?m)^permissions:", content) or not re.matches(r"(?m)^jobs:", content):
            continue
        head, sep, tail = content.partition("\njobs:")
        if content.startswith("jobs:"):
            updated = block + content
        else:
            updated = head + "\n" + block + sep.removeprefix("\n") + tail
        repo.write(path, updated)
        changed.append(path)
    return changed
