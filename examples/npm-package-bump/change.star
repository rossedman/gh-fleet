load("@fleet//npm.star", "set_dependency")

def change(repo):
    changed = set_dependency(repo, env["PACKAGE"], env["RANGE"])
    dirs = [path.removesuffix("package.json").rstrip("/") or "." for path in changed]
    repo.set_var("dirs", dirs)
