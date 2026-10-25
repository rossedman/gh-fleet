"""Read and raise the go directive in go.mod files."""

def modules(repo):
    """Returns the paths of every go.mod outside vendor and testdata directories."""
    return repo.glob("**/go.mod", exclude = ["**/vendor/**", "**/testdata/**"])

def go_version(repo, path = "go.mod"):
    """Returns the version in a go.mod's `go` directive, or None when it has none or does not exist."""
    g = re.groups(r"(?m)^go\s+(\d+(?:\.\d+)*)", repo.read(path, default = ""))
    return g[0] if g else None

def set_go_version(repo, version, paths = None):
    """Raises the `go` directive to `version` in every go.mod, never lowering it.

    `paths` limits the change to some go.mod files; by default it is every
    module from `modules(repo)`. A `toolchain` line older than or equal to
    the new version is removed, since the go directive now covers it; a
    newer one is kept.

    Returns the paths that changed.
    """
    changed = []
    for path in paths or modules(repo):
        content = repo.read(path)
        current = go_version(repo, path)
        if current != None and not semver.less(current, version):
            continue
        if current == None:
            updated = re.sub(r"(?m)^(module\s+\S+.*)$", "${1}\n\ngo " + version, content, count = 1)
        else:
            updated = re.sub(r"(?m)^go\s+\S+.*$", "go " + version, content, count = 1)
        tc = re.groups(r"(?m)^toolchain\s+go(\d+(?:\.\d+)*)", updated)
        if tc and not semver.less(version, tc[0]):
            updated = re.sub(r"\n\ntoolchain\s+\S+[ \t]*\n\n", "\n\n", updated)
            updated = re.sub(r"\n\ntoolchain\s+\S+[ \t]*\n?$", "\n", updated)
            updated = re.sub(r"(?m)^toolchain\s+\S+[ \t]*\n", "", updated)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed

def requires(repo, path = "go.mod"):
    """Returns a dict of every module a go.mod requires to its version, single-line and block forms alike."""
    out = {}
    block = False
    for line in repo.read(path, default = "").splitlines():
        line = line.split("//")[0].strip()
        if line.startswith("require ("):
            block = True
            continue
        if block and line == ")":
            block = False
            continue
        fields = line.split()
        if block and len(fields) >= 2:
            out[fields[0]] = fields[1]
        elif len(fields) >= 3 and fields[0] == "require":
            out[fields[1]] = fields[2]
    return out

def outdated(repo, module, version):
    """Returns the directories of the go.mod files that require `module` below `version`.

    The repo root is `.`. Use it to pick where a `run` step should run
    `go get module@version`.
    """
    dirs = []
    for path in modules(repo):
        have = requires(repo, path).get(module)
        if have != None and semver.less(have, version):
            dirs.append(path.removesuffix("go.mod").rstrip("/") or ".")
    return dirs
