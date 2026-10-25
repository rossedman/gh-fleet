"""Bump pinned Python requirements and the supported Python version.

Edits are line-based, so comments, ordering and markers in
requirements files and pyproject.toml stay as they were.
"""

def _norm(name):
    # PEP 503: names compare case-insensitively, with runs of -_. equal.
    return re.sub(r"[-_.]+", "-", name).lower()

def requirements_files(repo):
    """Returns the paths of every requirements*.txt outside virtualenvs and node_modules."""
    return repo.glob(
        "**/requirements*.txt",
        "**/requirements/*.txt",
        exclude = ["**/node_modules/**", "**/.venv/**", "**/venv/**"],
    )

def _set_in_line(line, name, spec, quoted):
    # Returns line with the version spec of name replaced, or line unchanged.
    if quoted:
        pattern = r"""^(.*?["'])([A-Za-z0-9][A-Za-z0-9._-]*)(\[[^\]]*\])?\s*([<>=!~][^;"']*)?(\s*;[^"']*)?(["'].*)$"""
    else:
        pattern = r"""^(\s*)([A-Za-z0-9][A-Za-z0-9._-]*)(\[[^\]]*\])?\s*([<>=!~][^;#]*?)?(\s*(?:;[^#]*)?)(\s*(?:#.*)?)$"""
    g = re.groups(pattern, line)
    if not g or _norm(g[1]) != _norm(name):
        return line
    if (g[3] or "").strip() == spec:
        return line
    return g[0] + g[1] + g[2] + spec + g[4] + g[5]

def set_requirement(repo, name, spec):
    """Sets the version spec of `name` wherever it is already required.

    `spec` is a full PEP 440 specifier such as `==4.2.16` or `>=2.32,<3`.
    It changes requirements*.txt lines and the quoted entries of
    pyproject.toml dependency lists (`"django>=4.2"`), keeping extras such
    as `[socks]`, environment markers and comments. Names match the way pip
    matches them: `Django`, `django` and `DJANGO` are the same package.
    Files that do not mention `name` are left alone.

    Returns the paths that changed.
    """
    changed = []
    files = [(p, False) for p in requirements_files(repo)]
    files += [(p, True) for p in repo.glob("**/pyproject.toml", exclude = ["**/node_modules/**", "**/.venv/**"])]
    for path, quoted in files:
        content = repo.read(path)
        out = [_set_in_line(line, name, spec, quoted) for line in content.split("\n")]
        updated = "\n".join(out)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed

def requires_python(repo, path = "pyproject.toml"):
    """Returns the `requires-python` value of a pyproject.toml, or None."""
    g = re.groups(r"""(?m)^\s*requires-python\s*=\s*["']([^"']*)["']""", repo.read(path, default = ""))
    return g[0] if g else None

def set_requires_python(repo, spec, path = "pyproject.toml"):
    """Sets `requires-python` in a pyproject.toml that already has it.

    Returns True when the file changed.
    """
    content = repo.read(path, default = None)
    if content == None or requires_python(repo, path) in (None, spec):
        return False
    updated = re.sub(r"""(?m)^(\s*requires-python\s*=\s*["'])[^"']*(["'])""", "${1}" + spec.replace("$", "$$") + "${2}", content, count = 1)
    repo.write(path, updated)
    return True
