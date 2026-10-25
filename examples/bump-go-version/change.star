load("@fleet//gha.star", "set_input")
load("@fleet//gomod.star", "set_go_version")

def change(repo):
    version = env["GO_VERSION"]
    set_go_version(repo, version)
    # Only numeric pins: leave `go-version: stable` and matrix expressions alone.
    set_input(repo, "go-version", version, match = r"^\d")
