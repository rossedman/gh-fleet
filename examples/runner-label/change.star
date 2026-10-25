load("@fleet//gha.star", "workflows")
load("@fleet//yamlpath.star", "set", "values")

def change(repo):
    pinned = []
    for path in workflows(repo):
        for key, label in values(repo, path, "jobs.*.runs-on").items():
            if label == env["FROM"]:
                set(repo, path, key, env["TO"])
                pinned.append(key.split(".")[1])
    repo.set_var("jobs", ", ".join(pinned) or "no jobs")
