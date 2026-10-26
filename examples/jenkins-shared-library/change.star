load("@fleet//circleci.star", "bump_orb")
load("@fleet//jenkins.star", "bump_library")

def change(repo):
    bump_library(repo, "ci-lib", env["VERSION"])
    bump_orb(repo, "acme/ci-orb", env["ORB"])
