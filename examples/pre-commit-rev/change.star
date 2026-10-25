load("@fleet//precommit.star", "revs", "set_rev")

def change(repo):
    repo.set_var("previous", revs(repo).get(env["HOOKS"], "unknown"))
    set_rev(repo, env["HOOKS"], env["REV"])
