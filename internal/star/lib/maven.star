"""Bump dependency, plugin and property versions in Maven pom.xml files.

Edits are line-based, so comments and formatting stay as they were. Each
`<groupId>`, `<artifactId>` and `<version>` is expected on its own line, as
Maven and most editors write them.
"""

def _lit(s):
    return s.replace("$", "$$")

def poms(repo):
    """Lists the repo's pom.xml files.

    Returns:
        The sorted paths of every pom.xml outside target directories.
    Example:
        before pom.xml:
            <project/>
        before core/pom.xml:
            <project/>
        before target/classes/pom.xml:
            <project/>
        call:
            poms(repo)
        returns:
            ["core/pom.xml", "pom.xml"]
    """
    return repo.glob("**/pom.xml", exclude = ["**/target/**"])

def _tag(line, tag):
    g = re.groups(r"<" + tag + r">\s*([^<]*?)\s*</" + tag + ">", line)
    return g[0] if g else None

def _set_property(lines, name, value, never_lower = False):
    inside = False
    changed = False
    for i, line in enumerate(lines):
        if "<properties>" in line:
            inside = True
        if inside and _tag(line, re.escape(name)) != None:
            new = re.sub(r"(<" + re.escape(name) + r">)([^<]*)(</" + re.escape(name) + ">)", lambda m: m[0] if never_lower and semver.lowers(m[2], value) else m[1] + value + m[3], line, count = 1)
            if new != line:
                lines[i] = new
                changed = True
        if "</properties>" in line:
            inside = False
    return changed

def set_property(repo, name, value):
    """Sets a `<properties>` entry, such as `java.version`, in every pom.xml that defines it.

    Args:
        name: the property name.
        value: the new value.
    Returns:
        The list of paths that changed.
    Example:
        before pom.xml:
            <project>
              <properties>
                <java.version>17</java.version>
                <jackson.version>2.15.2</jackson.version>
              </properties>
            </project>
        call:
            set_property(repo, "java.version", "21")
        after pom.xml:
            <project>
              <properties>
                <java.version>21</java.version>
                <jackson.version>2.15.2</jackson.version>
              </properties>
            </project>
        returns:
            ["pom.xml"]
    """
    changed = []
    for path in poms(repo):
        content = repo.read(path)
        lines = content.split("\n")
        if _set_property(lines, name, value):
            repo.write(path, "\n".join(lines))
            changed.append(path)
    return changed

def set_dependency(repo, group, artifact, version):
    """Sets the version of a dependency or plugin wherever a pom.xml declares one.

    A newer version already there is kept, so a campaign never lowers one.

    Covers `<dependency>` and `<plugin>` entries, including those under
    `<dependencyManagement>` and `<pluginManagement>`. When the version is
    a property reference such as `${jackson.version}`, the property is set
    instead. A plugin without a `<groupId>` matches the group
    `org.apache.maven.plugins`, as Maven assumes. Entries without a
    `<version>` are left alone.

    Args:
        group: the `groupId`.
        artifact: the `artifactId`.
        version: the new version.
    Returns:
        The list of paths that changed.
    Example:
        before pom.xml:
            <project>
              <properties>
                <jackson.version>2.15.2</jackson.version>
              </properties>
              <dependencies>
                <dependency>
                  <groupId>com.fasterxml.jackson.core</groupId>
                  <artifactId>jackson-databind</artifactId>
                  <version>${jackson.version}</version>
                </dependency>
                <dependency>
                  <groupId>org.slf4j</groupId>
                  <artifactId>slf4j-api</artifactId>
                  <version>2.0.7</version>
                </dependency>
              </dependencies>
            </project>
        call:
            set_dependency(repo, "com.fasterxml.jackson.core", "jackson-databind", "2.17.2")
            set_dependency(repo, "org.slf4j", "slf4j-api", "2.0.16")
        after pom.xml:
            <project>
              <properties>
                <jackson.version>2.17.2</jackson.version>
              </properties>
              <dependencies>
                <dependency>
                  <groupId>com.fasterxml.jackson.core</groupId>
                  <artifactId>jackson-databind</artifactId>
                  <version>${jackson.version}</version>
                </dependency>
                <dependency>
                  <groupId>org.slf4j</groupId>
                  <artifactId>slf4j-api</artifactId>
                  <version>2.0.16</version>
                </dependency>
              </dependencies>
            </project>
        returns:
            ["pom.xml"]
    """
    changed = []
    for path in poms(repo):
        content = repo.read(path)
        lines = content.split("\n")
        block = None
        props = []
        for i, line in enumerate(lines):
            if re.matches(r"<(dependency|plugin)>", line):
                kind = "plugin" if "<plugin>" in line else "dependency"
                block = {"kind": kind, "group": None, "artifact": None, "version": None}
            if block == None:
                continue
            for tag in ("groupId", "artifactId", "version"):
                v = _tag(line, tag)
                if v != None:
                    key = {"groupId": "group", "artifactId": "artifact", "version": "version"}[tag]
                    block[key] = (i, v) if tag == "version" else v
            if re.matches(r"</(dependency|plugin)>", line):
                g = block["group"] or ("org.apache.maven.plugins" if block["kind"] == "plugin" else None)
                if g == group and block["artifact"] == artifact and block["version"] != None:
                    at, current = block["version"]
                    prop = re.groups(r"^\$\{([^}]+)\}$", current)
                    if prop:
                        props.append(prop[0])
                    elif current != version and not semver.lowers(current, version):
                        lines[at] = re.sub(r"(<version>)[^<]*(</version>)", "${1}" + _lit(version) + "${2}", lines[at], count = 1)
                block = None
        for p in props:
            _set_property(lines, p, version, never_lower = True)
        updated = "\n".join(lines)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed
