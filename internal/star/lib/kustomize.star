"""Set image tags and names in kustomization files, the usual way to roll out a release with Kustomize.

Edits are line-based, so comments and layout stay as they were.
"""

load("./yamlpath.star", _set_sibling = "set_sibling")

def files(repo):
    """Lists the repo's kustomization files.

    Returns:
        The sorted paths of every kustomization.yaml, kustomization.yml
        and Kustomization file.
    Example:
        before deploy/base/kustomization.yaml:
            resources: []
        before deploy/prod/kustomization.yml:
            resources: []
        call:
            files(repo)
        returns:
            ["deploy/base/kustomization.yaml", "deploy/prod/kustomization.yml"]
    """
    return repo.glob("**/kustomization.yaml", "**/kustomization.yml", "**/Kustomization")

def set_image(repo, name, new_tag = None, new_name = None, digest = None):
    """Sets `newTag`, `newName` or `digest` for an image in every kustomization's `images` list.

    Only kustomizations that already list the image change; an entry is
    found by its `name`. A missing `newTag` or `newName` key is added to the
    entry.

    Args:
        name: the image name as the manifests use it, such as `ghcr.io/acme/api`.
        new_tag: the tag to set, or None to leave it.
        new_name: the replacement image name to set, or None to leave it.
        digest: the digest to set, such as `sha256:...`, or None to leave it.
    Returns:
        The list of paths that changed.
    Example:
        before deploy/prod/kustomization.yaml:
            resources:
              - ../base
            images:
              - name: ghcr.io/acme/api
                newTag: "2.3.1"  # release
              - name: redis
                newTag: "7"
        call:
            set_image(repo, "ghcr.io/acme/api", new_tag = "2.4.0")
        after deploy/prod/kustomization.yaml:
            resources:
              - ../base
            images:
              - name: ghcr.io/acme/api
                newTag: "2.4.0"  # release
              - name: redis
                newTag: "7"
        returns:
            ["deploy/prod/kustomization.yaml"]
    """
    changed = []
    for path in files(repo):
        dirty = False
        for key, value in (("newName", new_name), ("newTag", new_tag), ("digest", digest)):
            if value != None and _set_sibling(repo, path, "name", name, key, value):
                dirty = True
        if dirty:
            changed.append(path)
    return changed
