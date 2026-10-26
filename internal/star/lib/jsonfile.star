"""Read and edit any JSON file, keeping key order and indentation.

For files such as renovate.json, tsconfig.json without comments,
.eslintrc.json and composer.json. Keys are addressed with dots, such as
`compilerOptions.strict`. Files with comments (JSONC) can't be decoded and
fail the run.
"""

def _indent(content):
    g = re.groups(r"\n([ \t]+)\"", content or "")
    return g[0] if g else "  "

def read(repo, path, default = None):
    """Reads a JSON file into a value whose dict keys keep the file's order.

    Args:
        path: the JSON file, relative to the repo root.
        default: what to return when the file does not exist; without it, a
            missing file fails the run.
    Returns:
        The decoded value.
    Example:
        before renovate.json:
            {"extends": ["config:base"], "schedule": ["weekly"]}
        call:
            read(repo, "renovate.json")["extends"]
        returns:
            ["config:base"]
    """
    content = repo.read(path, default = None)
    if content == None:
        if default == None:
            fail("jsonfile.read: %s does not exist" % path)
        return default
    return json.decode(content)

def write(repo, path, data):
    """Writes a value as JSON, using the indentation the file already has.

    A new file gets two spaces. Output ends with a newline.

    Args:
        path: the JSON file, relative to the repo root.
        data: the value to write, usually from `read`.
    Returns:
        True when the file changed.
    Example:
        before .eslintrc.json:
            {
                "root": true
            }
        call:
            config = read(repo, ".eslintrc.json")
            config["extends"] = ["eslint:recommended"]
            write(repo, ".eslintrc.json", config)
        after .eslintrc.json:
            {
                "root": true,
                "extends": [
                    "eslint:recommended"
                ]
            }
        returns:
            True
    """
    content = repo.read(path, default = None)
    updated = json.encode_indent(data, indent = _indent(content)) + "\n"
    if content != None and (content == updated or json.decode(content) == data and content.rstrip("\n") == updated.rstrip("\n")):
        return False
    repo.write(path, updated)
    return True

def get(repo, path, key, default = None):
    """Reads a value by dotted key path.

    Args:
        path: the JSON file, relative to the repo root.
        key: the dotted key path, such as `compilerOptions.strict`.
        default: what to return when the file or key is missing.
    Returns:
        The value, or `default`.
    Example:
        before tsconfig.json:
            {"compilerOptions": {"strict": false, "target": "ES2020"}}
        call:
            get(repo, "tsconfig.json", "compilerOptions.target")
        returns:
            "ES2020"
    """
    node = read(repo, path, default = {})
    for part in key.split("."):
        if type(node) != "dict" or part not in node:
            return default
        node = node[part]
    return node

def set(repo, path, key, value):
    """Sets a value by dotted key path, adding missing objects on the way.

    The file is created if needed. Existing keys keep their place; new keys
    go at the end of their object.

    Args:
        path: the JSON file, relative to the repo root.
        key: the dotted key path, such as `compilerOptions.strict`.
        value: any JSON value: string, number, boolean, None, list or dict.
    Returns:
        True when the file changed.
    Example:
        before tsconfig.json:
            {
              "compilerOptions": {
                "strict": false
              }
            }
        call:
            set(repo, "tsconfig.json", "compilerOptions.strict", True)
        after tsconfig.json:
            {
              "compilerOptions": {
                "strict": true
              }
            }
        returns:
            True
    """
    data = read(repo, path, default = {})
    parts = key.split(".")
    node = data
    for part in parts[:-1]:
        child = node.get(part)
        if child == None:
            child = {}
            node[part] = child
        elif type(child) != "dict":
            fail("jsonfile.set: %s in %s is not an object" % (part, path))
        node = child
    if parts[-1] in node and node[parts[-1]] == value and repo.exists(path):
        return False
    node[parts[-1]] = value
    return write(repo, path, data)

def delete(repo, path, key):
    """Removes a key by dotted key path.

    Args:
        path: the JSON file, relative to the repo root.
        key: the dotted key path.
    Returns:
        True when the file changed.
    Example:
        before package.json:
            {
              "name": "web",
              "scripts": {
                "test": "jest",
                "postinstall": "husky install"
              }
            }
        call:
            delete(repo, "package.json", "scripts.postinstall")
        after package.json:
            {
              "name": "web",
              "scripts": {
                "test": "jest"
              }
            }
        returns:
            True
    """
    if not repo.exists(path):
        return False
    data = read(repo, path)
    parts = key.split(".")
    node = data
    for part in parts[:-1]:
        node = node.get(part) if type(node) == "dict" else None
        if node == None:
            return False
    if type(node) != "dict" or parts[-1] not in node:
        return False
    node.pop(parts[-1])
    return write(repo, path, data)
