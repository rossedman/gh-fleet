"""Bump Ansible role and collection versions in requirements files.

Edits are line-based, so comments and layout stay as they were.
"""

load("./yamlpath.star", _set_sibling = "set_sibling")

def requirements(repo):
    """Lists the repo's Ansible Galaxy requirements files.

    Returns:
        The sorted paths of every requirements.yml or requirements.yaml,
        including those under roles/ and collections/.
    Example:
        before requirements.yml:
            collections: []
        before roles/requirements.yml:
            roles: []
        call:
            requirements(repo)
        returns:
            ["requirements.yml", "roles/requirements.yml"]
    """
    return repo.glob("**/requirements.yml", "**/requirements.yaml", exclude = ["**/node_modules/**", "**/.venv/**"])

def set_version(repo, name, version):
    """Sets the version of a role or collection wherever a requirements file lists it.

    A newer version already there is kept, so a campaign never lowers one.

    Entries are matched by `name`, and for roles also by `src`. A missing
    `version` key is added.

    Args:
        name: the collection or role name, such as `community.general`.
        version: the new version or range, such as `9.4.0` or `>=9.0.0`.
    Returns:
        The list of paths that changed.
    Example:
        before requirements.yml:
            collections:
              - name: community.general
                version: 7.0.0
              - name: ansible.posix
                version: 1.5.4
        call:
            set_version(repo, "community.general", "9.4.0")
        after requirements.yml:
            collections:
              - name: community.general
                version: 9.4.0
              - name: ansible.posix
                version: 1.5.4
        returns:
            ["requirements.yml"]
    """
    changed = []
    for path in requirements(repo):
        a = _set_sibling(repo, path, "name", name, "version", version, never_lower = True)
        b = _set_sibling(repo, path, "src", name, "version", version, never_lower = True)
        if a or b:
            changed.append(path)
    return changed
