---
title: jenkins.star
---

# jenkins.star

Edit Jenkinsfiles: shared library versions, agent images and tool versions.

Jenkinsfiles are Groovy, which can't be parsed safely here, so edits are
line-based matches on the common declarative and scripted forms.

```python
load("@fleet//jenkins.star", "jenkinsfiles", "libraries", "bump_library", "bump_image", "set_tool")
```

| Function | What it does |
| --- | --- |
| [`jenkinsfiles`](#jenkinsfiles) | Lists the repo's Jenkinsfiles. |
| [`libraries`](#libraries) | Lists the shared libraries Jenkinsfiles load, with their versions. |
| [`bump_library`](#bump_library) | Moves a shared library to a new version in every Jenkinsfile. |
| [`bump_image`](#bump_image) | Moves a container image used by an agent or step from one tag to another. |
| [`set_tool`](#set_tool) | Sets the installation a `tools` block asks for, such as `maven 'Maven 3.9'`. |

## jenkinsfiles

```python
jenkinsfiles(repo)
```

Lists the repo's Jenkinsfiles.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The sorted paths of every Jenkinsfile, Jenkinsfile.*, *.jenkinsfile
and *.Jenkinsfile.

**Example**

```python
jenkinsfiles(repo)
# returns ["Jenkinsfile", "Jenkinsfile.nightly", "ci/release.Jenkinsfile"]
```

=== "Before"

    `Jenkinsfile`

    ```text
    pipeline {}
    ```

    `ci/release.Jenkinsfile`

    ```text
    pipeline {}
    ```

    `Jenkinsfile.nightly`

    ```text
    pipeline {}
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/jenkins.star#L14)

## libraries

```python
libraries(repo)
```

Lists the shared libraries Jenkinsfiles load, with their versions.

Covers `@Library('name@ref')`, `@Library(['a@ref', 'b@ref'])` and
`library('name@ref')`. Libraries loaded without a version are listed
with an empty ref.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The sorted, de-duplicated `name@ref` strings.

**Example**

```python
libraries(repo)
# returns ["ci-lib@v2.3", "deploy-lib@main", "notify@1.0"]
```

=== "Before"

    `Jenkinsfile`

    ```text
    @Library(['ci-lib@v2.3', 'notify@1.0']) _
    library 'deploy-lib@main'
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/jenkins.star#L34)

## bump_library

```python
bump_library(repo, name, version, from_ref = None)
```

Moves a shared library to a new version in every Jenkinsfile.

Libraries loaded without a version are left alone, since they follow
the default version configured in Jenkins.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `name` | The library name, as configured in Jenkins. |
| `version` | The new branch or tag, such as `v3.0.0`. |
| `from_ref` | When set, only this exact version is replaced. |

**Returns:** The list of paths that changed.

**Example**

```python
bump_library(repo, "ci-lib", "v3.0.0")
# returns ["Jenkinsfile"]
```

=== "Before"

    `Jenkinsfile`

    ```text
    @Library('ci-lib@v2.3') _
    pipeline {
      agent any
    }
    ```

=== "After"

    `Jenkinsfile`

    ```text
    @Library('ci-lib@v3.0.0') _
    pipeline {
      agent any
    }
    ```

**Used in:** [`jenkins-shared-library`](../examples/jenkins-shared-library.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/jenkins.star#L70)

## bump_image

```python
bump_image(repo, image, from_tag, to_tag)
```

Moves a container image used by an agent or step from one tag to another.

Matches the image inside quotes anywhere in a Jenkinsfile, which covers
`docker { image 'node:18' }`, `agent { docker 'node:18' }`,
`docker.image('node:18')` and Kubernetes pod templates. A variant suffix
is kept, so `maven:3.8-eclipse-temurin-11` becomes
`maven:3.9-eclipse-temurin-11` when moving from `3.8` to `3.9`.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `image` | The image name as written, including any registry. |
| `from_tag` | The tag to replace. |
| `to_tag` | The tag to put in its place. |

**Returns:** The list of paths that changed.

**Example**

```python
bump_image(repo, "maven", "3.8", "3.9")
bump_image(repo, "node", "18", "22")
# returns ["Jenkinsfile"]
```

=== "Before"

    `Jenkinsfile`

    ```text
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
    ```

=== "After"

    `Jenkinsfile`

    ```text
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
    ```

**Used in:** [`jenkins-agent-image`](../examples/jenkins-agent-image.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/jenkins.star#L100)

## set_tool

```python
set_tool(repo, tool, name, from_name = None)
```

Sets the installation a `tools` block asks for, such as `maven 'Maven 3.9'`.

Only lines inside a `tools { ... }` block change.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `tool` | The tool type, such as `maven`, `jdk` or `nodejs`. |
| `name` | The installation name configured in Jenkins. |
| `from_name` | When set, only this installation is replaced. |

**Returns:** The list of paths that changed.

**Example**

```python
set_tool(repo, "jdk", "jdk21")
# returns ["Jenkinsfile"]
```

=== "Before"

    `Jenkinsfile`

    ```text
    pipeline {
      agent any
      tools {
        maven 'Maven 3.8'
        jdk 'jdk11'
      }
    }
    ```

=== "After"

    `Jenkinsfile`

    ```text
    pipeline {
      agent any
      tools {
        maven 'Maven 3.8'
        jdk 'jdk21'
      }
    }
    ```

**Used in:** [`jenkins-agent-image`](../examples/jenkins-agent-image.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/jenkins.star#L155)
