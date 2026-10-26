load("@fleet//drone.star", "bump_image")

def change(repo):
    bump_image(repo, "golang", env["FROM"], env["TO"])
