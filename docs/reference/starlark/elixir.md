---
title: elixir.star
---

# elixir.star

Bump Mix dependencies and the Elixir version requirement in mix.exs.

Edits are line-based, so comments and layout stay as they were. mix.lock
isn't updated; run `mix deps.update <name>` in a `run` step when it is
committed. For .tool-versions, see `runtime.set_version(repo, "elixir", ...)`.

```python
load("@fleet//elixir.star", "projects", "set_dep", "set_elixir")
```

| Function | What it does |
| --- | --- |
| [`projects`](#projects) | Lists the repo's mix.exs files. |
| [`set_dep`](#set_dep) | Sets a dependency's requirement wherever a mix.exs lists it with one. |
| [`set_elixir`](#set_elixir) | Sets the `elixir:` requirement in each project definition. |

## projects

```python
projects(repo)
```

Lists the repo's mix.exs files.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The sorted paths of every mix.exs outside deps directories.

**Example**

```python
projects(repo)
# returns ["apps/web/mix.exs", "mix.exs"]
```

=== "Before"

    `mix.exs`

    ```text
    defmodule App.MixProject do
    ```

    `apps/web/mix.exs`

    ```text
    defmodule Web.MixProject do
    ```

    `deps/x/mix.exs`

    ```text
    x
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/elixir.star#L11)

## set_dep

```python
set_dep(repo, name, requirement)
```

Sets a dependency's requirement wherever a mix.exs lists it with one.

A newer version already there is kept, so a campaign never lowers one.

Covers `{:phoenix, "~> 1.7"}` with any options after it. Dependencies
from `git:` or `path:` without a version are left alone.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `name` | The dependency, without the leading colon, such as `phoenix`. |
| `requirement` | The requirement, such as `~> 1.7.14`. |

**Returns:** The list of paths that changed.

**Example**

```python
set_dep(repo, "ecto_sql", "~> 3.12")
# returns ["mix.exs"]
```

=== "Before"

    `mix.exs`

    ```text
    defp deps do
      [
        {:phoenix, "~> 1.7.0"},
        {:ecto_sql, "~> 3.10", only: :prod},
        {:credo, "~> 1.7", only: [:dev, :test], runtime: false}
      ]
    end
    ```

=== "After"

    `mix.exs`

    ```text
    defp deps do
      [
        {:phoenix, "~> 1.7.0"},
        {:ecto_sql, "~> 3.12", only: :prod},
        {:credo, "~> 1.7", only: [:dev, :test], runtime: false}
      ]
    end
    ```

**Used in:** [`elixir-phoenix`](https://github.com/rossedman/gh-fleet/tree/main/examples/elixir-phoenix)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/elixir.star#L30)

## set_elixir

```python
set_elixir(repo, requirement)
```

Sets the `elixir:` requirement in each project definition.

A newer version already there is kept, so a campaign never lowers one.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `requirement` | The requirement, such as `~> 1.17`. |

**Returns:** The list of paths that changed.

**Example**

```python
set_elixir(repo, "~> 1.17")
# returns ["mix.exs"]
```

=== "Before"

    `mix.exs`

    ```text
    def project do
      [
        app: :app,
        elixir: "~> 1.14",
        deps: deps()
      ]
    end
    ```

=== "After"

    `mix.exs`

    ```text
    def project do
      [
        app: :app,
        elixir: "~> 1.17",
        deps: deps()
      ]
    end
    ```

**Used in:** [`elixir-phoenix`](https://github.com/rossedman/gh-fleet/tree/main/examples/elixir-phoenix)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/elixir.star#L75)
