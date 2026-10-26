"""Move Terragrunt and Terraform module sources to a new `?ref=` tag.

Edits are line-based, so comments and alignment stay as they were.
"""

load("./refs.star", _bump = "bump", _find = "find")

FILES = ["**/*.hcl", "**/*.tf"]
EXCLUDE = ["**/.terraform/**", "**/.terragrunt-cache/**"]

def sources(repo):
    """Lists every module source pinned with `?ref=`.

    Returns:
        A list of dicts with `path`, `source` and `ref`, in file order.
    Example:
        before live/prod/vpc/terragrunt.hcl:
            terraform {
              source = "git::https://github.com/acme/modules.git//vpc?ref=v1.2.0"
            }
        call:
            [s["source"] + " " + s["ref"] for s in sources(repo)]
        returns:
            ["git::https://github.com/acme/modules.git//vpc v1.2.0"]
    """
    return [{"path": r["path"], "source": r["name"], "ref": r["ref"]} for r in _find(repo, FILES, "*", sep = "?ref=", exclude = EXCLUDE)]

def set_source_ref(repo, source, ref):
    """Sets the `?ref=` of every module source that matches.

    Covers `source` in terragrunt.hcl `terraform` blocks and in Terraform
    `module` blocks. Sources pinned to a commit SHA are left alone.

    Args:
        source: the source before `?ref=`; `*` matches any run of
            characters, so `git::https://github.com/acme/modules.git//*`
            covers every module in that repository.
        ref: the new tag or branch.
    Returns:
        The list of paths that changed.
    Example:
        before live/prod/vpc/terragrunt.hcl:
            terraform {
              source = "git::https://github.com/acme/modules.git//vpc?ref=v1.2.0"
            }
        before main.tf:
            module "db" {
              source = "git::https://github.com/acme/modules.git//rds?ref=v1.1.0"
            }
        call:
            set_source_ref(repo, "git::https://github.com/acme/modules.git//*", "v1.3.0")
        after live/prod/vpc/terragrunt.hcl:
            terraform {
              source = "git::https://github.com/acme/modules.git//vpc?ref=v1.3.0"
            }
        after main.tf:
            module "db" {
              source = "git::https://github.com/acme/modules.git//rds?ref=v1.3.0"
            }
        returns:
            ["live/prod/vpc/terragrunt.hcl", "main.tf"]
    """
    return _bump(repo, FILES, source, ref, sep = "?ref=", exclude = EXCLUDE)
