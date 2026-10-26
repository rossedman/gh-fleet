"""Read and set variables in Makefiles, such as `GO_VERSION ?= 1.22`.

Edits are line-based and keep the assignment operator (`=`, `:=`, `::=`,
`?=`, `+=`), any `export` or `override` prefix, and trailing comments.
"""

def makefiles(repo):
    """Lists the repo's Makefiles.

    Returns:
        The sorted paths of every Makefile, GNUmakefile, makefile and *.mk
        file outside vendor and node_modules.
    Example:
        before Makefile:
            all:
        before build/common.mk:
            X = 1
        before vendor/x/Makefile:
            all:
        call:
            makefiles(repo)
        returns:
            ["Makefile", "build/common.mk"]
    """
    return repo.glob("**/Makefile", "**/makefile", "**/GNUmakefile", "**/*.mk", exclude = ["**/vendor/**", "**/node_modules/**"])

def _assign(name):
    return r"^(\s*(?:(?:export|override)\s+)*" + re.escape(name) + r"\s*(?:::=|:=|\?=|\+=|!=|=)\s*)(.*?)(\s*#.*)?$"

def get_var(repo, path, name):
    """Reads the value of the first assignment to a variable.

    Args:
        path: the Makefile to read.
        name: the variable name.
    Returns:
        The value as written, or None when the file doesn't assign it.
    Example:
        before Makefile:
            GO_VERSION ?= 1.22.1  # keep in sync with go.mod
            BIN := bin/app
        call:
            get_var(repo, "Makefile", "GO_VERSION")
        returns:
            "1.22.1"
    """
    for line in repo.read(path, default = "").split("\n"):
        g = re.groups(_assign(name), line)
        if g:
            return g[1]
    return None

def set_var(repo, name, value, files = None):
    """Sets every assignment to a variable, in every Makefile that has one.

    Makefiles that don't assign the variable are left alone. Appends
    (`+=`) are left alone too, since they add to a value rather than set it.

    Args:
        name: the variable name.
        value: the new value, as it should appear after the operator.
        files: the Makefiles to edit; every Makefile by default.
    Returns:
        The list of paths that changed.
    Example:
        before Makefile:
            GO_VERSION ?= 1.22.1  # keep in sync with go.mod
            export GOFLAGS = -mod=mod
        call:
            set_var(repo, "GO_VERSION", "1.27.3")
        after Makefile:
            GO_VERSION ?= 1.27.3  # keep in sync with go.mod
            export GOFLAGS = -mod=mod
        returns:
            ["Makefile"]
    """
    changed = []
    for path in files or makefiles(repo):
        content = repo.read(path)
        lines = content.split("\n")
        for i, line in enumerate(lines):
            g = re.groups(_assign(name), line)
            if g and "+=" not in g[0] and g[1] != value:
                lines[i] = g[0] + value + g[2]
        updated = "\n".join(lines)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed
