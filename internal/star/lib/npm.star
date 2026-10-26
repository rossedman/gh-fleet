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

    A newer version already there is kept, so a campaign never lowers one.

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
            if type(deps) == "dict" and name in deps and deps[name] != version and not semver.lowers(deps[name], version):
                deps[name] = version
                dirty = True
        if dirty:
            write_package(repo, path, pkg)
            changed.append(path)
    return changed

def set_package_manager(repo, spec, path = "package.json"):
    """Sets the `packageManager` field Corepack uses, such as `pnpm@9.12.0`, where a package already has one.

    A newer version of the same package manager is kept.

    Args:
        spec: the package manager and version, such as `pnpm@9.12.0` or
            `yarn@4.5.0`.
        path: the package.json to edit.
    Returns:
        True when the file changed.
    Example:
        before package.json:
            {
              "name": "web",
              "packageManager": "pnpm@8.15.4"
            }
        call:
            set_package_manager(repo, "pnpm@9.12.0")
        after package.json:
            {
              "name": "web",
              "packageManager": "pnpm@9.12.0"
            }
        returns:
            True
    """
    if not repo.exists(path):
        return False
    pkg = read_package(repo, path)
    have = pkg.get("packageManager")
    if have == None or have == spec:
        return False
    if have.split("@")[0] == spec.split("@")[0] and semver.lowers(have, spec):
        return False
    pkg["packageManager"] = spec
    return write_package(repo, path, pkg)

def set_npmrc(repo, key, value, path = ".npmrc"):
    """Sets a `key=value` setting in .npmrc, adding it, or the file, when missing.

    Args:
        key: the setting, such as `engine-strict` or `save-exact`.
        value: the value, such as `true`.
        path: the .npmrc to edit.
    Returns:
        True when the file changed.
    Example:
        before .npmrc:
            # registry for @acme packages
            @acme:registry=https://npm.pkg.github.com
            save-exact=false
        call:
            set_npmrc(repo, "save-exact", "true")
            set_npmrc(repo, "engine-strict", "true")
        after .npmrc:
            # registry for @acme packages
            @acme:registry=https://npm.pkg.github.com
            save-exact=true
            engine-strict=true
        returns:
            True
    """
    if type(value) == "bool":
        value = "true" if value else "false"
    value = str(value)
    content = repo.read(path, default = "")
    lines = content.split("\n") if content else []
    if lines and lines[-1] == "":
        lines = lines[:-1]
    for i, line in enumerate(lines):
        g = re.groups(r"^(\s*" + re.escape(key) + r"\s*=\s*)(.*?)\s*$", line)
        if g:
            if g[1] == value:
                return False
            lines[i] = g[0] + value
            repo.write(path, "\n".join(lines) + "\n")
            return True
    repo.write(path, "\n".join(lines + [key + "=" + value]) + "\n")
    return True
