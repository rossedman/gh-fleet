"""Set rules in markdownlint, yamllint and hadolint configs.

Each helper edits the config file the tool reads by default, keeping
comments in YAML files and key order in JSON files.
"""

load("./jsonfile.star", _jset = "set")
load("./yamlpath.star", _yset = "set")

def _first(repo, paths):
    for path in paths:
        if repo.exists(path):
            return path
    return None

def markdownlint_rule(repo, rule, value):
    """Sets a markdownlint rule, such as `MD013` (line length).

    Edits .markdownlint.json, .markdownlint.jsonc (without comments),
    .markdownlint.yaml or .markdownlint.yml.

    Args:
        rule: the rule ID or alias, such as `MD013` or `line-length`.
        value: True or False to turn the rule on or off, or a dict of its
            options.
    Returns:
        True when the file changed; False when the repo has no config.
    Example:
        before .markdownlint.json:
            {
              "default": true
            }
        call:
            markdownlint_rule(repo, "MD013", False)
        after .markdownlint.json:
            {
              "default": true,
              "MD013": false
            }
        returns:
            True
    """
    path = _first(repo, [".markdownlint.json", ".markdownlint.jsonc", ".markdownlint.yaml", ".markdownlint.yml"])
    if path == None:
        return False
    if path.endswith("json") or path.endswith("jsonc"):
        return _jset(repo, path, [rule], value)
    if type(value) == "dict":
        fail("lintconfig.markdownlint_rule: rule options in YAML configs aren't supported; set them by key")
    return _yset(repo, path, [rule], value)

def yamllint_rule(repo, rule, value):
    """Sets a yamllint rule, such as turning off `document-start`.

    Edits .yamllint, .yamllint.yaml or .yamllint.yml.

    Args:
        rule: the rule, such as `line-length`.
        value: `disable` or `enable`, or an option of the rule as
            `{"max": 120}`, set key by key.
    Returns:
        True when the file changed; False when the repo has no config.
    Example:
        before .yamllint.yml:
            extends: default
            rules:
              line-length:
                max: 80  # strict
        call:
            yamllint_rule(repo, "line-length", {"max": 120})
            yamllint_rule(repo, "document-start", "disable")
        after .yamllint.yml:
            extends: default
            rules:
              line-length:
                max: 120  # strict
              document-start: disable
        returns:
            True
    """
    path = _first(repo, [".yamllint", ".yamllint.yaml", ".yamllint.yml"])
    if path == None:
        return False
    if type(value) == "dict":
        changed = False
        for k, v in value.items():
            if _yset(repo, path, ["rules", rule, k], v):
                changed = True
        return changed
    return _yset(repo, path, ["rules", rule], value)

def hadolint_ignore(repo, rules):
    """Adds rules to the `ignored` list of .hadolint.yaml, creating the file if needed.

    Args:
        rules: the rule IDs, such as `["DL3008", "DL3013"]`.
    Returns:
        True when the file changed.
    Example:
        before .hadolint.yaml:
            ignored:
              - DL3008  # apt pins
        call:
            hadolint_ignore(repo, ["DL3013", "DL3008"])
        after .hadolint.yaml:
            ignored:
              - DL3008  # apt pins
              - DL3013
        returns:
            True
    """
    path = _first(repo, [".hadolint.yaml", ".hadolint.yml"]) or ".hadolint.yaml"
    content = repo.read(path, default = "")
    have = [m.strip().lstrip("-").strip().split(" ")[0].strip("\"'") for m in re.find_all(r"(?m)^\s*-\s*\S+", content)]
    add = [r for r in rules if r not in have]
    if not add:
        return False
    lines = content.split("\n") if content else []
    if lines and lines[-1] == "":
        lines = lines[:-1]
    at = -1
    indent = "  "
    for i, line in enumerate(lines):
        if re.matches(r"^ignored:\s*(#.*)?$", line):
            at = i
            for j in range(i + 1, len(lines)):
                if re.matches(r"^\s*-", lines[j]):
                    at = j
                    indent = lines[j][:len(lines[j]) - len(lines[j].lstrip(" "))]
                elif lines[j].strip():
                    break
            break
    if at < 0:
        lines = lines + ["ignored:"]
        at = len(lines) - 1
    lines = lines[:at + 1] + [indent + "- " + r for r in add] + lines[at + 1:]
    repo.write(path, "\n".join(lines) + "\n")
    return True
