"""Find package.json files and edit them as data, keeping key order and indentation.

These helpers edit package.json only. To refresh package-lock.json, follow
the Starlark step with a `run` step such as
`npm install --package-lock-only --ignore-scripts`.
"""

load("./jsonfile.star", _write = "write")

SECTIONS = ["dependencies", "devDependencies", "peerDependencies", "optionalDependencies"]

def packages(repo):
    """Lists the repo's package.json files.

    Returns:
        The sorted paths of every package.json outside node_modules.
    Example:
        before package.json:
            {"name": "web"}
        before packages/ui/package.json:
            {"name": "ui"}
        before node_modules/x/package.json:
            {"name": "x"}
        call:
            packages(repo)
        returns:
            ["package.json", "packages/ui/package.json"]
    """
    return repo.glob("**/package.json", exclude = ["**/node_modules/**"])

def read_package(repo, path = "package.json"):
    """Reads a package.json as a dict whose keys keep the file's order.

    Args:
        path: the package.json to read.
    Returns:
        The decoded package.json.
    Example:
        before package.json:
            {"name": "web", "private": true}
        call:
            read_package(repo)["name"]
        returns:
            "web"
    """
    return json.decode(repo.read(path))

def write_package(repo, path, pkg):
    """Writes a dict as a package.json, keeping the file's indentation.

    The indentation the file already uses is detected; a new file gets two
    spaces. Key order is kept, so only what you changed shows in the diff.

    Args:
        path: the package.json to write.
        pkg: the package, usually from `read_package`.
    Returns:
        True when the file changed.
    Example:
        before package.json:
            {
                "name": "web"
            }
        call:
            pkg = read_package(repo)
            pkg["private"] = True
            write_package(repo, "package.json", pkg)
        after package.json:
            {
                "name": "web",
                "private": true
            }
        returns:
            True
    """
    return _write(repo, path, pkg)

def set_dependency(repo, name, version, sections = SECTIONS):
    """Sets a dependency's version range in every package.json that already lists it.

    Packages that do not depend on `name` are left alone, so this never adds
    a dependency. package-lock.json is not touched; follow with a `run` step
    such as `npm install --package-lock-only --ignore-scripts`.

    Args:
        name: the package name.
        version: the new range, such as `^1.7.9`.
        sections: the dependency sections to look in.
    Returns:
        The list of paths that changed.
    Example:
        before package.json:
            {
              "dependencies": {
                "axios": "^1.6.0"
              }
            }
        before web/package.json:
            {"name": "web"}
        call:
            set_dependency(repo, "axios", "^1.7.9")
        after package.json:
            {
              "dependencies": {
                "axios": "^1.7.9"
              }
            }
        returns:
            ["package.json"]
    """
    changed = []
    for path in packages(repo):
        pkg = read_package(repo, path)
        dirty = False
        for section in sections:
            deps = pkg.get(section)
            if type(deps) == "dict" and name in deps and deps[name] != version:
                deps[name] = version
                dirty = True
        if dirty:
            write_package(repo, path, pkg)
            changed.append(path)
    return changed
