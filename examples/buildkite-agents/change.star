load("@fleet//buildkite.star", "set_queue", "set_step_defaults")

def change(repo):
    set_queue(repo, env["TO"], from_queue = env["FROM"])
    set_step_defaults(repo, {"timeout_in_minutes": int(env["TIMEOUT"])})
