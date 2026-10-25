"""Bump Rust crate dependencies and rust-version in Cargo.toml files.

Edits are line-based, so comments, features and table layout stay as they
were. Cargo.lock is not touched: follow with a `run` step such as
`cargo update -p <crate>` when the lockfile is committed.
"""

def manifests(repo):
    """Returns the paths of every Cargo.toml outside target directories."""
    return repo.glob("**/Cargo.toml", exclude = ["**/target/**"])

def _lit(s):
    return s.replace("$", "$$")

def set_dependency(repo, name, version):
    """Sets the version of crate `name` wherever a Cargo.toml already depends on it.

    Covers `[dependencies]`, `[dev-dependencies]`, `[build-dependencies]`,
    `[workspace.dependencies]` and their `[target.'...'.dependencies]`
    forms, written as `name = "1.0"` or `name = { version = "1.0", ... }`.
    Dependencies given by `path` or `git` alone have no version and are left
    alone.

    Returns the paths that changed.
    """
    key = re.escape(name)
    simple = r"""^(\s*""" + key + r"""\s*=\s*")([^"]*)(".*)$"""
    table = r"""^(\s*""" + key + r"""\s*=\s*\{.*?\bversion\s*=\s*")([^"]*)(".*)$"""
    changed = []
    for path in manifests(repo):
        content = repo.read(path)
        out = []
        in_deps = False
        for line in content.split("\n"):
            header = re.groups(r"^\s*\[([^\]]+)\]", line)
            if header:
                in_deps = header[0].strip().endswith("dependencies")
            elif in_deps:
                for pattern in (simple, table):
                    g = re.groups(pattern, line)
                    if g and g[1] != version:
                        line = g[0] + version + g[2]
                        break
            out.append(line)
        updated = "\n".join(out)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed

def rust_version(repo, path = "Cargo.toml"):
    """Returns the `rust-version` of a Cargo.toml, or None."""
    g = re.groups(r"""(?m)^\s*rust-version\s*=\s*"([^"]*)\"""", repo.read(path, default = ""))
    return g[0] if g else None

def set_rust_version(repo, version):
    """Raises `rust-version` to `version` in every Cargo.toml that sets it, never lowering it.

    Returns the paths that changed.
    """
    changed = []
    for path in manifests(repo):
        current = rust_version(repo, path)
        if current == None or not semver.less(current, version):
            continue
        content = repo.read(path)
        repo.write(path, re.sub(r"""(?m)^(\s*rust-version\s*=\s*")[^"]*(")""", "${1}" + _lit(version) + "${2}", content, count = 1))
        changed.append(path)
    return changed
