---
title: jenkins-agent-image
---

# jenkins-agent-image

Move Jenkins builds to Maven 3.9 and JDK 21.

**Uses:** [`jenkins.bump_image`](../starlark/jenkins.md#bump_image), [`jenkins.set_tool`](../starlark/jenkins.md#set_tool)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from jenkins-agent-image --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//jenkins.star", "bump_image", "set_tool")

    def change(repo):
        bump_image(repo, "maven", env["FROM"], env["TO"])
        set_tool(repo, "jdk", env["JDK"])
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: jenkins-maven-3-9
    description: Move Jenkins builds to Maven 3.9 and JDK 21

    targets:
      orgs: [acme]
      include:
        files: [Jenkinsfile]

    change:
      steps:
        # Moves the maven:3.8 agent image to 3.9 (keeping the JDK variant), and
        # points `tools { jdk ... }` at the JDK 21 installation.
        - name: jenkins-agent-image
          starlark: ./change.star
          timeout: 1m
      env:
        FROM: "3.8"
        TO: "3.9"
        JDK: jdk21

    pr:
      title: "ci: maven 3.9 and JDK 21 in Jenkins"
      body: |
        Moves {{ .Repo.Name }}'s Jenkins pipeline to Maven 3.9 and JDK 21.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/jenkins-agent-image)
