load("@fleet//npm.star", "set_npmrc", "set_package_manager")

def change(repo):
    repo.set_var("pnpm", env["PNPM"])
    set_package_manager(repo, env["PNPM"])
    set_npmrc(repo, "engine-strict", True)
