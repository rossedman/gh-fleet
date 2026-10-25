load("@fleet//files.star", "ensure_lines")

def change(repo):
    ensure_lines(repo, ".gitignore", env["ENTRIES"], header = "# Local environment files")
