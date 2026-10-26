"""Bump Bazel itself, Bzlmod module dependencies and http_archive downloads.

Edits are line-based, so comments and formatting in MODULE.bazel,
WORKSPACE and .bzl files stay as they were. MODULE.bazel.lock isn't
updated; run `bazel mod deps --lockfile_mode=update` in a `run` step when
it is committed.
"""

def _lit(s):
    return s.replace("$", "$$")

def version(repo):
    """Reads the Bazel version pinned in .bazelversion.

    Returns:
        The version, or None when the repo has no .bazelversion.
    Example:
        before .bazelversion:
            7.1.1
        call:
            version(repo)
        returns:
            "7.1.1"
    """
    v = repo.read(".bazelversion", default = "").strip()
    return v.splitlines()[0].strip() if v else None

def set_version(repo, version):
    """Sets the Bazel version in .bazelversion, when the repo pins one.

    A newer version already there is kept, so a campaign never lowers one.

    Args:
        version: the Bazel version, such as `7.4.1`.
    Returns:
        True when the file changed.
    Example:
        before .bazelversion:
            6.4.0
        call:
            set_version(repo, "7.4.1")
        after .bazelversion:
            7.4.1
        returns:
            True
    """
    content = repo.read(".bazelversion", default = None)
    if content == None or content.strip() == version or semver.lowers(content.strip(), version):
        return False
    repo.write(".bazelversion", version + "\n")
    return True

def _module_files(repo):
    return repo.glob("**/MODULE.bazel", "**/*.MODULE.bazel", exclude = ["**/bazel-*/**"])

def set_dep(repo, name, version):
    """Sets the version of a `bazel_dep` in every MODULE.bazel that declares it.

    A newer version already there is kept, so a campaign never lowers one.

    The call may span lines and its arguments may come in any order.

    Args:
        name: the module name, such as `rules_go`.
        version: the new version.
    Returns:
        The list of paths that changed.
    Example:
        before MODULE.bazel:
            module(name = "app")

            bazel_dep(name = "rules_go", version = "0.46.0")
            bazel_dep(
                name = "gazelle",
                version = "0.35.0",
            )
        call:
            set_dep(repo, "gazelle", "0.39.1")
            set_dep(repo, "rules_go", "0.50.1")
        after MODULE.bazel:
            module(name = "app")

            bazel_dep(name = "rules_go", version = "0.50.1")
            bazel_dep(
                name = "gazelle",
                version = "0.39.1",
            )
        returns:
            ["MODULE.bazel"]
    """
    changed = []
    for path in _module_files(repo):
        content = repo.read(path)
        lines = content.split("\n")
        start = -1
        for i, line in enumerate(lines):
            if start < 0 and re.matches(r"\bbazel_dep\s*\(", line):
                start = i
            if start < 0:
                continue
            call = "\n".join(lines[start:i + 1])
            if ")" not in line:
                continue
            if re.matches(r"""\bname\s*=\s*["']""" + re.escape(name) + r"""["']""", call):
                for j in range(start, i + 1):
                    g = re.groups(r"""^(.*\bversion\s*=\s*["'])([^"']*)(["'].*)$""", lines[j])
                    if g and g[1] != version and not semver.lowers(g[1], version):
                        lines[j] = g[0] + version + g[2]
            start = -1
        updated = "\n".join(lines)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed

def set_archive(repo, name, urls, sha256, strip_prefix = None):
    """Points an `http_archive` at a new download, with its checksum.

    Sets `urls` (or `url`), `sha256` and, when given, `strip_prefix` of the
    `http_archive` with this name in WORKSPACE files and .bzl files. Get
    the checksum from the release page, or from `sha256sum` on the
    download; the run fails rather than leave an archive without one.

    Args:
        name: the repository name, as in `http_archive(name = ...)`.
        urls: the new download URL, or a list of mirrors.
        sha256: the SHA-256 of the download.
        strip_prefix: the new `strip_prefix`, or None to leave it.
    Returns:
        The list of paths that changed.
    Example:
        before WORKSPACE:
            http_archive(
                name = "rules_python",
                sha256 = "aaaa",
                strip_prefix = "rules_python-0.27.0",
                url = "https://github.com/bazelbuild/rules_python/releases/download/0.27.0/rules_python-0.27.0.tar.gz",
            )
        call:
            set_archive(repo, "rules_python", "https://github.com/bazelbuild/rules_python/releases/download/0.36.0/rules_python-0.36.0.tar.gz", "bbbb", strip_prefix = "rules_python-0.36.0")
        after WORKSPACE:
            http_archive(
                name = "rules_python",
                sha256 = "bbbb",
                strip_prefix = "rules_python-0.36.0",
                url = "https://github.com/bazelbuild/rules_python/releases/download/0.36.0/rules_python-0.36.0.tar.gz",
            )
        returns:
            ["WORKSPACE"]
    """
    if not re.matches(r"^[0-9a-fA-F]+$", sha256):
        fail("bazel.set_archive: sha256 %r is not a hex digest" % sha256)
    if type(urls) == "string":
        urls = [urls]
    changed = []
    for path in repo.glob("**/WORKSPACE", "**/WORKSPACE.bazel", "**/*.bzl", exclude = ["**/bazel-*/**"]):
        content = repo.read(path)
        lines = content.split("\n")
        start = -1
        for i, line in enumerate(lines):
            if start < 0 and re.matches(r"\bhttp_archive\s*\(", line):
                start = i
            if start < 0 or not re.matches(r"^\s*\)", line) and not (")" in line and start == i):
                continue
            call = "\n".join(lines[start:i + 1])
            if re.matches(r"""\bname\s*=\s*["']""" + re.escape(name) + r"""["']""", call):
                for j in range(start, i + 1):
                    l = lines[j]
                    l = re.sub(r"""(\bsha256\s*=\s*["'])[^"']*(["'])""", "${1}" + sha256 + "${2}", l)
                    if strip_prefix != None:
                        l = re.sub(r"""(\bstrip_prefix\s*=\s*["'])[^"']*(["'])""", "${1}" + _lit(strip_prefix) + "${2}", l)
                    l = re.sub(r"""(\burl\s*=\s*["'])[^"']*(["'])""", "${1}" + _lit(urls[0]) + "${2}", l)
                    l = re.sub(r"""(\burls\s*=\s*)\[[^\]]*\]""", "${1}" + _lit(json.encode(urls)), l)
                    lines[j] = l
            start = -1
        updated = "\n".join(lines)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed
