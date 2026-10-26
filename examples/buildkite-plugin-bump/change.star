load("@fleet//buildkite.star", "bump_plugin", "plugins")

def change(repo):
    name = env["PLUGIN"]
    old = [p.split("#")[1] for p in plugins(repo) if p.split("#")[0] == name]
    repo.set_var("previous", ", ".join(old) or "none")
    bump_plugin(repo, name, env["VERSION"])
