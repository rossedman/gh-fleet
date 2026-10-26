load("@fleet//jenkins.star", "bump_image", "set_tool")

def change(repo):
    bump_image(repo, "maven", env["FROM"], env["TO"])
    set_tool(repo, "jdk", env["JDK"])
