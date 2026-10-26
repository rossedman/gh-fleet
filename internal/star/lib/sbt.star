"""Bump sbt, Scala, library dependencies and sbt plugins.

Edits are line-based, so comments and layout in build.sbt and the
project/ directory stay as they were.
"""

def _lit(s):
    return s.replace("$", "$$")

def build_files(repo):
    """Lists the sbt build files.

    Returns:
        The sorted paths of every *.sbt file, and every .scala and .sbt
        file under a project directory, outside target directories.
    Example:
        before build.sbt:
            name := "api"
        before project/plugins.sbt:
            addSbtPlugin("x" % "y" % "1")
        before project/Dependencies.scala:
            object Dependencies
        call:
            build_files(repo)
        returns:
            ["build.sbt", "project/Dependencies.scala", "project/plugins.sbt"]
    """
    return repo.glob("**/*.sbt", "**/project/*.scala", exclude = ["**/target/**"])

def set_sbt_version(repo, version):
    """Sets `sbt.version` in project/build.properties.

    A newer version already there is kept, so a campaign never lowers one.

    Args:
        version: the sbt version, such as `1.10.2`.
    Returns:
        True when the file changed.
    Example:
        before project/build.properties:
            sbt.version=1.9.7
        call:
            set_sbt_version(repo, "1.10.2")
        after project/build.properties:
            sbt.version=1.10.2
        returns:
            True
    """
    path = "project/build.properties"
    content = repo.read(path, default = None)
    if content == None:
        return False
    updated = re.sub(r"(?m)^(\s*sbt\.version\s*=\s*)(\S+)", lambda m: m[0] if semver.lowers(m[2], version) else m[1] + version, content)
    if updated == content:
        return False
    repo.write(path, updated)
    return True

def set_dependency(repo, organization, name, version):
    """Sets the version of a library or sbt plugin wherever the build declares it.

    A newer version already there is kept, so a campaign never lowers one.

    Covers `"org" % "name" % "v"` and `"org" %% "name" % "v"` (with any
    configuration such as `% Test` after it), and `addSbtPlugin(...)`.
    Versions held in a `val` are left alone; set the `val` with
    `set_val`.

    Args:
        organization: the group, such as `org.typelevel`.
        name: the artifact name, without a Scala suffix.
        version: the new version.
    Returns:
        The list of paths that changed.
    Example:
        before build.sbt:
            libraryDependencies ++= Seq(
              "org.typelevel" %% "cats-core" % "2.9.0",
              "org.scalatest" %% "scalatest" % "3.2.15" % Test
            )
        call:
            set_dependency(repo, "org.typelevel", "cats-core", "2.12.0")
        after build.sbt:
            libraryDependencies ++= Seq(
              "org.typelevel" %% "cats-core" % "2.12.0",
              "org.scalatest" %% "scalatest" % "3.2.15" % Test
            )
        returns:
            ["build.sbt"]
    """
    pattern = r"""("\s*""" + re.escape(organization) + r"""\s*"\s*%{1,3}\s*"\s*""" + re.escape(name) + r"""\s*"\s*%\s*")([^"]+)(")"""
    changed = []
    for path in build_files(repo):
        content = repo.read(path)
        updated = re.sub(pattern, lambda m: m[0] if semver.lowers(m[2], version) else m[1] + version + m[3], content)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed

def set_val(repo, name, value):
    """Sets a string `val`, such as `val catsVersion = "2.9.0"`, wherever it is defined.

    Args:
        name: the val's name.
        value: the new string value.
    Returns:
        The list of paths that changed.
    Example:
        before project/Dependencies.scala:
            object Dependencies {
              val catsVersion = "2.9.0"
            }
        call:
            set_val(repo, "catsVersion", "2.12.0")
        after project/Dependencies.scala:
            object Dependencies {
              val catsVersion = "2.12.0"
            }
        returns:
            ["project/Dependencies.scala"]
    """
    pattern = r"""(?m)^(\s*(?:lazy\s+)?val\s+""" + re.escape(name) + r"""\s*(?::\s*String\s*)?=\s*")([^"]*)(")"""
    changed = []
    for path in build_files(repo):
        content = repo.read(path)
        updated = re.sub(pattern, "${1}" + _lit(value) + "${3}", content)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed

def set_scala_version(repo, version, from_version = None):
    """Sets `scalaVersion`, and matching entries of `crossScalaVersions`.

    Args:
        version: the Scala version, such as `2.13.15`.
        from_version: when set, only this version is replaced, so a
            cross-built project keeps its other Scala versions.
    Returns:
        The list of paths that changed.
    Example:
        before build.sbt:
            ThisBuild / scalaVersion := "2.13.12"
            crossScalaVersions := Seq("2.13.12", "3.3.1")
        call:
            set_scala_version(repo, "2.13.15", from_version = "2.13.12")
        after build.sbt:
            ThisBuild / scalaVersion := "2.13.15"
            crossScalaVersions := Seq("2.13.15", "3.3.1")
        returns:
            ["build.sbt"]
    """
    changed = []
    for path in build_files(repo):
        content = repo.read(path)
        lines = content.split("\n")
        for i, line in enumerate(lines):
            if re.matches(r"\bscalaVersion\s*:=", line):
                g = re.groups(r'^(.*\bscalaVersion\s*:=\s*")([^"]*)(".*)$', line)
                if g and (from_version == None or g[1] == from_version):
                    lines[i] = g[0] + version + g[2]
            elif from_version != None and re.matches(r"\bcrossScalaVersions\s*:=", line):
                lines[i] = line.replace('"' + from_version + '"', '"' + version + '"')
        updated = "\n".join(lines)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed
