"""Read and raise the go directive in go.mod files."""

def modules(repo):
    """Lists the repo's Go modules.

    Returns:
        The sorted paths of every go.mod outside vendor and testdata
        directories.
    Example:
        before go.mod:
            module acme/api
        before tools/go.mod:
            module acme/tools
        before vendor/x/go.mod:
            module x
        call:
            modules(repo)
        returns:
            ["go.mod", "tools/go.mod"]
    """
    return repo.glob("**/go.mod", exclude = ["**/vendor/**", "**/testdata/**"])

def go_version(repo, path = "go.mod"):
    """Reads the version in a go.mod's `go` directive.

    Args:
        path: the go.mod to read.
    Returns:
        The version, such as `1.22.1`, or None when the file has no `go`
        directive or does not exist.
    Example:
        before go.mod:
            module acme/api

            go 1.22.1
        call:
            go_version(repo)
        returns:
            "1.22.1"
    """
    g = re.groups(r"(?m)^go\s+(\d+(?:\.\d+)*)", repo.read(path, default = ""))
    return g[0] if g else None

def set_go_version(repo, version, paths = None):
    """Raises the `go` directive in every go.mod, never lowering it.

    A go.mod without a directive gets one after its `module` line. A
    `toolchain` line the new version covers is removed; a newer one is kept.

    Args:
        version: the Go version, such as `1.27.3`.
        paths: the go.mod files to change; every module by default.
    Returns:
        The list of paths that changed.
    Example:
        before go.mod:
            module acme/api

            go 1.22.1

            toolchain go1.22.5
        before tools/go.mod:
            module acme/tools

            go 1.28
        call:
            set_go_version(repo, "1.27.3")
        after go.mod:
            module acme/api

            go 1.27.3
        returns:
            ["go.mod"]
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
    """Reads every module a go.mod requires, in single-line and block form.

    Args:
        path: the go.mod to read.
    Returns:
        A dict of module path to version, in file order; empty when the file
        does not exist.
    Example:
        before go.mod:
            module acme/api

            require (
                golang.org/x/net v0.20.0 // indirect
                github.com/spf13/cobra v1.8.0
            )
        call:
            requires(repo)
        returns:
            {"golang.org/x/net": "v0.20.0", "github.com/spf13/cobra": "v1.8.0"}
    """
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
    """Finds the modules that require a dependency below a version.

    Use it to pick where a `run` step should run `go get module@version`.

    Args:
        module: the dependency's module path.
        version: the version it should be at least, such as `v0.33.0`.
    Returns:
        The directories of the matching go.mod files; the repo root is `.`.
    Example:
        before go.mod:
            module acme/api

            require golang.org/x/net v0.20.0
        before tools/go.mod:
            module acme/tools

            require golang.org/x/net v0.40.0
        call:
            outdated(repo, "golang.org/x/net", "v0.33.0")
        returns:
            ["."]
    """
    dirs = []
    for path in modules(repo):
        have = requires(repo, path).get(module)
        if have != None and semver.less(have, version):
            dirs.append(path.removesuffix("go.mod").rstrip("/") or ".")
    return dirs
