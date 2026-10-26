load("@fleet//ansible.star", "set_version")

def change(repo):
    set_version(repo, env["COLLECTION"], env["VERSION"])
