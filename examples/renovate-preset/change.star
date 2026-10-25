load("@fleet//jsonfile.star", "set")

def change(repo):
    set(repo, "renovate.json", "extends", [env["PRESET"]])
