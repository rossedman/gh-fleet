---
title: maven.star
---

# maven.star

Bump dependency, plugin and property versions in Maven pom.xml files.

Edits are line-based, so comments and formatting stay as they were. Each
`<groupId>`, `<artifactId>` and `<version>` is expected on its own line, as
Maven and most editors write them.

```python
load("@fleet//maven.star", "poms", "set_property", "set_dependency")
```

| Function | What it does |
| --- | --- |
| [`poms`](#poms) | Lists the repo's pom.xml files. |
| [`set_property`](#set_property) | Sets a `<properties>` entry, such as `java.version`, in every pom.xml that defines it. |
| [`set_dependency`](#set_dependency) | Sets the version of a dependency or plugin wherever a pom.xml declares one. |

## poms

```python
poms(repo)
```

Lists the repo's pom.xml files.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The sorted paths of every pom.xml outside target directories.

**Example**

```python
poms(repo)
# returns ["core/pom.xml", "pom.xml"]
```

=== "Before"

    `pom.xml`

    ```text
    <project/>
    ```

    `core/pom.xml`

    ```text
    <project/>
    ```

    `target/classes/pom.xml`

    ```text
    <project/>
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/maven.star#L11)

## set_property

```python
set_property(repo, name, value)
```

Sets a `<properties>` entry, such as `java.version`, in every pom.xml that defines it.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `name` | The property name. |
| `value` | The new value. |

**Returns:** The list of paths that changed.

**Example**

```python
set_property(repo, "java.version", "21")
# returns ["pom.xml"]
```

=== "Before"

    `pom.xml`

    ```text
    <project>
      <properties>
        <java.version>17</java.version>
        <jackson.version>2.15.2</jackson.version>
      </properties>
    </project>
    ```

=== "After"

    `pom.xml`

    ```text
    <project>
      <properties>
        <java.version>21</java.version>
        <jackson.version>2.15.2</jackson.version>
      </properties>
    </project>
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/maven.star#L49)

## set_dependency

```python
set_dependency(repo, group, artifact, version)
```

Sets the version of a dependency or plugin wherever a pom.xml declares one.

Covers `<dependency>` and `<plugin>` entries, including those under
`<dependencyManagement>` and `<pluginManagement>`. When the version is
a property reference such as `${jackson.version}`, the property is set
instead. A plugin without a `<groupId>` matches the group
`org.apache.maven.plugins`, as Maven assumes. Entries without a
`<version>` are left alone.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `group` | The `groupId`. |
| `artifact` | The `artifactId`. |
| `version` | The new version. |

**Returns:** The list of paths that changed.

**Example**

```python
set_dependency(repo, "com.fasterxml.jackson.core", "jackson-databind", "2.17.2")
set_dependency(repo, "org.slf4j", "slf4j-api", "2.0.16")
# returns ["pom.xml"]
```

=== "Before"

    `pom.xml`

    ```text
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
    ```

=== "After"

    `pom.xml`

    ```text
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
    ```

**Used in:** [`jvm-dependency-bump`](https://github.com/rossedman/gh-fleet/tree/main/examples/jvm-dependency-bump)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/maven.star#L86)
