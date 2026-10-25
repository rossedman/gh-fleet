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
    """Reads the revision each hook repository is pinned to.

    Args:
        path: the pre-commit config to read.
    Returns:
        A dict of repository URL to `rev`, in file order.
    Example:
        before .pre-commit-config.yaml:
            repos:
              - repo: https://github.com/pre-commit/pre-commit-hooks
                rev: v4.6.0
                hooks:
                  - id: check-yaml
              - repo: https://github.com/psf/black
                rev: "24.4.0"
                hooks:
                  - id: black
        call:
            revs(repo)
        returns:
            {"https://github.com/pre-commit/pre-commit-hooks": "v4.6.0", "https://github.com/psf/black": "24.4.0"}
    """
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
    """Pins a hook repository to a new revision, keeping quotes and comments.

    The URL matches with or without a trailing `.git`. A config that does
    not use the repository is left alone.

    Args:
        url: the hook repository URL.
        rev: the new revision, such as `v5.0.0`.
        path: the pre-commit config to edit.
    Returns:
        True when the file changed.
    Example:
        before .pre-commit-config.yaml:
            repos:
              - repo: https://github.com/psf/black
                rev: "24.4.0"  # formatter
                hooks:
                  - id: black
        call:
            set_rev(repo, "https://github.com/psf/black", "24.10.0")
        after .pre-commit-config.yaml:
            repos:
              - repo: https://github.com/psf/black
                rev: "24.10.0"  # formatter
                hooks:
                  - id: black
        returns:
            True
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
