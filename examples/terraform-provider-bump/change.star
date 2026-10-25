load("@fleet//terraform.star", "set_provider_version", "set_required_version")

def change(repo):
    if set_provider_version(repo, env["PROVIDER"], env["CONSTRAINT"]):
        set_required_version(repo, env["TERRAFORM"])
