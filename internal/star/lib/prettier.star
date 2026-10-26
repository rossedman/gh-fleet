"""Set Prettier options in whichever config a repo uses.

Supports .prettierrc (JSON or YAML), .prettierrc.json, .prettierrc.yaml,
.prettierrc.yml and the `prettier` key of package.json. JavaScript
configs (prettier.config.js and friends) can't be edited safely and are
not supported.
"""

load("./jsonfile.star", _jget = "get", _jset = "set")
load("./yamlpath.star", _yget = "get", _yset = "set")

FILES = [".prettierrc", ".prettierrc.json", ".prettierrc.yaml", ".prettierrc.yml"]

def find(repo):
    """Finds the repo's Prettier config.

    Returns:
        The path of the first of .prettierrc, .prettierrc.json,
        .prettierrc.yaml and .prettierrc.yml that exists, `package.json`
        when it has a `prettier` key, or None.
    Example:
        before package.json:
            {"name": "web", "prettier": {"semi": false}}
        call:
            find(repo)
        returns:
            "package.json"
    """
    for path in FILES:
        if repo.exists(path):
            return path
    if repo.exists("package.json") and type(_jget(repo, "package.json", "prettier")) == "dict":
        return "package.json"
    return None

def _is_json(repo, path):
    if path.endswith(".json"):
        return True
    if path.endswith(".yaml") or path.endswith(".yml"):
        return False
    return repo.read(path).lstrip().startswith("{")

def get(repo, option):
    """Reads an option from the Prettier config.

    Args:
        option: the option, such as `printWidth`.
    Returns:
        The value, or None when it isn't set or there is no config. YAML
        values come back as strings.
    Example:
        before .prettierrc:
            singleQuote: true
            printWidth: 100
        call:
            get(repo, "printWidth")
        returns:
            "100"
    """
    path = find(repo)
    if path == None:
        return None
    if path == "package.json":
        return _jget(repo, path, ["prettier", option])
    if _is_json(repo, path):
        return _jget(repo, path, [option])
    return _yget(repo, path, [option])

def set_option(repo, option, value):
    """Sets an option in the Prettier config, keeping its format.

    Args:
        option: the option, such as `printWidth` or `trailingComma`.
        value: a string, number or boolean.
    Returns:
        True when the file changed; False when the repo has no config.
    Example:
        before .prettierrc.json:
            {
              "semi": false,
              "trailingComma": "es5"
            }
        call:
            set_option(repo, "trailingComma", "all")
        after .prettierrc.json:
            {
              "semi": false,
              "trailingComma": "all"
            }
        returns:
            True
    """
    path = find(repo)
    if path == None:
        return False
    if path == "package.json":
        return _jset(repo, path, ["prettier", option], value)
    if _is_json(repo, path):
        return _jset(repo, path, [option], value)
    return _yset(repo, path, [option], value)
