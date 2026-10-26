load("@fleet//toml.star", "get", "set")

def change(repo):
    if not repo.exists("pyproject.toml"):
        return

    # A project that already picked a line length keeps it.
    if get(repo, "pyproject.toml", "tool.ruff", "line-length") == None:
        set(repo, "pyproject.toml", "tool.ruff", "line-length", 100)

    # extend-select adds the shared rules to whatever the project selects,
    # instead of replacing its own list.
    set(repo, "pyproject.toml", "tool.ruff.lint", "extend-select", ["E", "F", "I"])
