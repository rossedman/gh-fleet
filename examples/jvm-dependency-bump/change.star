load("@fleet//gradle.star", gradle_dependency = "set_dependency")
load("@fleet//maven.star", maven_dependency = "set_dependency")

def change(repo):
    group, artifact, version = env["GROUP"], env["ARTIFACT"], env["VERSION"]
    changed = maven_dependency(repo, group, artifact, version)
    changed += gradle_dependency(repo, group + ":" + artifact, version)
    repo.set_var("files", ", ".join(changed))
