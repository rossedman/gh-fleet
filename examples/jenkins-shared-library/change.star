load("@fleet//refs.star", "bump")

def change(repo):
    bump(repo, ["Jenkinsfile", "**/Jenkinsfile", "**/*.groovy"], "ci-lib", env["VERSION"])
    bump(repo, [".circleci/*.yml", ".circleci/*.yaml"], "acme/ci-orb", env["ORB"])
