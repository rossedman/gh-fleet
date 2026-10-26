"""Read, set and delete keys in TOML files, such as pyproject.toml and Cargo.toml.

Edits are line-based, so comments, ordering and formatting stay as they
were. A key is addressed by its table, such as `tool.ruff` (`""` for the
top level), and its name. Array-of-tables (`[[...]]`) entries and dotted
keys can't be addressed.
"""

def _header(line):
    g = re.groups(r"^\s*\[([^\[\]]+)\]\s*(?:#.*)?$", line)
    if g:
        return ".".join([p.strip().strip("\"'") for p in g[0].split(".")])
    if re.matches(r"^\s*\[\[", line):
        return "[["  # an array of tables: not addressable
    return None

def _depth(text):
    # Net bracket depth of text, ignoring brackets inside strings.
    depth = 0
    quote = ""
    for c in text.elems():
        if quote:
            if c == quote:
                quote = ""
        elif c == '"' or c == "'":
            quote = c
        elif c == "#":
            break
        elif c in "[{":
            depth += 1
        elif c in "]}":
            depth -= 1
    return depth

def _split_comment(rest):
    quote = ""
    for i in range(len(rest)):
        c = rest[i]
        if quote:
            if c == quote:
                quote = ""
        elif c == '"' or c == "'":
            quote = c
        elif c == "#":
            value = rest[:i].rstrip()
            return value, rest[len(value):]
    return rest.rstrip(), ""

def _entries(lines):
    # Returns one dict per key: table, key, raw key, value text, comment,
    # first and last line, and indent.
    out = []
    table = ""
    i = 0
    for _ in range(len(lines)):
        if i >= len(lines):
            break
        line = lines[i]
        h = _header(line)
        if h != None:
            table = h
            out.append({"header": True, "table": table, "line": i})
            i += 1
            continue
        g = re.groups(r"""^(\s*)([A-Za-z0-9_-]+|"[^"]*"|'[^']*')\s*=\s*(.*)$""", line)
        if not g or table == "[[":
            i += 1
            continue
        value, comment = _split_comment(g[2])
        end = i
        depth = _depth(value)
        if value.startswith('"""') or value.startswith("'''"):
            q = value[:3]
            if value.count(q) < 2:
                for j in range(i + 1, len(lines)):
                    end = j
                    if q in lines[j]:
                        break
        else:
            for j in range(i + 1, len(lines)):
                if depth <= 0:
                    break
                depth += _depth(lines[j])
                end = j
        out.append({"header": False, "table": table, "key": g[1].strip("\"'"), "raw": g[1], "value": value, "comment": comment, "line": i, "end": end, "indent": g[0]})
        i = end + 1
    return out

def _decode(text):
    t = text.strip()
    if t.startswith('"') and t.endswith('"') and not t.startswith('"""'):
        return json.decode(t)
    if t.startswith("'") and t.endswith("'") and not t.startswith("'''"):
        return t[1:-1]
    if t in ("true", "false"):
        return t == "true"
    if re.matches(r"^[-+]?\d+$", t.replace("_", "")):
        return int(t.replace("_", ""))
    if re.matches(r"^[-+]?\d+\.\d+([eE][-+]?\d+)?$", t):
        return float(t)
    return text

def _scalar(value, old = ""):
    t = type(value)
    if t == "bool":
        return "true" if value else "false"
    if t == "int" or t == "float":
        return str(value)
    if t == "string":
        if old.startswith("'") and "'" not in value and "\n" not in value:
            return "'" + value + "'"
        return json.encode(value)
    fail("toml: can't write a %s here; lists and tables hold strings, numbers and booleans" % t)

def _format(value, old = ""):
    t = type(value)
    if t == "list" or t == "tuple":
        return "[" + ", ".join([_scalar(v) for v in value]) + "]"
    if t == "dict":
        return "{ " + ", ".join(["%s = %s" % (k, _scalar(v)) for k, v in value.items()]) + " }"
    return _scalar(value, old)

