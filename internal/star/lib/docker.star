"""Find Dockerfiles, replace base images, and move images to a new tag in Dockerfiles, Compose files and Kubernetes manifests."""

def _lit(s):
    return s.replace("$", "$$")

def dockerfiles(repo):
    """Lists the repo's Dockerfiles.

    Returns:
        The sorted paths of every `Dockerfile`, `Dockerfile.*`, `*.Dockerfile`
        and `*.dockerfile` outside node_modules.
    Example:
        before Dockerfile:
            FROM scratch
        before web/app.Dockerfile:
            FROM node:22
        before node_modules/x/Dockerfile:
            FROM node:18
        call:
            dockerfiles(repo)
        returns:
            ["Dockerfile", "web/app.Dockerfile"]
    """
    return repo.glob(
        "**/Dockerfile",
        "**/Dockerfile.*",
        "**/*.Dockerfile",
        "**/*.dockerfile",
        exclude = ["**/node_modules/**"],
    )

def bump_base_image(repo, image, from_tag, to_tag):
    """Moves `FROM` lines of an image from one tag to another in every Dockerfile.

    A variant suffix is kept, so with tags `18` and `22`, `node:18-alpine`
    becomes `node:22-alpine`, while `node:18.2` is not touched. `--platform`
    flags and stage names are kept, and images pinned only by digest
    (`node@sha256:...`) never match.

    Args:
        image: the image name as written, including any registry.
        from_tag: the tag to replace.
        to_tag: the tag to put in its place.
    Returns:
        The list of paths that changed.
    Example:
        before Dockerfile:
            FROM node:18-alpine AS build
            RUN npm ci
            FROM node:18.2
        call:
            bump_base_image(repo, "node", "18", "22")
        after Dockerfile:
            FROM node:22-alpine AS build
            RUN npm ci
            FROM node:18.2
        returns:
            ["Dockerfile"]
    """
    pattern = r"(?im)^(\s*FROM\s+(?:--platform=\S+\s+)?)" + re.escape(image) + ":" + re.escape(from_tag) + r"([-@\s]|$)"
    repl = "${1}" + _lit(image) + ":" + _lit(to_tag) + "${2}"
    changed = []
    for path in dockerfiles(repo):
        content = repo.read(path)
        updated = re.sub(pattern, repl, content)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed

def replace_base_image(repo, from_image, to_image):
    """Replaces a whole image reference in the `FROM` lines of every Dockerfile.

    Use it to move to another image entirely, such as from Docker Hub to
    your own registry, where `bump_base_image` only changes the tag.

    `from_image` is matched as written. A bare name, such as `python` or
    `ghcr.io/acme/base`, matches that image at any tag or digest, so
    `python:3.12-slim` and `python@sha256:...` are both replaced; it doesn't
    match `python-dev` or `python/tools`. A name with a tag or digest, such
    as `python:3.12-slim`, matches only that exact reference. The whole
    reference is replaced by `to_image`, and `--platform` flags and stage
    names are kept.

    Args:
        from_image: the image to replace: a name, or a name with a tag or digest.
        to_image: the full reference to put in its place, such as `registry.acme.io/python:3.14-slim`.
    Returns:
        The list of paths that changed.
    Example:
        before Dockerfile:
            FROM --platform=linux/amd64 python:3.12-slim AS build
            RUN pip install -r requirements.txt
            FROM python:3.12-slim
            FROM python-dev:1
        call:
            replace_base_image(repo, "python", "registry.acme.io/library/python/3.14-slim:main-latest")
        after Dockerfile:
            FROM --platform=linux/amd64 registry.acme.io/library/python/3.14-slim:main-latest AS build
            RUN pip install -r requirements.txt
            FROM registry.acme.io/library/python/3.14-slim:main-latest
            FROM python-dev:1
        returns:
            ["Dockerfile"]
    """
    name = from_image[from_image.rfind("/") + 1:]
    ref = re.escape(from_image)
    if ":" not in name and "@" not in from_image:
        # Any tag, then any digest, or neither.
        ref += r"(?::[^\s@]+)?(?:@\S+)?"
    pattern = r"(?im)^(\s*FROM\s+(?:--platform=\S+\s+)?)" + ref + r"(\s|$)"
    repl = "${1}" + _lit(to_image) + "${2}"
    changed = []
    for path in dockerfiles(repo):
        content = repo.read(path)
        updated = re.sub(pattern, repl, content)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed

def bump_image(repo, image, from_tag, to_tag, include = ["**/*.yml", "**/*.yaml"], exclude = ["**/node_modules/**"]):
    """Moves `image:` lines from one tag to another in YAML files.

    Covers docker-compose.yml, Kubernetes manifests and anything else that
    writes `image: name:tag`. Tags match like `bump_base_image`: a variant
    suffix is kept, and `redis:7.2` does not match tag `7`. Quotes are kept.

    Args:
        image: the full image name as written, such as `ghcr.io/acme/api`.
        from_tag: the tag to replace.
        to_tag: the tag to put in its place.
        include: glob patterns of the files to edit.
        exclude: glob patterns of files to skip.
    Returns:
        The list of paths that changed.
    Example:
        before compose.yaml:
            services:
              cache:
                image: redis:7-alpine
              pinned:
                image: redis:7.2.4
        call:
            bump_image(repo, "redis", "7", "8")
        after compose.yaml:
            services:
              cache:
                image: redis:8-alpine
              pinned:
                image: redis:7.2.4
        returns:
            ["compose.yaml"]
    """
    pattern = r"""(?m)^(\s*-?\s*image:\s*["']?)""" + re.escape(image) + ":" + re.escape(from_tag) + r"""([-@"'\s]|$)"""
    repl = "${1}" + _lit(image) + ":" + _lit(to_tag) + "${2}"
    changed = []
    for path in repo.glob(exclude = exclude, *include):
        content = repo.read(path)
        updated = re.sub(pattern, repl, content)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed
