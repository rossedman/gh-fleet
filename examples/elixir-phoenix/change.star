load("@fleet//elixir.star", "set_dep", "set_elixir")
load("@fleet//runtime.star", "set_version")

def change(repo):
    if set_dep(repo, "phoenix", env["PHOENIX"]):
        set_elixir(repo, "~> 1.17")
        set_version(repo, "elixir", "1.17.3-otp-27")
        set_version(repo, "erlang", "27.1")
