"""Bump Swift Package Manager dependencies and the swift-tools-version.

Edits are line-based, so comments and layout in Package.swift stay as
they were. Package.resolved isn't updated; run `swift package update
<name>` in a `run` step when it is committed.
"""

def _lit(s):
    return s.replace("$", "$$")

def manifests(repo):
    """Lists the repo's Package.swift files.

    Returns:
        The sorted paths of every Package.swift outside .build directories.
    Example:
        before Package.swift:
            // swift-tools-version:5.9
        before Tools/Package.swift:
            // swift-tools-version:5.9
        before .build/checkouts/x/Package.swift:
            // swift-tools-version:5.5
        call:
            manifests(repo)
        returns:
            ["Package.swift", "Tools/Package.swift"]
    """
    return repo.glob("**/Package.swift", exclude = ["**/.build/**"])

def set_package(repo, url, version):
    """Sets the version of a package dependency wherever a manifest declares it.

    A newer version already there is kept, so a campaign never lowers one.

    Covers `from:`, `exact:`, `.upToNextMajor(from:)` and
    `.upToNextMinor(from:)` requirements; the kind of requirement stays.
    Packages pinned to a branch or revision are left alone. The URL matches
    with or without a trailing `.git`.

    Args:
        url: the package URL, such as `https://github.com/apple/swift-log`.
        version: the new version.
    Returns:
        The list of paths that changed.
    Example:
        before Package.swift:
            dependencies: [
                .package(url: "https://github.com/apple/swift-log.git", from: "1.5.3"),
                .package(url: "https://github.com/vapor/vapor", .upToNextMinor(from: "4.89.0")),
            ]
        call:
            set_package(repo, "https://github.com/apple/swift-log", "1.6.1")
        after Package.swift:
            dependencies: [
                .package(url: "https://github.com/apple/swift-log.git", from: "1.6.1"),
                .package(url: "https://github.com/vapor/vapor", .upToNextMinor(from: "4.89.0")),
            ]
        returns:
            ["Package.swift"]
    """
    base = re.escape(url.removesuffix(".git"))
    pattern = r"""(\.package\s*\(\s*(?:name:\s*"[^"]*"\s*,\s*)?url:\s*\"""" + base + r"""(?:\.git)?"\s*,\s*(?:from:|exact:|\.upToNextMajor\(\s*from:|\.upToNextMinor\(\s*from:|\.exact\()\s*")([^"]+)(")"""
    changed = []
    for path in manifests(repo):
        content = repo.read(path)
        updated = re.sub(pattern, lambda m: m[0] if semver.lowers(m[2], version) else m[1] + version + m[3], content)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed

def set_tools_version(repo, version):
    """Raises the `// swift-tools-version:` line, never lowering it.

    Args:
        version: the Swift tools version, such as `5.10`.
    Returns:
        The list of paths that changed.
    Example:
        before Package.swift:
            // swift-tools-version:5.7
            import PackageDescription
        call:
            set_tools_version(repo, "5.10")
        after Package.swift:
            // swift-tools-version:5.10
            import PackageDescription
        returns:
            ["Package.swift"]
    """
    changed = []
    for path in manifests(repo):
        content = repo.read(path)
        g = re.groups(r"(?m)^//\s*swift-tools-version\s*:\s*(\d+(?:\.\d+)*)", content)
        if not g or not semver.less(g[0], version):
            continue
        updated = re.sub(r"(?m)^(//\s*swift-tools-version\s*:\s*)\d+(?:\.\d+)*", "${1}" + _lit(version), content, count = 1)
        repo.write(path, updated)
        changed.append(path)
    return changed
