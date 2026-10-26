"""Bump Gradle dependencies, version catalog entries and the Gradle wrapper.

Edits are line-based, so comments and formatting in build scripts and
libs.versions.toml stay as they were.
"""

WRAPPER = "gradle/wrapper/gradle-wrapper.properties"

def _lit(s):
    return s.replace("$", "$$")

def build_files(repo):
    """Lists the build scripts and version catalogs.

    Returns:
        The sorted paths of every build.gradle, build.gradle.kts,
        settings.gradle(.kts) and *.versions.toml outside build directories.
    Example:
        before build.gradle.kts:
            plugins { java }
        before app/build.gradle:
            apply plugin: 'java'
        before gradle/libs.versions.toml:
            [versions]
        before build/tmp/build.gradle:
            generated
        call:
            build_files(repo)
        returns:
            ["app/build.gradle", "build.gradle.kts", "gradle/libs.versions.toml"]
    """
    return repo.glob("**/*.gradle", "**/*.gradle.kts", "**/*.versions.toml", exclude = ["**/build/**", "**/.gradle/**"])

def set_dependency(repo, module, version):
    """Sets a library's version wherever the build declares it.

    Covers string coordinates in build scripts (`"group:name:1.0"`, in
    Groovy or Kotlin), and in version catalogs both `alias = "group:name:1.0"`
    and `{ module = "group:name", version = "1.0" }`, including a
    `version.ref` that points into `[versions]`. Coordinates whose version is
    a variable (`"group:name:$v"`) are left alone.

    Args:
        module: the library as `group:name`.
        version: the new version.
    Returns:
        The list of paths that changed.
    Example:
        before build.gradle.kts:
            dependencies {
                implementation("com.squareup.okhttp3:okhttp:4.11.0")
                testImplementation(libs.junit)
            }
        before gradle/libs.versions.toml:
            [versions]
            jackson = "2.15.2"

            [libraries]
            jackson-databind = { module = "com.fasterxml.jackson.core:jackson-databind", version.ref = "jackson" }
        call:
            set_dependency(repo, "com.squareup.okhttp3:okhttp", "4.12.0")
            set_dependency(repo, "com.fasterxml.jackson.core:jackson-databind", "2.17.2")
        after build.gradle.kts:
            dependencies {
                implementation("com.squareup.okhttp3:okhttp:4.12.0")
                testImplementation(libs.junit)
            }
        after gradle/libs.versions.toml:
            [versions]
            jackson = "2.17.2"

            [libraries]
            jackson-databind = { module = "com.fasterxml.jackson.core:jackson-databind", version.ref = "jackson" }
        returns:
            ["gradle/libs.versions.toml"]
    """
    coord = re.escape(module)
    group, _, name = module.partition(":")
    script = r"""(["'])(""" + coord + r""":)([^"'$:@]+)((?:@[^"']*)?)(["'])"""
    inline = r"""(module\s*=\s*["']""" + coord + r"""["'][^}]*\bversion\s*=\s*["'])([^"']+)(["'])"""
    split = r"""(group\s*=\s*["']""" + re.escape(group) + r"""["']\s*,\s*name\s*=\s*["']""" + re.escape(name) + r"""["'][^}]*\bversion\s*=\s*["'])([^"']+)(["'])"""
    ref = r"""(?:module\s*=\s*["']""" + coord + r"""["']|group\s*=\s*["']""" + re.escape(group) + r"""["']\s*,\s*name\s*=\s*["']""" + re.escape(name) + r"""["'])[^}]*\bversion\.ref\s*=\s*["']([^"']+)["']"""
    changed = []
    for path in build_files(repo):
        content = repo.read(path)
        updated = re.sub(script, "${1}${2}" + _lit(version) + "${4}${5}", content)
        if path.endswith(".toml"):
            for pattern in (inline, split):
                updated = re.sub(pattern, "${1}" + _lit(version) + "${3}", updated)
            for r in re.find_all(ref, updated):
                key = re.groups(ref, r)[0]
                updated = re.sub(r"""(?m)^(\s*""" + re.escape(key) + r"""\s*=\s*["'])[^"']*(["'])""", "${1}" + _lit(version) + "${2}", updated, count = 1)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed

def wrapper_version(repo):
    r"""Reads the Gradle version the wrapper downloads.

    Returns:
        The version, such as `8.5`, or None when the repo has no wrapper.
    Example:
        before gradle/wrapper/gradle-wrapper.properties:
            distributionUrl=https\://services.gradle.org/distributions/gradle-8.5-bin.zip
        call:
            wrapper_version(repo)
        returns:
            "8.5"
    """
    g = re.groups(r"(?m)^distributionUrl=.*gradle-([^-/]+(?:-rc-\d+|-milestone-\d+)?)-(?:bin|all)\.zip", repo.read(WRAPPER, default = ""))
    return g[0] if g else None

def set_wrapper_version(repo, version, sha256 = None):
    r"""Points the Gradle wrapper at a new version, never lowering it.

    Only gradle-wrapper.properties changes; Gradle downloads the new
    distribution on the next run. When the file pins a checksum with
    `distributionSha256Sum`, pass the new distribution's `sha256` from
    https://gradle.org/release-checksums/, or the run fails rather than
    leave a checksum that no longer matches.

    Args:
        version: the Gradle version, such as `8.10.2`.
        sha256: the SHA-256 of the new distribution zip, for wrappers that
            verify it, or a dict of it by distribution type (`bin`, `all`),
            so one campaign covers both.
    Returns:
        True when the file changed.
    Example:
        before gradle/wrapper/gradle-wrapper.properties:
            distributionBase=GRADLE_USER_HOME
            distributionUrl=https\://services.gradle.org/distributions/gradle-8.5-bin.zip
        call:
            set_wrapper_version(repo, "8.10.2")
        after gradle/wrapper/gradle-wrapper.properties:
            distributionBase=GRADLE_USER_HOME
            distributionUrl=https\://services.gradle.org/distributions/gradle-8.10.2-bin.zip
        returns:
            True
    """
    content = repo.read(WRAPPER, default = None)
    current = wrapper_version(repo)
    if content == None or current == None or current == version:
        return False
    if semver.valid(current) and semver.valid(version) and not semver.less(current, version):
        return False
    updated = re.sub(r"(?m)^(distributionUrl=.*gradle-)[^-/]+(?:-rc-\d+|-milestone-\d+)?(-(?:bin|all)\.zip)", "${1}" + _lit(version) + "${2}", content, count = 1)
    if re.matches(r"(?m)^distributionSha256Sum=", updated):
        if type(sha256) == "dict":
            kind = re.groups(r"(?m)^distributionUrl=.*-(bin|all)\.zip", updated)[0]
            sha256 = sha256.get(kind)
        if not sha256:
            fail("gradle.set_wrapper_version: %s pins distributionSha256Sum; pass sha256 for Gradle %s from https://gradle.org/release-checksums/" % (WRAPPER, version))
        updated = re.sub(r"(?m)^distributionSha256Sum=.*$", "distributionSha256Sum=" + _lit(sha256), updated)
    repo.write(WRAPPER, updated)
    return True
