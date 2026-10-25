load("@fleet//codeowners.star", "find", "set_owners")

def change(repo):
    if find(repo) != None:
        return  # the repo already says who owns it
    team = "@%s/%s" % (repo.owner, repo.vars["team"])
    set_owners(repo, "*", team, header = "# Owners are requested for review on every pull request.")
