"""Edit Dev Container configs: images, feature versions and top-level settings.

devcontainer.json is JSON with comments, so edits are line-based and keep
comments, trailing commas and layout. Reading strips the comments first.
"""

FILES = [".devcontainer/devcontainer.json", ".devcontainer.json", ".devcontainer/*/devcontainer.json"]

def _lit(s):
    return s.replace("$", "$$")

def files(repo):
    """Lists the repo's Dev Container configs.

    Returns:
        The sorted paths of .devcontainer/devcontainer.json,
        .devcontainer.json and .devcontainer/<name>/devcontainer.json.
    Example:
        before .devcontainer/devcontainer.json:
            {}
        before .devcontainer/python/devcontainer.json:
            {}
        call:
            files(repo)
        returns:
            [".devcontainer/devcontainer.json", ".devcontainer/python/devcontainer.json"]
    """
    return repo.glob(*FILES)

def _strip(text):
    # Removes // and /* */ comments and trailing commas outside strings.
    out = []
    i = 0
    n = len(text)
    in_str = False
    for _ in range(n):
        if i >= n:
            break
        c = text[i]
        if in_str:
            out.append(c)
            if c == "\\" and i + 1 < n:
                out.append(text[i + 1])
                i += 2
                continue
            if c == '"':
                in_str = False
            i += 1
            continue
        if c == '"':
            in_str = True
            out.append(c)
            i += 1
        elif text[i:i + 2] == "//":
            j = text.find("\n", i)
            i = n if j < 0 else j
        elif text[i:i + 2] == "/*":
            j = text.find("*/", i + 2)
            i = n if j < 0 else j + 2
        else:
            out.append(c)
            i += 1
    return re.sub(r",(\s*[}\]])", "${1}", "".join(out))

def read(repo, path = ".devcontainer/devcontainer.json"):
    """Reads a Dev Container config, ignoring comments and trailing commas.

    Args:
        path: the config file.
    Returns:
        The decoded config, or None when the file does not exist.
    Example:
        before .devcontainer/devcontainer.json:
            {
              // The Go image
              "image": "mcr.microsoft.com/devcontainers/go:1.22",
              "features": {
                "ghcr.io/devcontainers/features/node:1": {},
              },
            }
        call:
            read(repo)["image"]
        returns:
            "mcr.microsoft.com/devcontainers/go:1.22"
    """
    content = repo.read(path, default = None)
    if content == None:
        return None
    return json.decode(_strip(content))

def bump_image(repo, image, from_tag, to_tag):
    """Moves the config's `image` from one tag to another.

    A variant suffix is kept, so `go:1.22-bookworm` becomes
    `go:1.23-bookworm`.

    Args:
        image: the image name, such as `mcr.microsoft.com/devcontainers/go`.
        from_tag: the tag to replace.
        to_tag: the tag to put in its place.
    Returns:
        The list of paths that changed.
    Example:
        before .devcontainer/devcontainer.json:
            {
              "name": "api",
              "image": "mcr.microsoft.com/devcontainers/go:1.22-bookworm", // base
            }
        call:
            bump_image(repo, "mcr.microsoft.com/devcontainers/go", "1.22", "1.23")
        after .devcontainer/devcontainer.json:
            {
              "name": "api",
              "image": "mcr.microsoft.com/devcontainers/go:1.23-bookworm", // base
            }
        returns:
            [".devcontainer/devcontainer.json"]
    """
    pattern = r'"' + re.escape(image) + ":" + re.escape(from_tag) + r'((?:-[^"]*)?")'
    changed = []
    for path in files(repo):
        content = repo.read(path)
        updated = re.sub(pattern, '"' + _lit(image) + ":" + _lit(to_tag) + "${1}", content)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed

def features(repo, path = ".devcontainer/devcontainer.json"):
    """Lists the features a config installs, with their versions.

    Args:
        path: the config file.
    Returns:
        A dict of feature ID, without its version, to the version (the part
        after the last `:`), in file order.
    Example:
        before .devcontainer/devcontainer.json:
            {
              "features": {
                "ghcr.io/devcontainers/features/node:1": {"version": "22"},
                "ghcr.io/devcontainers/features/docker-in-docker:2": {}
              }
            }
        call:
            features(repo)
        returns:
            {"ghcr.io/devcontainers/features/node": "1", "ghcr.io/devcontainers/features/docker-in-docker": "2"}
    """
    config = read(repo, path) or {}
    out = {}
    for key in (config.get("features") or {}).keys():
        name, _, version = key.rpartition(":")
        if not name or "/" in version:
            name, version = key, ""
        out[name] = version
    return out

def set_feature_version(repo, feature, version):
    """Moves a feature to a new major version, such as `docker-in-docker:1` to `:2`.

    A newer version already there is kept, so a campaign never lowers one.

    Args:
        feature: the feature ID without its version.
        version: the new version, as written after `:`.
    Returns:
        The list of paths that changed.
    Example:
        before .devcontainer/devcontainer.json:
            {
              "features": {
                "ghcr.io/devcontainers/features/docker-in-docker:1": {} // dind
              }
            }
        call:
            set_feature_version(repo, "ghcr.io/devcontainers/features/docker-in-docker", "2")
        after .devcontainer/devcontainer.json:
            {
              "features": {
                "ghcr.io/devcontainers/features/docker-in-docker:2": {} // dind
              }
            }
        returns:
            [".devcontainer/devcontainer.json"]
    """
    pattern = r'("' + re.escape(feature) + r':)([^"/]+)("\s*:)'
    changed = []
    for path in files(repo):
        content = repo.read(path)
        updated = re.sub(pattern, lambda m: m[0] if semver.lowers(m[2], version) else m[1] + version + m[3], content)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed

