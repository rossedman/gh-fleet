"""Bump CMake's minimum version and FetchContent dependency tags.

Edits are line-based, so comments and formatting in CMakeLists.txt stay
as they were.
"""

def _lit(s):
    return s.replace("$", "$$")

def files(repo):
    """Lists the repo's CMake files.

    Returns:
        The sorted paths of every CMakeLists.txt and *.cmake file outside
        build directories.
    Example:
        before CMakeLists.txt:
            project(app)
        before cmake/deps.cmake:
            include(FetchContent)
        before build/CMakeLists.txt:
            generated
        call:
            files(repo)
        returns:
            ["CMakeLists.txt", "cmake/deps.cmake"]
    """
    return repo.glob("**/CMakeLists.txt", "**/*.cmake", exclude = ["**/build/**", "**/_deps/**", "**/cmake-build-*/**"])

def minimum_version(repo, path = "CMakeLists.txt"):
    """Reads the version from `cmake_minimum_required`.

    Args:
        path: the CMake file to read.
    Returns:
        The minimum version, such as `3.16`, or None.
    Example:
        before CMakeLists.txt:
            cmake_minimum_required(VERSION 3.16...3.28)
        call:
            minimum_version(repo)
        returns:
            "3.16"
    """
    g = re.groups(r"(?i)cmake_minimum_required\s*\(\s*VERSION\s+(\d+(?:\.\d+)*)", repo.read(path, default = ""))
    return g[0] if g else None

def set_minimum_version(repo, version):
    """Raises `cmake_minimum_required` to a version, never lowering it.

    A `...max` policy range, such as `3.16...3.28`, keeps its upper end.

    Args:
        version: the new minimum, such as `3.20`.
    Returns:
        The list of paths that changed.
    Example:
        before CMakeLists.txt:
            cmake_minimum_required(VERSION 3.10)
            project(app)
        call:
            set_minimum_version(repo, "3.20")
        after CMakeLists.txt:
            cmake_minimum_required(VERSION 3.20)
            project(app)
        returns:
            ["CMakeLists.txt"]
    """
    changed = []
    for path in files(repo):
        content = repo.read(path)
        current = minimum_version(repo, path)
        if current == None or not semver.less(current, version):
            continue
        updated = re.sub(r"(?i)(cmake_minimum_required\s*\(\s*VERSION\s+)\d+(?:\.\d+)*", "${1}" + _lit(version), content, count = 1)
        repo.write(path, updated)
        changed.append(path)
    return changed

def set_fetchcontent_tag(repo, name, tag):
    """Sets the `GIT_TAG` of a `FetchContent_Declare` dependency.

    A newer version already there is kept, so a campaign never lowers one.

    The declaration may span several lines. Declarations of other names,
    and ones without a `GIT_TAG`, are left alone.

    Args:
        name: the dependency's name, the first argument of
            `FetchContent_Declare`.
        tag: the new tag or commit.
    Returns:
        The list of paths that changed.
    Example:
        before CMakeLists.txt:
            FetchContent_Declare(
              googletest
              GIT_REPOSITORY https://github.com/google/googletest.git
              GIT_TAG        v1.13.0  # pinned
            )
        call:
            set_fetchcontent_tag(repo, "googletest", "v1.15.2")
        after CMakeLists.txt:
            FetchContent_Declare(
              googletest
              GIT_REPOSITORY https://github.com/google/googletest.git
              GIT_TAG        v1.15.2  # pinned
            )
        returns:
            ["CMakeLists.txt"]
    """
    changed = []
    for path in files(repo):
        content = repo.read(path)
        lines = content.split("\n")
        inside = False
        for i, line in enumerate(lines):
            if re.matches(r"(?i)FetchContent_Declare\s*\(\s*" + re.escape(name) + r"(\s|\)|$)", line):
                inside = True
            elif re.matches(r"(?i)FetchContent_Declare\s*\(\s*$", line) and i + 1 < len(lines) and lines[i + 1].strip().split(" ")[0] == name:
                inside = True
            if inside:
                g = re.groups(r"^(.*\bGIT_TAG\s+)([^\s)]+)(.*)$", line)
                if g and g[1] != tag and not semver.lowers(g[1], tag):
                    lines[i] = g[0] + tag + g[2]
                if ")" in line:
                    inside = False
        updated = "\n".join(lines)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed
