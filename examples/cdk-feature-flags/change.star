load("@fleet//cdk.star", "set_context")

def change(repo):
    set_context(repo, "@aws-cdk/aws-s3:createDefaultLoggingPolicy", True)
