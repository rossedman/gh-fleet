load("@fleet//dotnet.star", "set_package", "set_sdk_version", "set_target_framework")

def change(repo):
    if set_target_framework(repo, env["TO"], from_framework = env["FROM"]):
        set_sdk_version(repo, env["SDK"])
        set_package(repo, "Microsoft.AspNetCore.Mvc.Testing", "8.0.8")
