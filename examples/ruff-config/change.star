load("@fleet//toml.star", "set")

def change(repo):
    if not repo.exists("pyproject.toml"):
        return
    set(repo, "pyproject.toml", "tool.ruff", "line-length", 100)
    set(repo, "pyproject.toml", "tool.ruff.lint", "select", ["E", "F", "I"])