def get(repo, path, table, key):
    """Reads a key's value.

    Strings, integers, floats and booleans are decoded; arrays, inline
    tables and dates are returned as written.

    Args:
        path: the TOML file, relative to the repo root.
        table: the table name, such as `tool.ruff`, or `""` for the top level.
        key: the key name.
    Returns:
        The value, or None when the file, table or key is missing.
    Example:
        before pyproject.toml:
            [tool.ruff]
            line-length = 88  # black's default
            target-version = "py39"
        call:
            get(repo, "pyproject.toml", "tool.ruff", "line-length")
        returns:
            88
    """
    content = repo.read(path, default = None)
    if content == None:
        return None
    for e in _entries(content.split("\n")):
        if not e["header"] and e["table"] == table and e["key"] == key:
            lines = content.split("\n")[e["line"]:e["end"] + 1]
            if len(lines) > 1:
                return "\n".join([lines[0].split("=", 1)[1].strip()] + lines[1:])
            return _decode(e["value"])
    return None

def set(repo, path, table, key, value):
    """Sets a key, adding it, and its table, when missing.

    An existing value is replaced, keeping a trailing comment and the
    string quote style. A new key goes after the last key of its table; a
    new table goes at the end of the file.

    Args:
        path: the TOML file, relative to the repo root; created if needed.
        table: the table name, such as `tool.ruff`, or `""` for the top level.
        key: the key name.
        value: a string, number, boolean, or a list or dict of those.
    Returns:
        True when the file changed.
    Example:
        before pyproject.toml:
            [project]
            name = "api"

            [tool.ruff]
            line-length = 88  # black's default
        call:
            set(repo, "pyproject.toml", "tool.ruff", "line-length", 100)
            set(repo, "pyproject.toml", "tool.ruff.lint", "select", ["E", "F", "I"])
        after pyproject.toml:
            [project]
            name = "api"

            [tool.ruff]
            line-length = 100  # black's default

            [tool.ruff.lint]
            select = ["E", "F", "I"]
        returns:
            True
    """
    content = repo.read(path, default = "")
    lines = content.split("\n")
    if lines and lines[-1] == "":
        lines = lines[:-1]
    entries = _entries(lines)
    for e in entries:
        if not e["header"] and e["table"] == table and e["key"] == key:
            new = e["indent"] + e["raw"] + " = " + _format(value, e["value"])
            if e["line"] == e["end"]:
                new += e["comment"]
            lines = lines[:e["line"]] + [new] + lines[e["end"] + 1:]
            return _save(repo, path, content, lines)
    line = key + " = " + _format(value)
    in_table = [e for e in entries if e["table"] == table]
    keys = [e for e in in_table if not e["header"]]
    if keys:
        at = keys[-1]["end"] + 1
        lines = lines[:at] + [keys[-1]["indent"] + line] + lines[at:]
    elif in_table:
        at = in_table[0]["line"] + 1
        lines = lines[:at] + [line] + lines[at:]
    elif table == "":
        lines = [line] + ([""] if lines else []) + lines
    else:
        while lines and lines[-1].strip() == "":
            lines = lines[:-1]
        lines = lines + ([""] if lines else []) + ["[" + table + "]", line]
    return _save(repo, path, content, lines)

def _save(repo, path, content, lines):
    updated = "\n".join(lines) + "\n"
    if updated == content:
        return False
    repo.write(path, updated)
    return True

def delete(repo, path, table, key):
    """Removes a key from a table.

    Args:
        path: the TOML file, relative to the repo root.
        table: the table name, or `""` for the top level.
        key: the key name.
    Returns:
        True when the file changed.
    Example:
        before pyproject.toml:
            [tool.black]
            line-length = 100
            experimental-string-processing = true
        call:
            delete(repo, "pyproject.toml", "tool.black", "experimental-string-processing")
        after pyproject.toml:
            [tool.black]
            line-length = 100
        returns:
            True
    """
    content = repo.read(path, default = None)
    if content == None:
        return False
    lines = content.split("\n")
    for e in _entries(lines):
        if not e["header"] and e["table"] == table and e["key"] == key:
            repo.write(path, "\n".join(lines[:e["line"]] + lines[e["end"] + 1:]))
            return True
    return False
