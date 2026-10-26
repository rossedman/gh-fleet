load("@fleet//scanignore.star", "prune_expired")

def change(repo):
    repo.set_var("expired", ", ".join(prune_expired(repo, repo.vars["today"])))
