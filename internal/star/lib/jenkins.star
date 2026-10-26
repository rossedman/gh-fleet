"""Edit Jenkinsfiles: shared library versions, agent images and tool versions.

Jenkinsfiles are Groovy, which can't be parsed safely here, so edits are
line-based matches on the common declarative and scripted forms.
"""

load("./refs.star", _bump = "bump")

FILES = ["Jenkinsfile", "**/Jenkinsfile", "**/Jenkinsfile.*", "**/*.jenkinsfile", "**/*.Jenkinsfile"]

def _lit(s):
    return s.replace("$", "$$")

def jenkinsfiles(repo):
    """Lists the repo's Jenkinsfiles.

    Returns:
        The sorted paths of every Jenkinsfile, Jenkinsfile.*, *.jenkinsfile
        and *.Jenkinsfile.
    Example:
        before Jenkinsfile:
            pipeline {}
        before ci/release.Jenkinsfile:
            pipeline {}
        before Jenkinsfile.nightly:
            pipeline {}
        call:
            jenkinsfiles(repo)
        returns:
            ["Jenkinsfile", "Jenkinsfile.nightly", "ci/release.Jenkinsfile"]
    """
    return sorted(set(repo.glob(*FILES)))

def libraries(repo):
    """Lists the shared libraries Jenkinsfiles load, with their versions.

    Covers `@Library('name@ref')`, `@Library(['a@ref', 'b@ref'])` and
    `library('name@ref')`. Libraries loaded without a version are listed
    with an empty ref.

    Returns:
        The sorted, de-duplicated `name@ref` strings.
    Example:
        before Jenkinsfile:
            @Library(['ci-lib@v2.3', 'notify@1.0']) _
            library 'deploy-lib@main'
        call:
            libraries(repo)
        returns:
            ["ci-lib@v2.3", "deploy-lib@main", "notify@1.0"]
    """
    found = set()
    quoted = r"""['"]([^'"@\s]+)(@[^'"\s]*)?['"]"""
    for path in jenkinsfiles(repo):
        for line in repo.read(path).split("\n"):
            names = []
            g = re.groups(r"""\blibrary\s*\(?\s*identifier\s*:\s*(['"][^'"]+['"])""", line)
            if g:
                names = [g[0]]
            else:
                g = re.groups(r"@Library\s*\(([^)]*)\)", line) or re.groups(r"""\blibrary\s*\(?\s*(['"][^'"]+['"])""", line)
                if g:
                    names = re.find_all(quoted, g[0])
            for n in names:
                q = re.groups(quoted, n)
                if q:
                    found.add(q[0] + (q[1] or "@"))
    return sorted(found)

def bump_library(repo, name, version, from_ref = None):
    """Moves a shared library to a new version in every Jenkinsfile.

    Libraries loaded without a version are left alone, since they follow
    the default version configured in Jenkins.

    Args:
        name: the library name, as configured in Jenkins.
        version: the new branch or tag, such as `v3.0.0`.
        from_ref: when set, only this exact version is replaced.
    Returns:
        The list of paths that changed.
    Example:
        before Jenkinsfile:
            @Library('ci-lib@v2.3') _
            pipeline {
              agent any
            }
        call:
            bump_library(repo, "ci-lib", "v3.0.0")
        after Jenkinsfile:
            @Library('ci-lib@v3.0.0') _
            pipeline {
              agent any
            }
        returns:
            ["Jenkinsfile"]
    """
    return _bump(repo, FILES, name, version, sep = "@", from_ref = from_ref)

def bump_image(repo, image, from_tag, to_tag):
    """Moves a container image used by an agent or step from one tag to another.

    Matches the image inside quotes anywhere in a Jenkinsfile, which covers
    `docker { image 'node:18' }`, `agent { docker 'node:18' }`,
    `docker.image('node:18')` and Kubernetes pod templates. A variant suffix
    is kept, so `maven:3.8-eclipse-temurin-11` becomes
    `maven:3.9-eclipse-temurin-11` when moving from `3.8` to `3.9`.

    Args:
        image: the image name as written, including any registry.
        from_tag: the tag to replace.
        to_tag: the tag to put in its place.
    Returns:
        The list of paths that changed.
    Example:
        before Jenkinsfile:
            pipeline {
              agent {
                docker { image 'maven:3.8-eclipse-temurin-11' }
              }
              stages {
                stage('lint') {
                  steps { script { docker.image("node:18").inside { sh 'npm run lint' } } }
                }
              }
            }
        call:
            bump_image(repo, "maven", "3.8", "3.9")
            bump_image(repo, "node", "18", "22")
        after Jenkinsfile:
            pipeline {
              agent {
                docker { image 'maven:3.9-eclipse-temurin-11' }
              }
              stages {
                stage('lint') {
                  steps { script { docker.image("node:22").inside { sh 'npm run lint' } } }
                }
              }
            }
        returns:
            ["Jenkinsfile"]
    """
    pattern = r"""(['"])""" + re.escape(image) + ":" + re.escape(from_tag) + r"""((?:-[^'"\s]*)?['"])"""
    repl = "${1}" + _lit(image) + ":" + _lit(to_tag) + "${2}"
    changed = []
    for path in jenkinsfiles(repo):
        content = repo.read(path)
        updated = re.sub(pattern, repl, content)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed

def set_tool(repo, tool, name, from_name = None):
    """Sets the installation a `tools` block asks for, such as `maven 'Maven 3.9'`.

    Only lines inside a `tools { ... }` block change.

    Args:
        tool: the tool type, such as `maven`, `jdk` or `nodejs`.
        name: the installation name configured in Jenkins.
        from_name: when set, only this installation is replaced.
    Returns:
        The list of paths that changed.
    Example:
        before Jenkinsfile:
            pipeline {
              agent any
              tools {
                maven 'Maven 3.8'
                jdk 'jdk11'
              }
            }
        call:
            set_tool(repo, "jdk", "jdk21")
        after Jenkinsfile:
            pipeline {
              agent any
              tools {
                maven 'Maven 3.8'
                jdk 'jdk21'
              }
            }
        returns:
            ["Jenkinsfile"]
    """
    changed = []
    for path in jenkinsfiles(repo):
        content = repo.read(path)
        lines = content.split("\n")
        tool_re = r"""(\b""" + re.escape(tool) + r"""\s+['"])([^'"]+)(['"])"""
        depth = 0
        for i, line in enumerate(lines):
            # The part of the line inside a tools { } block, if any.
            start = 0
            if depth == 0:
                m = re.search(r"\btools\s*\{", line)
                if m == None:
                    continue
                start = line.index(m) + len(m)
                depth = 1
            end = start
            for _ in range(len(line) - start):
                if end >= len(line) or depth == 0:
                    break
                c = line[end]
                depth += 1 if c == "{" else (-1 if c == "}" else 0)
                end += 1
            part = line[start:end]
            for m in re.find_all(tool_re, part):
                g = re.groups(tool_re, m)
                if g[1] != name and (from_name == None or g[1] == from_name):
                    part = part.replace(m, g[0] + name + g[2], 1)
            lines[i] = line[:start] + part + line[end:]
        updated = "\n".join(lines)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed
