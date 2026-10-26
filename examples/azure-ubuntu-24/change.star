load("@fleet//azure_pipelines.star", "bump_task", "set_vm_image")

def change(repo):
    set_vm_image(repo, env["TO"], from_image = env["FROM"])
    bump_task(repo, "Docker", 2, from_major = 1)
