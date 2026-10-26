load("@fleet//gradle.star", "set_wrapper_version")

def change(repo):
    set_wrapper_version(repo, env["GRADLE"], sha256 = {"bin": env["SHA256_BIN"], "all": env["SHA256_ALL"]})
