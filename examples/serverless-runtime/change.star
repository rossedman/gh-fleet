load("@fleet//serverless.star", "set_runtime")

def change(repo):
    set_runtime(repo, env["TO"], from_runtime = env["FROM"])
