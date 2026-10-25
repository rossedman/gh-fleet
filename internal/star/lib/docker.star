"""Find Dockerfiles and move images to a new tag in Dockerfiles, Compose files and Kubernetes manifests."""

def _lit(s):
    return s.replace("$", "$$")

def dockerfiles(repo):
    """Returns the paths of every Dockerfile, *.Dockerfile and Dockerfile.* outside node_modules."""
    return repo.glob(
        "**/Dockerfile",
        "**/Dockerfile.*",
        "**/*.Dockerfile",
        "**/*.dockerfile",
        exclude = ["**/node_modules/**"],
    )

def bump_base_image(repo, image, from_tag, to_tag):
    """Rewrites `FROM <image>:<from_tag>` to `<image>:<to_tag>` in every Dockerfile.

    A variant suffix is kept, so with `from_tag = "18"` and `to_tag = "22"`,
    `node:18-alpine` becomes `node:22-alpine`, while `node:18.2` is not
    touched. `--platform` flags and stage names are kept, and images pinned
    by digest only (`node@sha256:...`) never match.

    Returns the paths that changed.
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

def bump_image(repo, image, from_tag, to_tag, include = ["**/*.yml", "**/*.yaml"], exclude = ["**/node_modules/**"]):
    """Rewrites `image: <image>:<from_tag>` to `<image>:<to_tag>` in YAML files such as docker-compose.yml and Kubernetes manifests.

    Matching works like `bump_base_image`: a variant suffix is kept
    (`redis:7-alpine` becomes `redis:8-alpine`), and `redis:7.2` does not
    match `from_tag = "7"`. `image` is the full name as written, including
    any registry, such as `ghcr.io/acme/api`. Quotes are kept.

    Returns the paths that changed.
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
