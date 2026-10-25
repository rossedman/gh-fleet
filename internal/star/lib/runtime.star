"""Set language versions in version files: .nvmrc, .python-version, .tool-versions and friends."""

# Each runtime's dedicated version files and its name in .tool-versions.
RUNTIMES = {
    "node": ([".nvmrc", ".node-version"], "nodejs"),
    "python": ([".python-version"], "python"),
    "ruby": ([".ruby-version"], "ruby"),
    "go": ([".go-version"], "golang"),
    "java": ([".java-version"], "java"),
    "terraform": ([".terraform-version"], "terraform"),
}

def _runtime(name):
    if name not in RUNTIMES:
        fail("runtime: unknown runtime %r; have %s" % (name, ", ".join(sorted(RUNTIMES.keys()))))
    return RUNTIMES[name]

def version(repo, runtime):
    """Returns the version a repo pins for `runtime` (such as "node"), or None.

    Dedicated files such as .nvmrc are read before .tool-versions.
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
    """Sets `runtime`'s version in every version file the repo already has.

    Updates the dedicated files (.nvmrc and .node-version for node,
    .python-version for python, and so on) and the runtime's line in
    .tool-versions. A repo that pins the runtime nowhere is left alone,
    unless `create` names a file to write, such as ".nvmrc".

    Returns the paths that changed.
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
