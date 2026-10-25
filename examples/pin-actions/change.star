load("@fleet//gha.star", "unpinned")

def change(repo):
    repo.set_var("refs", unpinned(repo))
