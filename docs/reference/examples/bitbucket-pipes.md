---
title: bitbucket-pipes
---

# bitbucket-pipes

Move Bitbucket Pipelines to current AWS pipes and Node 22.

**Uses:** [`bitbucket.bump_image`](../starlark/bitbucket.md#bump_image), [`bitbucket.bump_pipe`](../starlark/bitbucket.md#bump_pipe), [`bitbucket.pipes`](../starlark/bitbucket.md#pipes)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from bitbucket-pipes --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//bitbucket.star", "bump_image", "bump_pipe", "pipes")

    def change(repo):
        old = [p.split(":")[1] for p in pipes(repo) if p.split(":")[0] == env["PIPE"]]
        repo.set_var("previous", ", ".join(old) or "none")
        bump_pipe(repo, env["PIPE"], env["VERSION"])
        bump_image(repo, "node", "18", "22")
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: bitbucket-aws-pipes
    description: Move Bitbucket Pipelines to current AWS pipes and Node 22

    targets:
      orgs: [acme]
      include:
        files: [bitbucket-pipelines.yml]

    change:
      steps:
        # Moves the aws-s3-deploy pipe and the node image, and reports the pipe
        # versions it replaced.
        - name: bitbucket-pipes
          starlark: ./change.star
          timeout: 1m
      env:
        PIPE: atlassian/aws-s3-deploy
        VERSION: "1.6.1"

    pr:
      title: "ci: aws-s3-deploy 1.6.1 and node 22"
      body: |
        Moves {{ .Repo.Name }} to aws-s3-deploy 1.6.1 (from {{ .Vars.previous }})
        and Node 22.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/bitbucket-pipes)
