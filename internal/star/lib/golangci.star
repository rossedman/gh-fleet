"""Edit golangci-lint configs: enable and disable linters and set linter settings.

Works on YAML configs (.golangci.yml and .golangci.yaml), in both the v1
and v2 formats; settings go where each version expects them. Edits are
line-based, so comments stay. To move a config from v1 to v2, run the
official `golangci-lint migrate` in a `run` step: it knows every renamed
and merged linter, which a text edit can't.
"""

load("./yamlpath.star", _get = "get", _set = "set", _set_in = "set_in")

FILES = [".golangci.yml", ".golangci.yaml"]

def find(repo):
    """Finds the repo's golangci-lint config.

    Returns:
        The path of .golangci.yml or .golangci.yaml, or None.
    Example:
        before .golangci.yaml:
            linters: {}
        call:
            find(repo)
        returns:
            ".golangci.yaml"
    """
    for path in FILES:
        if repo.exists(path):
            return path
    return None

def config_version(repo):
    """Reads the config format version.

    Returns:
        `"2"` for a v2 config, `"1"` for a config without a `version` key, or
        None when the repo has no config.
    Example:
        before .golangci.yml:
            version: "2"
            linters:
              default: standard
        call:
            config_version(repo)
        returns:
            "2"
    """
    path = find(repo)
    if path == None:
        return None
    return _get(repo, path, "version") or "1"

def _list(lines, key_path):
    # Returns (start, end, indent, items) for a block or flow list at a
    # dotted path, or None. start is the key line; items map name to line.
    parts = key_path.split(".")
    stack = []
    for i, line in enumerate(lines):
        if not line.strip() or line.strip().startswith("#"):
            continue
        indent = len(line) - len(line.lstrip(" "))
        for _ in range(len(stack)):
            if stack and stack[-1][0] >= indent:
                stack.pop()
        g = re.groups(r"^\s*([\w.-]+):\s*(.*?)\s*(#.*)?$", line)
        if not g or line.lstrip().startswith("-"):
            continue
        stack.append((indent, g[0]))
        if [k for _, k in stack] != parts:
            continue
        if g[1].startswith("["):
            items = {}
            for item in g[1].strip("[]").split(","):
                if item.strip():
                    items[item.strip().strip("\"'")] = i
            return (i, i, indent, items, True)
        items = {}
        end = i
        item_indent = indent + 2
        for j in range(i + 1, len(lines)):
            s = lines[j]
            if not s.strip() or s.strip().startswith("#"):
                continue
            lead = len(s) - len(s.lstrip(" "))
            if lead <= indent and not s.lstrip().startswith("-"):
                break
            if lead < indent:
                break
            m = re.groups(r"""^(\s*)-\s*["']?([\w.-]+)["']?""", s)
            if m:
                items[m[1]] = j
                item_indent = len(m[0])
                end = j
            else:
                break
        return (i, end, item_indent, items, False)
    return None

def _add(lines, path, name):
    found = _list(lines, path)
    if found == None:
        return None
    start, end, indent, items, flow = found
    if name in items:
        return lines
    if flow:
        g = re.groups(r"^(.*\[)(.*?)(\].*)$", lines[start])
        inner = g[1].strip()
        lines[start] = g[0] + (inner + ", " if inner else "") + name + g[2]
        return lines
    return lines[:end + 1] + [" " * indent + "- " + name] + lines[end + 1:]

def _remove(lines, path, name):
    found = _list(lines, path)
    if found == None or name not in found[3]:
        return lines
    start, _, _, items, flow = found
    if flow:
        g = re.groups(r"^(.*\[)(.*?)(\].*)$", lines[start])
        kept = [x for x in g[1].split(",") if x.strip() and x.strip().strip("\"'") != name]
        lines[start] = g[0] + ", ".join([x.strip() for x in kept]) + g[2]
        return lines
    i = items[name]
    if len(items) == 1:
        return lines[:start] + lines[i + 1:]  # the list is now empty: drop its key too
    return lines[:i] + lines[i + 1:]

def enable_linter(repo, name):
    """Enables a linter, removing it from `disable` if it was there.

    The linter is added to `linters.enable`, as a block or flow list
    (whichever the file uses), which is created if missing.

    Args:
        name: the linter, such as `gosec`.
    Returns:
        True when the file changed.
    Example:
        before .golangci.yml:
            linters:
              enable:
                - errcheck  # always
                - govet
              disable:
                - gosec
        call:
            enable_linter(repo, "gosec")
        after .golangci.yml:
            linters:
              enable:
                - errcheck  # always
                - govet
                - gosec
        returns:
            True
    """
    return _toggle(repo, "linters.enable", "linters.disable", name)

def disable_linter(repo, name):
    """Disables a linter, removing it from `enable` if it was there.

    Args:
        name: the linter, such as `lll`.
    Returns:
        True when the file changed.
    Example:
        before .golangci.yml:
            linters:
              enable: [errcheck, lll, govet]
        call:
            disable_linter(repo, "lll")
        after .golangci.yml:
            linters:
              enable: [errcheck, govet]
              disable: [lll]
        returns:
            True
    """
    return _toggle(repo, "linters.disable", "linters.enable", name)

def _toggle(repo, add_to, remove_from, name):
    # Moves a linter into one list and out of the other.
    path = find(repo)
    if path == None:
        return False
    content = repo.read(path)
    lines = _remove(content.split("\n"), remove_from, name)
    added = _add(lines, add_to, name)
    text = "\n".join(added if added != None else lines)
    if added == None:
        text = _set_in(text, add_to.split("."), [name])
    if text == content:
        return False
    repo.write(path, text)
    return True

def set_setting(repo, linter, key, value):
    """Sets a linter's setting, in `linters-settings` (v1) or `linters.settings` (v2).

    Args:
        linter: the linter, such as `govet`.
        key: the setting, dotted for nested keys, such as `enable-all`.
        value: a string, number, boolean or list.
    Returns:
        True when the file changed.
    Example:
        before .golangci.yml:
            version: "2"
            linters:
              settings:
                lll:
                  line-length: 120
        call:
            set_setting(repo, "lll", "line-length", 140)
        after .golangci.yml:
            version: "2"
            linters:
              settings:
                lll:
                  line-length: 140
        returns:
            True
    """
    path = find(repo)
    if path == None:
        return False
    prefix = ["linters", "settings"] if config_version(repo) == "2" else ["linters-settings"]
    return _set(repo, path, prefix + [linter] + key.split("."), value)
