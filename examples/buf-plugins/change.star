load("@fleet//buf.star", "set_plugin_version")

def change(repo):
    set_plugin_version(repo, "buf.build/protocolbuffers/go", env["PROTOC_GEN_GO"])
    set_plugin_version(repo, "buf.build/grpc/go", env["GRPC_GO"])
