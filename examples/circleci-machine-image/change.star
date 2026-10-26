load("@fleet//circleci.star", "bump_image", "set_machine_image")

def change(repo):
    set_machine_image(repo, env["MACHINE"], from_image = "ubuntu-2004:*")
    bump_image(repo, "cimg/base", "2022.06", "2024.01")
