load("@fleet//gha.star", "pin")

def change(repo):
    pin(repo, repo.vars.get("shas", ""))
