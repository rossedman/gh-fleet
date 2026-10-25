# Require Node >= env["NODE"] in every package.json outside node_modules.
load("@fleet//npm.star", "packages", "read_package", "write_package")

def required_major(spec):
    """Returns the lowest major version a range like ">=18" or "^20.1" allows, or None."""
    m = re.groups(r"(\d+)", spec)
    return int(m[0]) if m else None

def change(repo):
    want = int(env["NODE"])
    repo.set_var("node", want)
    previous = []
    for path in packages(repo):
        pkg = read_package(repo, path)
        engines = pkg.get("engines", {})
        current = engines.get("node", "")
        have = required_major(current)
        if have != None and have >= want:
            continue  # already requires this version or newer
        previous.append(current or "unset")
        engines["node"] = ">=%d" % want
        pkg["engines"] = engines
        write_package(repo, path, pkg)
    repo.set_var("previous", ", ".join(previous) or "already set")
