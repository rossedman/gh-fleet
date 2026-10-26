"""Read and set properties in .editorconfig, line by line.

Sections are named as in the file, such as `*` or `*.{js,ts}`; use `""`
for the preamble before the first section, where `root = true` goes.
"""

PATH = ".editorconfig"

def _sections(lines):
    # Returns (name, first line, last line) for the preamble and each section.
    out = []
    name, start = "", 0
    for i, line in enumerate(lines):
        g = re.groups(r"^\s*\[(.+)\]\s*([#;].*)?$", line)
        if g:
            out.append((name, start, i - 1))
            name, start = g[0], i
    out.append((name, start, len(lines) - 1))
    return out

def get(repo, section, key):
    """Reads a property.

    Args:
        section: the section name, such as `*` or `*.go`, or `""` for the
            preamble.
        key: the property name.
    Returns:
        The value as written, or None.
    Example:
        before .editorconfig:
            root = true

            [*]
            indent_style = space
            indent_size = 2

            [*.go]
            indent_style = tab
        call:
            get(repo, "*.go", "indent_style")
        returns:
            "tab"
    """
    lines = repo.read(PATH, default = "").split("\n")
    for name, start, end in _sections(lines):
        if name != section:
            continue
        for line in lines[start:end + 1]:
            g = re.groups(r"^\s*([^=#;\[\s][^=]*?)\s*=\s*(.*?)\s*$", line)
            if g and g[0].lower() == key.lower():
                return g[1]
    return None

def set(repo, section, key, value):
    """Sets a property, adding it, or its section, when missing.

    A new property goes after the section's last property; a new section
    goes at the end of the file. The file is created if needed.

    Args:
        section: the section name, or `""` for the preamble.
        key: the property name.
        value: the value, such as `lf` or `4`; booleans become `true` and
            `false`.
    Returns:
        True when the file changed.
    Example:
        before .editorconfig:
            root = true

            [*]
            indent_style = space
        call:
            set(repo, "*", "end_of_line", "lf")
            set(repo, "Makefile", "indent_style", "tab")
        after .editorconfig:
            root = true

            [*]
            indent_style = space
            end_of_line = lf

            [Makefile]
            indent_style = tab
        returns:
            True
    """
    if type(value) == "bool":
        value = "true" if value else "false"
    value = str(value)
    content = repo.read(PATH, default = "")
    lines = content.split("\n") if content else []
    if lines and lines[-1] == "":
        lines = lines[:-1]
    for name, start, end in _sections(lines):
        if name != section:
            continue
        last = start if section else -1
        for i in range(start, end + 1):
            g = re.groups(r"^(\s*)([^=#;\[\s][^=]*?)(\s*=\s*)(.*?)(\s*)$", lines[i])
            if not g:
                continue
            last = i
            if g[1].lower() == key.lower():
                if g[3] == value:
                    return False
                lines[i] = g[0] + g[1] + g[2] + value + g[4]
                return _save(repo, content, lines)
        if section == "" and last < 0:
            lines = [key + " = " + value] + ([""] if lines else []) + lines
        else:
            lines = lines[:last + 1] + [key + " = " + value] + lines[last + 1:]
        return _save(repo, content, lines)
    while lines and lines[-1].strip() == "":
        lines = lines[:-1]
    lines = lines + ([""] if lines else []) + ["[" + section + "]", key + " = " + value]
    return _save(repo, content, lines)

def _save(repo, content, lines):
    updated = "\n".join(lines) + "\n"
    if updated == content:
        return False
    repo.write(PATH, updated)
    return True
