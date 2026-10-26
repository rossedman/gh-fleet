"""Set language versions in version files: .nvmrc, .python-version, .tool-versions and friends."""

# Each runtime's dedicated version files and its name in .tool-versions.
RUNTIMES = {
    "node": ([".nvmrc", ".node-version"], "nodejs"),
    "python": ([".python-version"], "python"),
    "ruby": ([".ruby-version"], "ruby"),
    "go": ([".go-version"], "golang"),
    "java": ([".java-version"], "java"),
    "terraform": ([".terraform-version"], "terraform"),
    "elixir": ([], "elixir"),
    "erlang": ([], "erlang"),
    "rust": ([], "rust"),
}

def _runtime(name):
    if name not in RUNTIMES:
        fail("runtime: unknown runtime %r; have %s" % (name, ", ".join(sorted(RUNTIMES.keys()))))
    return RUNTIMES[name]

def version(repo, runtime):
    """Reads the version a repo pins for a runtime.

    Dedicated files such as .nvmrc are read before .tool-versions.

    Args:
        runtime: one of `node`, `python`, `ruby`, `go`, `java`, `terraform`,
            `elixir`, `erlang` or `rust`.
    Returns:
        The version as written, or None when nothing pins it.
    Example:
        before .tool-versions:
            nodejs 20.11.0
            python 3.12.1
        call:
            version(repo, "node")
        returns:
            "20.11.0"
    """
    files, tool = _runtime(runtime)
    for path in files:
        content = repo.read(path, default = "").strip()
        if content:
            return content.splitlines()[0].strip()
    for line in repo.read(".tool-versions", default = "").splitlines():
        fields = line.split("#")[0].split()
        if len(fields) >= 2 and fields[0] == tool:
            return fields[1]
    return None

def set_version(repo, runtime, version, create = None):
    """Sets a runtime's version in every version file the repo already has.

    Updates the dedicated files (.nvmrc and .node-version for node,
    .python-version for python, and so on) and the runtime's line in
    .tool-versions. A repo that pins the runtime nowhere is left alone unless
    `create` is set.

    Args:
        runtime: one of `node`, `python`, `ruby`, `go`, `java`, `terraform`,
            `elixir`, `erlang` or `rust`.
        version: the version to pin, such as `22` or `3.12.7`.
        create: a file to write, such as `.nvmrc`, when nothing pins the
            runtime yet.
    Returns:
        The list of paths that changed.
    Example:
        before .nvmrc:
            20
        before .tool-versions:
            nodejs 20.11.0
            python 3.12.1
        call:
            set_version(repo, "node", "22")
        after .nvmrc:
            22
        after .tool-versions:
            nodejs 22
            python 3.12.1
        returns:
            [".nvmrc", ".tool-versions"]
    """
    files, tool = _runtime(runtime)
    changed = []
    pinned = False
    for path in files:
        content = repo.read(path, default = None)
        if content == None:
            continue
        pinned = True
        if content.strip() != version:
            repo.write(path, version + "\n")
            changed.append(path)
    tv = repo.read(".tool-versions", default = None)
    if tv != None:
        lines = tv.split("\n")
        for i, line in enumerate(lines):
            g = re.groups(r"^(\s*" + re.escape(tool) + r"\s+)(\S+)(.*)$", line)
            if g:
                pinned = True
                if g[1] != version:
                    lines[i] = g[0] + version + g[2]
        updated = "\n".join(lines)
        if updated != tv:
            repo.write(".tool-versions", updated)
            changed.append(".tool-versions")
    if create and not pinned:
        repo.write(create, version + "\n")
        changed.append(create)
    return changed
