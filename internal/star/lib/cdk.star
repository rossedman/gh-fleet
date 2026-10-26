"""Set AWS CDK context values, such as feature flags, in cdk.json.

Edits keep cdk.json's key order and indentation. The aws-cdk-lib version
itself lives in package.json, pyproject.toml, pom.xml or go.mod; use
`npm.set_dependency`, `python.set_requirement`, `maven.set_dependency` or a
`go get` step for it.
"""

load("./jsonfile.star", _get = "get", _set = "set")

def apps(repo):
    """Lists the repo's CDK apps.

    Returns:
        The sorted paths of every cdk.json outside node_modules and cdk.out.
    Example:
        before cdk.json:
            {"app": "npx ts-node bin/app.ts"}
        before infra/cdk.json:
            {"app": "python3 app.py"}
        call:
            apps(repo)
        returns:
            ["cdk.json", "infra/cdk.json"]
    """
    return repo.glob("**/cdk.json", exclude = ["**/node_modules/**", "**/cdk.out/**"])

def set_context(repo, key, value, only_existing = False):
    """Sets a context value, such as a feature flag, in every cdk.json.

    Args:
        key: the context key, such as `@aws-cdk/aws-lambda:recognizeLayerVersion`.
        value: any JSON value.
        only_existing: when True, only apps that already set the key change.
    Returns:
        The list of paths that changed.
    Example:
        before cdk.json:
            {
              "app": "npx ts-node bin/app.ts",
              "context": {
                "@aws-cdk/core:stackRelativeExports": true
              }
            }
        call:
            set_context(repo, "@aws-cdk/aws-s3:createDefaultLoggingPolicy", True)
        after cdk.json:
            {
              "app": "npx ts-node bin/app.ts",
              "context": {
                "@aws-cdk/core:stackRelativeExports": true,
                "@aws-cdk/aws-s3:createDefaultLoggingPolicy": true
              }
            }
        returns:
            ["cdk.json"]
    """
    changed = []
    for path in apps(repo):
        if only_existing and _get(repo, path, ["context", key]) == None:
            continue
        if _set(repo, path, ["context", key], value):
            changed.append(path)
    return changed
