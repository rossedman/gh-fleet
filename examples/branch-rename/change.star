load("@fleet//gha.star", "rename_branch")

def change(repo):
    rename_branch(repo, env["OLD"], env["NEW"])
