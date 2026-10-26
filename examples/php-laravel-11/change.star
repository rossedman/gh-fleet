load("@fleet//php.star", "set_php", "set_require")

def change(repo):
    changed = set_require(repo, "laravel/framework", env["LARAVEL"])
    set_php(repo, env["PHP"])
    repo.set_var("changed", len(changed) > 0)
