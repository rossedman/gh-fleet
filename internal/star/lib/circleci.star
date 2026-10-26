"""Edit CircleCI configuration: orb versions, executor images, machine images and resource classes.

Edits are line-based, so comments and anchors stay as they were.
"""

load("./docker.star", _bump_image = "bump_image")
load("./refs.star", _bump = "bump")

CONFIGS = [".circleci/*.yml", ".circleci/*.yaml", ".circleci/**/*.yml", ".circleci/**/*.yaml"]

def configs(repo):
    """Lists the repo's CircleCI configs.

    Returns:
        The sorted paths of every YAML file under .circleci, including
        dynamic and continuation configs.
    Example:
        before .circleci/config.yml:
            version: 2.1
        before .circleci/continue/deploy.yml:
            version: 2.1
        call:
            configs(repo)
        returns:
            [".circleci/config.yml", ".circleci/continue/deploy.yml"]
    """
    return sorted(set(repo.glob(*CONFIGS)))

def orbs(repo):
    """Lists the orbs configs use, with their versions.

    Returns:
        The sorted, de-duplicated `namespace/orb@version` strings.
    Example:
        before .circleci/config.yml:
            version: 2.1
            orbs:
              node: circleci/node@5.1.0
              aws-cli: "circleci/aws-cli@4.0"
        call:
            orbs(repo)
        returns:
            ["circleci/aws-cli@4.0", "circleci/node@5.1.0"]
    """
    found = set()
    for path in configs(repo):
        inside = -1
        for line in repo.read(path).split("\n"):
            indent = len(line) - len(line.lstrip(" "))
            if re.matches(r"^\s*orbs:\s*(#.*)?$", line):
                inside = indent
                continue
            if inside >= 0 and line.strip() and indent <= inside:
                inside = -1
            g = re.groups(r"""^\s*[\w-]+:\s*["']?([\w-]+/[\w-]+)@([^"'\s#]+)""", line)
            if inside >= 0 and g:
                found.add(g[0] + "@" + g[1])
    return sorted(found)

def bump_orb(repo, orb, version, from_ref = None):
    """Moves an orb to a new version in every config.

    Args:
        orb: the orb as `namespace/name`, such as `circleci/node`; `*`
            matches any run of name characters, as in `acme/*`.
        version: the new version, such as `5.2.0`.
        from_ref: when set, only this exact version is replaced.
    Returns:
        The list of paths that changed.
    Example:
        before .circleci/config.yml:
            orbs:
              node: circleci/node@5.1.0  # pinned
              slack: circleci/slack@4.12.0
        call:
            bump_orb(repo, "circleci/node", "5.2.0")
        after .circleci/config.yml:
            orbs:
              node: circleci/node@5.2.0  # pinned
              slack: circleci/slack@4.12.0
        returns:
            [".circleci/config.yml"]
    """
    return _bump(repo, CONFIGS, orb, version, sep = "@", from_ref = from_ref)

def bump_image(repo, image, from_tag, to_tag):
    """Moves a Docker executor image from one tag to another.

    A variant suffix is kept, so `cimg/node:18.20-browsers` becomes
    `cimg/node:22.11-browsers`.

    Args:
        image: the image name, such as `cimg/node`.
        from_tag: the tag to replace, such as `18.20`.
        to_tag: the tag to put in its place.
    Returns:
        The list of paths that changed.
    Example:
        before .circleci/config.yml:
            jobs:
              test:
                docker:
                  - image: cimg/node:18.20-browsers
                  - image: cimg/postgres:15.4
        call:
            bump_image(repo, "cimg/node", "18.20", "22.11")
        after .circleci/config.yml:
            jobs:
              test:
                docker:
                  - image: cimg/node:22.11-browsers
                  - image: cimg/postgres:15.4
        returns:
            [".circleci/config.yml"]
    """
    return _bump_image(repo, image, from_tag, to_tag, include = CONFIGS)

def _glob(pattern, s):
    # Whether s matches pattern, where * matches any run of characters.
    return re.matches("^" + ".*".join([re.escape(p) for p in pattern.split("*")]) + "$", s)

def _set_value(repo, key_line, value, from_value, scope = None):
    # Sets `key: value` lines (optionally only under a `scope:` key) in
    # every config, keeping quotes and comments.
    changed = []
    for path in configs(repo):
        content = repo.read(path)
        lines = content.split("\n")
        inside = -1
        for i, line in enumerate(lines):
            indent = len(line) - len(line.lstrip(" "))
            if scope != None:
                if re.matches(r"^\s*-?\s*" + scope + r":\s*(#.*)?$", line):
                    inside = indent
                    continue
                if inside >= 0 and line.strip() and indent <= inside:
                    inside = -1
                if inside < 0:
                    continue
            g = re.groups(r"""^(\s*""" + key_line + r""":\s*["']?)([^"'\s#]+)(.*)$""", line)
            if g and g[1] != value and (from_value == None or _glob(from_value, g[1])):
                lines[i] = g[0] + value + g[2]
        updated = "\n".join(lines)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed

def set_resource_class(repo, resource_class, from_class = None):
    """Sets `resource_class` on every job and executor that sets one.

    Args:
        resource_class: the new class, such as `large` or `arm.medium`.
        from_class: when set, only this class is replaced; `*` matches any
            run of characters.
    Returns:
        The list of paths that changed.
    Example:
        before .circleci/config.yml:
            jobs:
              build:
                resource_class: medium
              e2e:
                resource_class: xlarge
        call:
            set_resource_class(repo, "large", from_class = "medium")
        after .circleci/config.yml:
            jobs:
              build:
                resource_class: large
              e2e:
                resource_class: xlarge
        returns:
            [".circleci/config.yml"]
    """
    return _set_value(repo, "resource_class", resource_class, from_class)

def set_machine_image(repo, image, from_image = None):
    """Sets the `image` of every machine executor, such as when an Ubuntu image is retired.

    Only `image` keys under a `machine:` key change; Docker executor images
    are left alone.

    Args:
        image: the new machine image, such as `ubuntu-2404:2024.08.1`.
        from_image: when set, only this image is replaced; `*` matches any
            run of characters, so `ubuntu-2004:*` covers every tag.
    Returns:
        The list of paths that changed.
    Example:
        before .circleci/config.yml:
            jobs:
              integration:
                machine:
                  image: ubuntu-2004:202201-02
                docker_layer_caching: true
              unit:
                docker:
                  - image: cimg/base:2024.01
        call:
            set_machine_image(repo, "ubuntu-2404:2024.08.1", from_image = "ubuntu-2004:*")
        after .circleci/config.yml:
            jobs:
              integration:
                machine:
                  image: ubuntu-2404:2024.08.1
                docker_layer_caching: true
              unit:
                docker:
                  - image: cimg/base:2024.01
        returns:
            [".circleci/config.yml"]
    """
    return _set_value(repo, "image", image, from_image, scope = "machine")
