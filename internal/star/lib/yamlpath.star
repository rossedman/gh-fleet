"""Read, set and delete keys in any YAML file by path, keeping comments and layout.

Keys are addressed with dots, such as `spec.replicas` or `jobs.test.runs-on`,
and `*` matches any one key, as in `jobs.*.runs-on`. Edits change only the
lines they must, so comments, quoting, blank lines and key order survive,
which a decode and re-encode would lose. Paths go through mappings only:
keys inside list items can't be addressed. Only the first document of a
multi-document file is read.
"""

_KEY = r"""^("(?:[^"\\]|\\.)*"|'[^']*'|[^\s#'"\-?:][^#]*?|-[^\s#][^#]*?)\s*:(?:\s+(.*?))?\s*$"""

def _unquote(s):
    if len(s) >= 2 and s[0] == '"' and s[-1] == '"':
        return json.decode(s)
    if len(s) >= 2 and s[0] == "'" and s[-1] == "'":
        return s[1:-1].replace("''", "'")
    return s

def _split_comment(rest):
    # Splits "value  # comment" into ("value", "  # comment"), ignoring # inside quotes.
    quote = ""
    for i in range(len(rest)):
        c = rest[i]
        if quote:
            if c == quote:
                quote = ""
        elif c == '"' or c == "'":
            quote = c
        elif c == "#" and (i == 0 or rest[i - 1] in " \t"):
            value = rest[:i].rstrip()
            return value, rest[len(value):]
    return rest, ""

def _entries(lines):
    # Returns one dict per mapping key: its line, indent, path, raw key,
    # value text, comment and the last line of its block.
    out = []
    stack = []  # (indent, key); key "-" marks a list item
    block_indent = -1
    seen = False
    for i, line in enumerate(lines):
        stripped = line.strip()
        if stripped == "---" or stripped == "...":
            if seen:
                break
            continue
        if not stripped or stripped.startswith("#"):
            continue
        indent = len(line) - len(line.lstrip(" "))
        if block_indent >= 0:
            if indent > block_indent:
                continue
            block_indent = -1
        seen = True
        for _ in range(len(stack)):
            if stack and stack[-1][0] >= indent:
                stack.pop()
        body = line.lstrip(" ")
        if body == "-" or body.startswith("- "):
            stack.append((indent, "-"))
            continue
        g = re.groups(_KEY, body)
        if not g:
            continue
        value, comment = _split_comment(g[1] or "")
        if g[1] and value == "" and comment == "":
            value = g[1]
        key = _unquote(g[0])
        path = [k for _, k in stack] + [key]
        out.append({"line": i, "indent": indent, "raw": g[0], "path": path, "value": value.strip(), "comment": comment, "end": i})
        stack.append((indent, key))
        if value.strip()[:1] in ("|", ">"):
            block_indent = indent
    for e in out:
        end = e["line"]
        for j in range(e["line"] + 1, len(lines)):
            s = lines[j].strip()
            if not s or s.startswith("#"):
                continue
            if s == "---":
                break
            if len(lines[j]) - len(lines[j].lstrip(" ")) <= e["indent"]:
                break
            end = j
        e["end"] = end
    return out

def _doc_end(lines):
    # The index of the line ending the first document: a later "---" or
    # "...", or the end of the file.
    seen = False
    for i, line in enumerate(lines):
        stripped = line.strip()
        if stripped in ("---", "..."):
            if seen:
                return i
        elif stripped and not stripped.startswith("#"):
            seen = True
    return len(lines)

def _match(path, pattern):
    if len(path) != len(pattern):
        return False
    for a, b in zip(path, pattern):
        if b != "*" and a != b:
            return False
    return True

def _scalar(e):
    v = e["value"]
    if v == "" or v[:1] in ("|", ">") or e["end"] > e["line"]:
        return None
    return _unquote(v)

def _format(value, old = ""):
    if type(value) in ("list", "tuple"):
        return "[" + ", ".join([_format_scalar(v) for v in value]) + "]"
    return _format_scalar(value, old)

def _format_scalar(value, old = ""):
    t = type(value)
    if value == None:
        return "null"
    if t == "bool":
        return "true" if value else "false"
    if t == "int" or t == "float":
        return str(value)
    if t != "string":
        fail("yamlpath: can't write a %s; set nested keys one at a time" % t)
    if old[:1] == "'" and "\n" not in value:
        return "'" + value.replace("'", "''") + "'"
    if old[:1] == '"':
        return json.encode(value)
    plain = re.matches(r"^[A-Za-z0-9_./@+][A-Za-z0-9_./@+ -]*$", value) and not value.endswith(" ")
    special = value.lower() in ("true", "false", "yes", "no", "on", "off", "null", "~", "y", "n") or re.matches(r"^[-+]?(\d+(\.\d*)?|\.\d+)([eE][-+]?\d+)?$", value)
    return value if plain and not special else json.encode(value)

def _load(repo, path):
    content = repo.read(path, default = None)
    if content == None:
        return None, None, None
    lines = content.split("\n")
    return content, lines, _entries(lines)

