"""Move Serverless Framework services to a new Lambda runtime.

AWS retires Lambda runtimes on a schedule, so this is a recurring
fleet-wide change. Edits are line-based, so comments and variables in
serverless.yml stay as they were.
"""

load("./yamlpath.star", _set = "set", _values = "values")

def files(repo):
    """Lists the repo's Serverless configs.

    Returns:
        The sorted paths of every serverless.yml and serverless.yaml
        outside node_modules.
    Example:
        before serverless.yml:
            service: api
        before services/jobs/serverless.yaml:
            service: jobs
        call:
            files(repo)
        returns:
            ["serverless.yml", "services/jobs/serverless.yaml"]
    """
    return repo.glob("**/serverless.yml", "**/serverless.yaml", exclude = ["**/node_modules/**"])

def set_runtime(repo, runtime, from_runtime = None):
    """Sets the Lambda runtime of the provider and of every function that overrides it.

    Args:
        runtime: the new runtime, such as `nodejs20.x` or `python3.12`.
        from_runtime: when set, only this runtime is replaced, so functions
            on other languages stay as they are.
    Returns:
        The list of paths that changed.
    Example:
        before serverless.yml:
            service: api
            provider:
              name: aws
              runtime: nodejs16.x  # retired
            functions:
              hello:
                handler: handler.hello
              report:
                handler: report.main
                runtime: python3.12
        call:
            set_runtime(repo, "nodejs20.x", from_runtime = "nodejs16.x")
        after serverless.yml:
            service: api
            provider:
              name: aws
              runtime: nodejs20.x  # retired
            functions:
              hello:
                handler: handler.hello
              report:
                handler: report.main
                runtime: python3.12
        returns:
            ["serverless.yml"]
    """
    changed = []
    for path in files(repo):
        dirty = False
        for key in ("provider.runtime", "functions.*.runtime"):
            for k, v in _values(repo, path, key).items():
                if v != runtime and (from_runtime == None or v == from_runtime):
                    _set(repo, path, k, runtime)
                    dirty = True
        if dirty:
            changed.append(path)
    return changed
