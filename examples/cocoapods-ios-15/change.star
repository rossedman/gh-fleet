load("@fleet//cocoapods.star", "set_platform", "set_pod")

def change(repo):
    set_platform(repo, "ios", env["IOS"])
    set_pod(repo, "Alamofire", "~> 5.9")
