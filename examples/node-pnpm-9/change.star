load("@fleet//npm.star", "read_package", "set_npmrc", "set_package_manager")

def change(repo):
    current = read_package(repo).get("packageManager") or ""
    if not current.startswith("pnpm@") or semver.lowers(current, env["PNPM"]):
        return  # not pinned to pnpm, or already on a newer pnpm
    repo.set_var("pnpm", env["PNPM"])
    set_package_manager(repo, env["PNPM"])
    set_npmrc(repo, "engine-strict", True)
