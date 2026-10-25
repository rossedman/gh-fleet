load("@fleet//dependabot.star", "write_if_missing")

def change(repo):
    found = write_if_missing(repo, interval = "weekly")
    repo.set_var("ecosystems", ", ".join(found))
