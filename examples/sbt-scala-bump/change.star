load("@fleet//sbt.star", "build_files", "set_sbt_version", "set_scala_version")

def change(repo):
    want = env["SCALA"]
    old = set()
    for path in build_files(repo):
        for v in re.find_all(r"2\.13\.\d+", repo.read(path)):
            if v != want:
                old.add(v)
    for v in sorted(old):
        set_scala_version(repo, want, from_version = v)
    repo.set_var("scala", want)
    set_sbt_version(repo, env["SBT"])
