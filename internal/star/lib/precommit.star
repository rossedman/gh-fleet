"""Read and bump hook revisions in .pre-commit-config.yaml.

Edits are line-based, so comments and layout stay as they were.
"""

CONFIG = ".pre-commit-config.yaml"

def _same(a, b):
    return a.removesuffix(".git").rstrip("/") == b.removesuffix(".git").rstrip("/")

def _unquote(v):
    v = v.split(" #")[0].strip()
    return v.strip("\"'")

def revs(repo, path = CONFIG):
    """Returns a dict of each hook repository URL to its `rev`, in file order."""
    out = {}
    current = None
    for line in repo.read(path, default = "").splitlines():
        g = re.groups(r"^\s*-\s*repo:\s*(\S+)", line)
        if g:
            current = _unquote(g[0])
            continue
        g = re.groups(r"^\s*rev:\s*(.+)$", line)
        if g and current != None:
            out[current] = _unquote(g[0])
            current = None
    return out

def set_rev(repo, url, rev, path = CONFIG):
    """Sets the `rev` of the hook repository `url`, keeping quotes and comments.

    `url` matches with or without a trailing `.git`. A config that does
    not use `url` is left alone. Returns True when the file changed.
    """
    content = repo.read(path, default = None)
    if content == None:
        return False
    lines = content.split("\n")
    current = None
    for i, line in enumerate(lines):
        g = re.groups(r"^\s*-\s*repo:\s*(\S+)", line)
        if g:
            current = _unquote(g[0])
            continue
        g = re.groups(r"""^(\s*rev:\s*["']?)([^"'\s#]+)(.*)$""", line)
        if g and current != None and _same(current, url):
            if g[1] != rev:
                lines[i] = g[0] + rev + g[2]
            current = None
    updated = "\n".join(lines)
    if updated == content:
        return False
    repo.write(path, updated)
    return True
