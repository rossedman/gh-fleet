load("@fleet//gha.star", "set_input")
load("@fleet//python.star", "set_requires_python")
load("@fleet//runtime.star", "set_version")

def change(repo):
    version = env["PYTHON"]
    set_version(repo, "python", version)
    set_requires_python(repo, ">=" + version)
    # Only plain version pins: leave matrix expressions such as ${{ matrix.py }} alone.
    set_input(repo, "python-version", version, match = r"^\d")