def get(repo, path, key):
    """Reads a scalar value by key path.

    Args:
        path: the YAML file, relative to the repo root.
        key: the dotted key path, such as `spec.replicas`.
    Returns:
        The value as a string, unquoted, or None when the file or key is
        missing or the key holds a mapping or block scalar.
    Example:
        before deploy.yaml:
            spec:
              replicas: 2  # scaled by HPA
              template:
                metadata:
                  labels:
                    app: "api"
        call:
            get(repo, "deploy.yaml", "spec.template.metadata.labels.app")
        returns:
            "api"
    """
    _, _, entries = _load(repo, path)
    for e in entries or []:
        if e["path"] == key.split("."):
            return _scalar(e)
    return None

def values(repo, path, key):
    """Reads every scalar whose key path matches a pattern with `*` wildcards.

    Args:
        path: the YAML file, relative to the repo root.
        key: the dotted key path, where `*` matches any one key, such as
            `jobs.*.runs-on`.
    Returns:
        A dict of each matching key path to its value, in file order.
    Example:
        before .github/workflows/ci.yml:
            jobs:
              test:
                runs-on: ubuntu-latest
              lint:
                runs-on: ubuntu-22.04
        call:
            values(repo, ".github/workflows/ci.yml", "jobs.*.runs-on")
        returns:
            {"jobs.test.runs-on": "ubuntu-latest", "jobs.lint.runs-on": "ubuntu-22.04"}
    """
    out = {}
    pattern = key.split(".")
    for e in _load(repo, path)[2] or []:
        if _match(e["path"], pattern):
            v = _scalar(e)
            if v != None:
                out[".".join(e["path"])] = v
    return out

def set(repo, path, key, value):
    """Sets a key to a value, adding the key and any missing parents if needed.

    An existing value is replaced on its own line, keeping its quote style
    and trailing comment. A missing key is added as the last child of its
    deepest existing parent, at the indentation the file already uses. With
    `*` in the path, every matching existing key is set and nothing is
    added. Strings are written plain when YAML would read them back as the
    same string, and double-quoted otherwise.

    Args:
        path: the YAML file, relative to the repo root; created if needed.
        key: the dotted key path, where `*` matches any one key.
        value: a string, number, boolean, None or list of those.
    Returns:
        True when the file changed.
    Example:
        before deploy.yaml:
            spec:
              replicas: 2  # scaled by HPA
              template:
                spec:
                  containers: []
        call:
            set(repo, "deploy.yaml", "spec.replicas", 3)
            set(repo, "deploy.yaml", "spec.revisionHistoryLimit", 5)
        after deploy.yaml:
            spec:
              replicas: 3  # scaled by HPA
              template:
                spec:
                  containers: []
              revisionHistoryLimit: 5
        returns:
            True
    """
    content, lines, entries = _load(repo, path)
    if content == None:
        content, lines, entries = "", [], []
    parts = key.split(".")
    found = [e for e in entries if _match(e["path"], parts)]
    if found:
        for e in reversed(found):
            if e["end"] > e["line"] or e["value"][:1] in ("|", ">"):
                fail("yamlpath.set: %s in %s holds a nested value" % (".".join(e["path"]), path))
            new = " " * e["indent"] + e["raw"] + ": " + _format(value, e["value"]) + e["comment"]
            lines[e["line"]] = new
    elif "*" in parts:
        return False
    else:
        parent = None
        depth = 0
        children = []
        for n in range(len(parts) - 1, 0, -1):
            hits = [e for e in entries if e["path"] == parts[:n]]
            if hits:
                parent, depth = hits[0], n
                break
        if parent != None and parent["value"] != "" and parent["end"] == parent["line"]:
            fail("yamlpath.set: %s in %s is a scalar, not a mapping" % (".".join(parts[:depth]), path))
        if parent == None:
            at = _doc_end(lines)
            while at > 0 and lines[at - 1].strip() == "":
                at -= 1
            indent = 0
        else:
            at = parent["end"] + 1
            children = [e for e in entries if len(e["path"]) == depth + 1 and e["path"][:depth] == parts[:depth]]
            indent = children[0]["indent"] if children else parent["indent"] + 2
        step = max(indent - parent["indent"], 2) if children else 2
        new = []
        for j, part in enumerate(parts[depth:]):
            pad = " " * (indent + step * j)
            last = j == len(parts) - depth - 1
            new.append(pad + _format(part) + ":" + (" " + _format(value) if last else ""))
        lines = lines[:at] + new + lines[at:]
    updated = "\n".join(lines)
    if not updated.endswith("\n"):
        updated += "\n"
    if updated == content:
        return False
    repo.write(path, updated)
    return True

def delete(repo, path, key):
    """Removes a key and everything nested under it.

    Args:
        path: the YAML file, relative to the repo root.
        key: the dotted key path, where `*` matches any one key.
    Returns:
        True when the file changed.
    Example:
        before .github/workflows/ci.yml:
            jobs:
              test:
                runs-on: ubuntu-latest
                continue-on-error: true
                steps: []
        call:
            delete(repo, ".github/workflows/ci.yml", "jobs.test.continue-on-error")
        after .github/workflows/ci.yml:
            jobs:
              test:
                runs-on: ubuntu-latest
                steps: []
        returns:
            True
    """
    content, lines, entries = _load(repo, path)
    if content == None:
        return False
    found = [e for e in entries if _match(e["path"], key.split("."))]
    if not found:
        return False
    for e in reversed(found):
        lines = lines[:e["line"]] + lines[e["end"] + 1:]
    repo.write(path, "\n".join(lines))
    return True
