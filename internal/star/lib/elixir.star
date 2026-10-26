"""Bump Mix dependencies and the Elixir version requirement in mix.exs.

Edits are line-based, so comments and layout stay as they were. mix.lock
isn't updated; run `mix deps.update <name>` in a `run` step when it is
committed. For .tool-versions, see `runtime.set_version(repo, "elixir", ...)`.
"""

def _lit(s):
    return s.replace("$", "$$")

def projects(repo):
    """Lists the repo's mix.exs files.

    Returns:
        The sorted paths of every mix.exs outside deps directories.
    Example:
        before mix.exs:
            defmodule App.MixProject do
        before apps/web/mix.exs:
            defmodule Web.MixProject do
        before deps/x/mix.exs:
            x
        call:
            projects(repo)
        returns:
            ["apps/web/mix.exs", "mix.exs"]
    """
    return repo.glob("**/mix.exs", exclude = ["**/deps/**", "**/_build/**"])

def set_dep(repo, name, requirement):
    """Sets a dependency's requirement wherever a mix.exs lists it with one.

    Covers `{:phoenix, "~> 1.7"}` with any options after it. Dependencies
    from `git:` or `path:` without a version are left alone.

    Args:
        name: the dependency, without the leading colon, such as `phoenix`.
        requirement: the requirement, such as `~> 1.7.14`.
    Returns:
        The list of paths that changed.
    Example:
        before mix.exs:
            defp deps do
              [
                {:phoenix, "~> 1.7.0"},
                {:ecto_sql, "~> 3.10", only: :prod},
                {:credo, "~> 1.7", only: [:dev, :test], runtime: false}
              ]
            end
        call:
            set_dep(repo, "ecto_sql", "~> 3.12")
        after mix.exs:
            defp deps do
              [
                {:phoenix, "~> 1.7.0"},
                {:ecto_sql, "~> 3.12", only: :prod},
                {:credo, "~> 1.7", only: [:dev, :test], runtime: false}
              ]
            end
        returns:
            ["mix.exs"]
    """
    pattern = r"""(\{\s*:""" + re.escape(name) + r"""\s*,\s*")([^"]+)(")"""
    changed = []
    for path in projects(repo):
        content = repo.read(path)
        updated = re.sub(pattern, "${1}" + _lit(requirement) + "${3}", content)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed

def set_elixir(repo, requirement):
    """Sets the `elixir:` requirement in each project definition.

    Args:
        requirement: the requirement, such as `~> 1.17`.
    Returns:
        The list of paths that changed.
    Example:
        before mix.exs:
            def project do
              [
                app: :app,
                elixir: "~> 1.14",
                deps: deps()
              ]
            end
        call:
            set_elixir(repo, "~> 1.17")
        after mix.exs:
            def project do
              [
                app: :app,
                elixir: "~> 1.17",
                deps: deps()
              ]
            end
        returns:
            ["mix.exs"]
    """
    changed = []
    for path in projects(repo):
        content = repo.read(path)
        updated = re.sub(r"""(\belixir:\s*")[^"]*(")""", "${1}" + _lit(requirement) + "${2}", content)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed
