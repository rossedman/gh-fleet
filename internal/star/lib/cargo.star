"""Bump Rust crate dependencies and rust-version in Cargo.toml files.

Edits are line-based, so comments, features and table layout stay as they
were. Cargo.lock is not touched: follow with a `run` step such as
`cargo update -p <crate>` when the lockfile is committed.
"""

def manifests(repo):
    """Lists the repo's Cargo manifests.

    Returns:
        The sorted paths of every Cargo.toml outside target directories.
    Example:
        before Cargo.toml:
            [workspace]
        before crates/core/Cargo.toml:
            [package]
        before target/package/x/Cargo.toml:
            [package]
        call:
            manifests(repo)
        returns:
            ["Cargo.toml", "crates/core/Cargo.toml"]
    """
    return repo.glob("**/Cargo.toml", exclude = ["**/target/**"])

def _lit(s):
    return s.replace("$", "$$")

def set_dependency(repo, name, version):
    """Sets a crate's version wherever a Cargo.toml already depends on it.

    A newer version already there is kept, so a campaign never lowers one.

    Covers `[dependencies]`, `[dev-dependencies]`, `[build-dependencies]`,
    `[workspace.dependencies]` and their `[target.'...'.dependencies]` forms,
    written as `name = "1.0"` or `name = { version = "1.0", ... }`.
    Dependencies given only by `path` or `git` have no version and are left
    alone.

    Args:
        name: the crate name.
        version: the new version requirement, such as `1.0.210`.
    Returns:
        The list of paths that changed.
    Example:
        before Cargo.toml:
            [dependencies]
            serde = { version = "1.0.100", features = ["derive"] }
            tokio = "1.30"

            [dev-dependencies]
            serde = "1.0"
        call:
            set_dependency(repo, "serde", "1.0.210")
        after Cargo.toml:
            [dependencies]
            serde = { version = "1.0.210", features = ["derive"] }
            tokio = "1.30"

            [dev-dependencies]
            serde = "1.0.210"
        returns:
            ["Cargo.toml"]
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
                    if g and g[1] != version and not semver.lowers(g[1], version):
                        line = g[0] + version + g[2]
                        break
            out.append(line)
        updated = "\n".join(out)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed

def rust_version(repo, path = "Cargo.toml"):
    """Reads the `rust-version` of a Cargo.toml.

    Args:
        path: the Cargo.toml to read.
    Returns:
        The version, such as `1.70`, or None.
    Example:
        before Cargo.toml:
            [package]
            name = "api"
            rust-version = "1.70"
        call:
            rust_version(repo)
        returns:
            "1.70"
    """
    g = re.groups(r"""(?m)^\s*rust-version\s*=\s*"([^"]*)\"""", repo.read(path, default = ""))
    return g[0] if g else None

def set_rust_version(repo, version):
    """Raises `rust-version` in every Cargo.toml that sets it, never lowering it.

    Args:
        version: the minimum Rust version, such as `1.75`.
    Returns:
        The list of paths that changed.
    Example:
        before Cargo.toml:
            [package]
            name = "api"
            rust-version = "1.70"
        call:
            set_rust_version(repo, "1.75")
        after Cargo.toml:
            [package]
            name = "api"
            rust-version = "1.75"
        returns:
            ["Cargo.toml"]
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
