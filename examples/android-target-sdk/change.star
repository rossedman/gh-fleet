load("@fleet//android.star", "set_sdk")

def change(repo):
    level = int(env["SDK"])
    set_sdk(repo, compile = level, target = level)
