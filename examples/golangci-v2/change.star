load("@fleet//golangci.star", "config_version")

def change(repo):
    repo.set_var("format", config_version(repo) or "none")
