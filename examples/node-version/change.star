load("@fleet//gha.star", "set_input")
load("@fleet//runtime.star", "set_version")

def change(repo):
    version = repo.vars["node"]
    set_version(repo, "node", version, create = ".nvmrc")
    set_input(repo, "node-version", version, match = r"^\d")
