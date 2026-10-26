"""Edit Renovate configs: presets, package rules, schedules and other settings.

Works on JSON configs (renovate.json, .github/renovate.json, .renovaterc
and .renovaterc.json), keeping key order and indentation. JSON5 configs
can't be edited this way and are skipped by `find`.
"""

load("./jsonfile.star", _read = "read", _set = "set")

FILES = ["renovate.json", ".github/renovate.json", ".gitlab/renovate.json", ".renovaterc", ".renovaterc.json"]

def find(repo):
    """Finds the repo's Renovate config.

    Returns:
        The path of the first JSON config that exists, or None.
    Example:
        before .github/renovate.json:
            {}
        call:
            find(repo)
        returns:
            ".github/renovate.json"
    """
    for path in FILES:
        if repo.exists(path):
            return path
    return None

def add_preset(repo, preset):
    """Adds a preset to `extends` unless it is already there.

    Args:
        preset: the preset, such as `config:recommended` or
            `github>acme/renovate-config`.
    Returns:
        True when the file changed.
    Example:
        before renovate.json:
            {
              "extends": ["config:recommended"]
            }
        call:
            add_preset(repo, "github>acme/renovate-config")
        after renovate.json:
            {
              "extends": ["config:recommended", "github>acme/renovate-config"]
            }
        returns:
            True
    """
    path = find(repo)
    if path == None:
        return False
    extends = list(_read(repo, path).get("extends") or [])
    if preset in extends:
        return False
    return _set(repo, path, "extends", extends + [preset])

def remove_preset(repo, preset):
    """Removes a preset from `extends`, such as a retired `config:base`.

    Args:
        preset: the preset to remove.
    Returns:
        True when the file changed.
    Example:
        before renovate.json:
            {"extends": ["config:base", ":semanticCommits"]}
        call:
            remove_preset(repo, "config:base")
        after renovate.json:
            {"extends": [":semanticCommits"]}
        returns:
            True
    """
    path = find(repo)
    if path == None:
        return False
    extends = list(_read(repo, path).get("extends") or [])
    if preset not in extends:
        return False
    return _set(repo, path, "extends", [p for p in extends if p != preset])

def replace_preset(repo, old, new):
    """Replaces a preset in `extends` where it is, such as `config:base` with `config:recommended`.

    When `new` is already there too, `old` is just removed.

    Args:
        old: the preset to replace.
        new: the preset to put in its place.
    Returns:
        True when the file changed.
    Example:
        before renovate.json:
            {
              "extends": ["config:base", ":semanticCommits"]
            }
        call:
            replace_preset(repo, "config:base", "config:recommended")
        after renovate.json:
            {
              "extends": ["config:recommended", ":semanticCommits"]
            }
        returns:
            True
    """
    path = find(repo)
    if path == None:
        return False
    extends = list(_read(repo, path).get("extends") or [])
    if old not in extends:
        return False
    out = []
    for p in extends:
        p = new if p == old else p
        if p not in out:
            out.append(p)
    return _set(repo, path, "extends", out)

def add_package_rule(repo, rule):
    """Appends a rule to `packageRules` unless an identical rule is already there.

    Args:
        rule: the rule, a dict such as
            `{"matchUpdateTypes": ["patch"], "automerge": True}`.
    Returns:
        True when the file changed.
    Example:
        before renovate.json:
            {
              "extends": ["config:recommended"]
            }
        call:
            add_package_rule(repo, {"matchDepTypes": ["devDependencies"], "automerge": True})
        after renovate.json:
            {
              "extends": ["config:recommended"],
              "packageRules": [
                {
                  "matchDepTypes": ["devDependencies"],
                  "automerge": true
                }
              ]
            }
        returns:
            True
    """
    path = find(repo)
    if path == None:
        return False
    rules = list(_read(repo, path).get("packageRules") or [])
    if rule in rules:
        return False
    return _set(repo, path, "packageRules", rules + [rule])

def set_option(repo, key, value):
    """Sets a top-level or nested option, such as `schedule` or `lockFileMaintenance.enabled`.

    Args:
        key: the dotted option path.
        value: any JSON value.
    Returns:
        True when the file changed; False when the repo has no config.
    Example:
        before renovate.json:
            {
              "extends": ["config:recommended"],
              "schedule": ["at any time"]
            }
        call:
            set_option(repo, "schedule", ["before 6am on monday"])
        after renovate.json:
            {
              "extends": ["config:recommended"],
              "schedule": ["before 6am on monday"]
            }
        returns:
            True
    """
    path = find(repo)
    if path == None:
        return False
    return _set(repo, path, key, value)
