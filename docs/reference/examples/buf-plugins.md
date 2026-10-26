---
title: buf-plugins
---

# buf-plugins

Move protobuf code generation to the current Go plugins.

**Uses:** [`buf.set_plugin_version`](../starlark/buf.md#set_plugin_version)

Start a campaign from it, then point it at your organization:

```sh
gh fleet init --from buf-plugins --org your-org
```

## Campaign

=== "change.star"

    ```python
    load("@fleet//buf.star", "set_plugin_version")

    def change(repo):
        set_plugin_version(repo, "buf.build/protocolbuffers/go", env["PROTOC_GEN_GO"])
        set_plugin_version(repo, "buf.build/grpc/go", env["GRPC_GO"])
    ```

=== "fleet.yaml"

    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/rossedman/gh-fleet/main/internal/campaign/campaign.schema.json
    apiVersion: fleet.gh/v1
    name: buf-go-plugins
    description: Move protobuf code generation to the current Go plugins

    targets:
      orgs: [acme]
      include:
        files: [buf.gen.yaml]

    change:
      steps:
        # Moves the protocolbuffers/go and grpc/go remote plugins in every
        # buf.gen.yaml. Regenerate the code with `buf generate` before merging.
        - name: buf-plugins
          starlark: ./change.star
          timeout: 1m
      env:
        PROTOC_GEN_GO: v1.35.1
        GRPC_GO: v1.5.1

    pr:
      title: "build: protoc-gen-go v1.35.1 and grpc-go v1.5.1"
      body: |
        Moves {{ .Repo.Name }}'s Buf plugins to protoc-gen-go v1.35.1 and
        protoc-gen-go-grpc v1.5.1.

        Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
      labels: [fleet, dependencies]
    ```

[Source on GitHub](https://github.com/rossedman/gh-fleet/tree/main/examples/buf-plugins)