def _indent_of(line):
    return line[:len(line) - len(line.lstrip(" \t"))]

def add_feature(repo, feature, options = {}, path = ".devcontainer/devcontainer.json"):
    """Adds a feature to a config that doesn't have it yet, in any version.

    The feature goes first in `features`, which is created if missing.

    Args:
        feature: the feature ID with its version, such as
            `ghcr.io/devcontainers/features/github-cli:1`.
        options: the feature's options.
        path: the config file.
    Returns:
        True when the file changed.
    Example:
        before .devcontainer/devcontainer.json:
            {
              "image": "mcr.microsoft.com/devcontainers/base:ubuntu",
              "features": {
                "ghcr.io/devcontainers/features/node:1": {}
              }
            }
        call:
            add_feature(repo, "ghcr.io/devcontainers/features/github-cli:1")
        after .devcontainer/devcontainer.json:
            {
              "image": "mcr.microsoft.com/devcontainers/base:ubuntu",
              "features": {
                "ghcr.io/devcontainers/features/github-cli:1": {},
                "ghcr.io/devcontainers/features/node:1": {}
              }
            }
        returns:
            True
    """
    config = read(repo, path)
    if config == None:
        return False
    name = feature.rpartition(":")[0] or feature
    if name in features(repo, path) or feature in (config.get("features") or {}):
        return False
    content = repo.read(path)
    lines = content.split("\n")
    entry = json.encode(feature) + ": " + json.encode(options)
    for i, line in enumerate(lines):
        g = re.groups(r'^(\s*"features"\s*:\s*\{)\s*(\}?)(.*)$', line)
        if not g:
            continue
        if g[1] == "}":
            lines[i] = g[0] + " " + entry + " }" + g[2]
        else:
            inner = _indent_of(lines[i + 1]) if i + 1 < len(lines) and lines[i + 1].strip() not in ("", "}") else _indent_of(line) + "  "
            nonempty = i + 1 < len(lines) and not lines[i + 1].strip().startswith("}")
            lines = lines[:i + 1] + [inner + entry + ("," if nonempty else "")] + lines[i + 1:]
        repo.write(path, "\n".join(lines))
        return True
    return _add_top_level(repo, path, lines, '"features": { ' + entry + " }")

def _add_top_level(repo, path, lines, member):
    # Adds a member right after the root object's opening brace.
    for i, line in enumerate(lines):
        if line.strip().startswith("{"):
            rest = line.strip()[1:].strip()
            nxt = [l for l in lines[i + 1:] if l.strip()]
            indent = _indent_of(nxt[0]) if nxt and not nxt[0].strip().startswith("}") else "  "
            empty = rest.startswith("}") or (not rest and nxt and nxt[0].strip().startswith("}"))
            if rest.startswith("}"):
                lines[i] = line.replace("{", "{\n" + indent + member + "\n", 1)
            elif rest:
                # Members follow on the same line: insert before them.
                lines[i] = line.replace("{", "{ " + member + ",", 1)
            else:
                lines = lines[:i + 1] + [indent + member + ("" if empty else ",")] + lines[i + 1:]
            repo.write(path, "\n".join(lines))
            return True
    return False

def set(repo, key, value, path = ".devcontainer/devcontainer.json"):
    """Sets a top-level setting, such as `postCreateCommand` or `remoteUser`.

    An existing single-line value is replaced, keeping its trailing comma
    and comment; a missing key is added at the top. Values spanning
    several lines can't be replaced and fail the run.

    Args:
        key: the top-level key.
        value: a string, number, boolean, list or dict, written on one line.
        path: the config file.
    Returns:
        True when the file changed.
    Example:
        before .devcontainer/devcontainer.json:
            {
              "image": "mcr.microsoft.com/devcontainers/go:1.23",
              "postCreateCommand": "go mod download" // cache modules
            }
        call:
            set(repo, "postCreateCommand", "make setup")
            set(repo, "remoteUser", "vscode")
        after .devcontainer/devcontainer.json:
            {
              "remoteUser": "vscode",
              "image": "mcr.microsoft.com/devcontainers/go:1.23",
              "postCreateCommand": "make setup" // cache modules
            }
        returns:
            True
    """
    config = read(repo, path)
    if config == None:
        return False
    if key in config and config[key] == value:
        return False
    content = repo.read(path)
    lines = content.split("\n")
    encoded = json.encode(value)
    if key in config:
        for i, line in enumerate(lines):
            g = re.groups(r'^(\s*"' + re.escape(key) + r'"\s*:\s*)(.*?)(\s*,?\s*(?://.*)?)$', line)
            if not g:
                continue
            old = g[1]
            if old.endswith(","):
                old = old[:-1]
            try_value = _strip(old)
            if not try_value.strip() or try_value.strip()[-1:] in ("{", "["):
                fail("devcontainer.set: %s in %s spans several lines; edit it by hand" % (key, path))
            lines[i] = g[0] + encoded + g[2]
            repo.write(path, "\n".join(lines))
            return True
        return False
    return _add_top_level(repo, path, lines, json.encode(key) + ": " + encoded)
