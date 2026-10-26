---
title: gitlab.star
---

# gitlab.star

Edit GitLab CI configuration: job and service images, included template refs, CI/CD component versions and defaults.

Edits are line-based, so comments, anchors and `!reference` tags stay as
they were.

```python
load("@fleet//gitlab.star", "configs", "bump_image", "set_include_ref", "components", "bump_component", "set_default")
```

| Function | What it does |
| --- | --- |
| [`configs`](#configs) | Lists the repo's GitLab CI files. |
| [`bump_image`](#bump_image) | Moves a job or service image from one tag to another. |
| [`set_include_ref`](#set_include_ref) | Sets the `ref` of every `include` that pulls templates from another project. |
| [`components`](#components) | Lists the CI/CD components a config includes, with their versions. |
| [`bump_component`](#bump_component) | Moves an included CI/CD component to a new version. |
| [`set_default`](#set_default) | Sets a key under `default:` in .gitlab-ci.yml, adding `default:` if needed. |

## configs

```python
configs(repo)
```

Lists the repo's GitLab CI files.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The sorted paths of .gitlab-ci.yml, of YAML files under .gitlab
(often included locally), and of any *.gitlab-ci.yml.

**Example**

```python
configs(repo)
# returns [".gitlab-ci.yml", ".gitlab/ci/test.yml"]
```

=== "Before"

    `.gitlab-ci.yml`

    ```yaml
    stages: [test]
    ```

    `.gitlab/ci/test.yml`

    ```yaml
    test: {}
    ```

    `ci/other.yml`

    ```yaml
    x: 1
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/gitlab.star#L15)

## bump_image

```python
bump_image(repo, image, from_tag, to_tag)
```

Moves a job or service image from one tag to another.

Covers `image: node:18`, `image: { name: node:18 }` written as a block,
and `services` entries written as `- postgres:15` or `- name: postgres:15`.
A variant suffix is kept, so `node:18-alpine` becomes `node:22-alpine`,
while `node:18.2` is not touched.

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
bump_image(repo, "node", "18", "22")
bump_image(repo, "postgres", "15", "17")
# returns [".gitlab-ci.yml"]
```

=== "Before"

    `.gitlab-ci.yml`

    ```yaml
    default:
      image: node:18-alpine
    test:
      services:
        - postgres:15
        - name: redis:7
      script: npm test
    ```

=== "After"

    `.gitlab-ci.yml`

    ```yaml
    default:
      image: node:22-alpine
    test:
      services:
        - postgres:17
        - name: redis:7
      script: npm test
    ```

**Used in:** [`gitlab-image-bump`](../examples/gitlab-image-bump.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/gitlab.star#L35)

## set_include_ref

```python
set_include_ref(repo, project, ref)
```

Sets the `ref` of every `include` that pulls templates from another project.

A newer version already there is kept, so a campaign never lowers one.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `project` | The project path, as in `include: project:`. |
| `ref` | The new branch, tag or commit. |

**Returns:** The list of paths that changed.

**Example**

```python
set_include_ref(repo, "acme/ci-templates", "v2.0.0")
# returns [".gitlab-ci.yml"]
```

=== "Before"

    `.gitlab-ci.yml`

    ```yaml
    include:
      - project: acme/ci-templates
        ref: v1.4.0  # templates
        file: /go.yml
      - local: .gitlab/ci/test.yml
    ```

=== "After"

    `.gitlab-ci.yml`

    ```yaml
    include:
      - project: acme/ci-templates
        ref: v2.0.0  # templates
        file: /go.yml
      - local: .gitlab/ci/test.yml
    ```

**Used in:** [`gitlab-templates`](../examples/gitlab-templates.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/gitlab.star#L83)

## components

```python
components(repo)
```

Lists the CI/CD components a config includes, with their versions.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** A list of `component@version` strings, in file order.

**Example**

```python
components(repo)
# returns ["gitlab.com/acme/components/go-build@1.2.0", "$CI_SERVER_FQDN/acme/components/sast@~latest"]
```

=== "Before"

    `.gitlab-ci.yml`

    ```yaml
    include:
      - component: gitlab.com/acme/components/go-build@1.2.0
      - component: $CI_SERVER_FQDN/acme/components/sast@~latest
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/gitlab.star#L113)

## bump_component

```python
bump_component(repo, component, version, from_ref = None)
```

Moves an included CI/CD component to a new version.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `component` | The component path before `@`, such as `gitlab.com/acme/components/go-build`; `*` matches any run of path characters. |
| `version` | The new version, such as `1.3.0` or `~latest`. |
| `from_ref` | When set, only this exact version is replaced. |

**Returns:** The list of paths that changed.

**Example**

```python
bump_component(repo, "gitlab.com/acme/components/go-build", "1.3.0")
# returns [".gitlab-ci.yml"]
```

=== "Before"

    `.gitlab-ci.yml`

    ```yaml
    include:
      - component: gitlab.com/acme/components/go-build@1.2.0
        inputs:
          go_version: "1.22"
    ```

=== "After"

    `.gitlab-ci.yml`

    ```yaml
    include:
      - component: gitlab.com/acme/components/go-build@1.3.0
        inputs:
          go_version: "1.22"
    ```

**Used in:** [`gitlab-templates`](../examples/gitlab-templates.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/gitlab.star#L136)

## set_default

```python
set_default(repo, key, value)
```

Sets a key under `default:` in .gitlab-ci.yml, adding `default:` if needed.

Use it for settings every job should inherit, such as `interruptible`,
`timeout` or `retry`. In a config that starts with a `spec:` header
for its inputs, the key goes in the pipeline after the `---`, not the
header.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `key` | The key under `default`, such as `interruptible`; dotted for nested keys, such as `retry.max`. |
| `value` | A string, number, boolean or list. |

**Returns:** True when the file changed.

**Example**

```python
set_default(repo, "interruptible", True)
# returns True
```

=== "Before"

    `.gitlab-ci.yml`

    ```yaml
    stages: [test]
    default:
      image: node:22
    test:
      script: npm test
    ```

=== "After"

    `.gitlab-ci.yml`

    ```yaml
    stages: [test]
    default:
      image: node:22
      interruptible: true
    test:
      script: npm test
    ```

**Used in:** [`gitlab-templates`](../examples/gitlab-templates.md)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/gitlab.star#L165)
