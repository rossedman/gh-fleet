load("@fleet//golangci.star", "enable_linter")

def change(repo):
    enable_linter(repo, "gosec")
