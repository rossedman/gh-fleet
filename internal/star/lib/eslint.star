"""Set rules and presets in ESLint's JSON and YAML configs.

Supports the legacy (eslintrc) format: .eslintrc.json, .eslintrc,
.eslintrc.yaml, .eslintrc.yml and the `eslintConfig` key of package.json.
Flat configs (eslint.config.js) are JavaScript and can't be edited
safely; `find` reports them so a campaign can skip or flag those repos.
"""

load("./jsonfile.star", _jget = "get", _jset = "set")
load("./yamlpath.star", _yget = "get", _yset = "set")

FILES = [".eslintrc.json", ".eslintrc", ".eslintrc.yaml", ".eslintrc.yml"]
FLAT = ["eslint.config.js", "eslint.config.mjs", "eslint.config.cjs", "eslint.config.ts"]

def find(repo):
    """Finds the repo's ESLint config.

    Returns:
        The path of the first JSON or YAML config that exists, `package.json`
        when it has an `eslintConfig` key, the path of a flat config when the
        repo uses one, or None.
    Example:
        before eslint.config.mjs:
            export default []
        call:
            find(repo)
        returns:
            "eslint.config.mjs"
    """
    for path in FILES:
        if repo.exists(path):
            return path
    if repo.exists("package.json") and type(_jget(repo, "package.json", "eslintConfig")) == "dict":
        return "package.json"
    for path in FLAT:
        if repo.exists(path):
            return path
    return None

def _editable(repo):
    path = find(repo)
    if path == None or path in FLAT:
        return None, None
    if path == "package.json":
        return path, "json"
    if path.endswith(".json") or (path == ".eslintrc" and repo.read(path).lstrip().startswith("{")):
        return path, "json"
    return path, "yaml"

def set_rule(repo, rule, level, options = None):
    """Sets a rule's level, and optionally its options.

    Args:
        rule: the rule, such as `no-console` or `@typescript-eslint/no-unused-vars`.
        level: `off`, `warn` or `error` (or 0, 1, 2).
        options: the rule's options, written after the level as in
            `["error", {"allow": ["warn"]}]`; JSON configs only.
    Returns:
        True when the file changed; False when the repo has no editable
        config.
    Example:
        before .eslintrc.json:
            {
              "extends": ["eslint:recommended"],
              "rules": {
                "no-console": "warn"
              }
            }
        call:
            set_rule(repo, "no-console", "error", {"allow": ["warn", "error"]})
        after .eslintrc.json:
            {
              "extends": [
                "eslint:recommended"
              ],
              "rules": {
                "no-console": [
                  "error",
                  {
                    "allow": [
                      "warn",
                      "error"
                    ]
                  }
                ]
              }
            }
        returns:
            True
    """
    path, kind = _editable(repo)
    if path == None:
        return False
    keys = (["eslintConfig"] if path == "package.json" else []) + ["rules", rule]
    value = level if options == None else [level, options]
    if kind == "json":
        return _jset(repo, path, keys, value)
    if options != None:
        fail("eslint.set_rule: rule options need a JSON config; %s is YAML" % path)
    return _yset(repo, path, keys, level)

def add_extends(repo, preset):
    """Adds a preset to `extends` unless it is already there.

    Args:
        preset: the preset, such as `plugin:security/recommended`.
    Returns:
        True when the file changed; False when the repo has no editable
        JSON config.
    Example:
        before .eslintrc.json:
            {"extends": "eslint:recommended"}
        call:
            add_extends(repo, "plugin:security/recommended")
        after .eslintrc.json:
            {
              "extends": [
                "eslint:recommended",
                "plugin:security/recommended"
              ]
            }
        returns:
            True
    """
    path, kind = _editable(repo)
    if path == None or kind != "json":
        return False
    keys = (["eslintConfig"] if path == "package.json" else []) + ["extends"]
    current = _jget(repo, path, keys) or []
    if type(current) == "string":
        current = [current]
    if preset in current:
        return False
    return _jset(repo, path, keys, list(current) + [preset])
