"""Bump pinned Python requirements and the supported Python version.

Edits are line-based, so comments, ordering and markers in
requirements files and pyproject.toml stay as they were.
"""

def _norm(name):
    # PEP 503: names compare case-insensitively, with runs of -_. equal.
    return re.sub(r"[-_.]+", "-", name).lower()

def requirements_files(repo):
    """Lists the repo's requirements files.

    Returns:
        The sorted paths of every requirements*.txt, and every .txt in a
        requirements directory, outside virtualenvs and node_modules.
    Example:
        before requirements.txt:
            flask
        before requirements/dev.txt:
            pytest
        before .venv/requirements.txt:
            flask
        call:
            requirements_files(repo)
        returns:
            ["requirements.txt", "requirements/dev.txt"]
    """
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
    """Sets a package's version specifier wherever it is already required.

    Changes requirements*.txt lines and the quoted entries of pyproject.toml
    dependency lists, keeping extras such as `[socks]`, environment markers
    and comments. Names match the way pip matches them: `Django`, `django`
    and `DJANGO` are the same package. Files that do not mention the package
    are left alone.

    Args:
        name: the package name.
        spec: a full PEP 440 specifier, such as `==4.2.16` or `>=2.32,<3`.
    Returns:
        The list of paths that changed.
    Example:
        before requirements.txt:
            Django==4.2.1  # web
            requests[socks]>=2.31
        before pyproject.toml:
            [project]
            dependencies = ["django>=4.0"]
        call:
            set_requirement(repo, "django", "==4.2.16")
        after requirements.txt:
            Django==4.2.16  # web
            requests[socks]>=2.31
        after pyproject.toml:
            [project]
            dependencies = ["django==4.2.16"]
        returns:
            ["requirements.txt", "pyproject.toml"]
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
    """Reads the `requires-python` of a pyproject.toml.

    Args:
        path: the pyproject.toml to read.
    Returns:
        The specifier, such as `>=3.9`, or None.
    Example:
        before pyproject.toml:
            [project]
            requires-python = ">=3.9"
        call:
            requires_python(repo)
        returns:
            ">=3.9"
    """
    g = re.groups(r"""(?m)^\s*requires-python\s*=\s*["']([^"']*)["']""", repo.read(path, default = ""))
    return g[0] if g else None

def set_requires_python(repo, spec, path = "pyproject.toml"):
    """Sets `requires-python` in a pyproject.toml that already has it.

    Args:
        spec: the new specifier, such as `>=3.12`.
        path: the pyproject.toml to edit.
    Returns:
        True when the file changed.
    Example:
        before pyproject.toml:
            [project]
            requires-python = ">=3.9"
        call:
            set_requires_python(repo, ">=3.12")
        after pyproject.toml:
            [project]
            requires-python = ">=3.12"
        returns:
            True
    """
    content = repo.read(path, default = None)
    if content == None or requires_python(repo, path) in (None, spec):
        return False
    updated = re.sub(r"""(?m)^(\s*requires-python\s*=\s*["'])[^"']*(["'])""", "${1}" + spec.replace("$", "$$") + "${2}", content, count = 1)
    repo.write(path, updated)
    return True
