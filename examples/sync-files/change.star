load("@fleet//files.star", "sync")

def change(repo):
    sync(repo, "files")
