"""Find package.json files and edit them as data, keeping key order and indentation.

These helpers edit package.json only. To refresh package-lock.json, follow
the Starlark step with a `run` step such as
`npm install --package-lock-only --ignore-scripts`.
"""

SECTIONS = ["dependencies", "devDependencies", "peerDependencies", "optionalDependencies"]

def packages(repo):
    """Returns the paths of every package.json outside node_modules."""
    return repo.glob("**/package.json", exclude = ["**/node_modules/**"])

def read_package(repo, path = "package.json"):
    """Returns a package.json decoded into a dict whose keys keep the file's order."""
    return json.decode(repo.read(path))

def write_package(repo, path, pkg):
    """Writes `pkg` to `path` as JSON, using the indentation the file already has (two spaces for a new file)."""
    indent = "  "
    g = re.groups(r"\n([ \t]+)\"", repo.read(path, default = ""))
    if g:
        indent = g[0]
    repo.write(path, json.encode_indent(pkg, indent = indent) + "\n")

def set_dependency(repo, name, version, sections = SECTIONS):
    """Sets `name` to `version` in each package.json that already lists it in one of `sections`.

    Packages that do not depend on `name` are left alone, so this never
    adds a dependency. Returns the paths that changed.
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
