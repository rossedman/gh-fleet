load("@fleet//codeql.star", "add_language", "add_workflow", "languages", "set_version")

# GitHub's name for a repo's main language, mapped to CodeQL's.
CODEQL = {
    "C#": "csharp",
    "C++": "c-cpp",
    "Go": "go",
    "Java": "java-kotlin",
    "JavaScript": "javascript-typescript",
    "Kotlin": "java-kotlin",
    "Python": "python",
    "Ruby": "ruby",
    "Swift": "swift",
    "TypeScript": "javascript-typescript",
}

def change(repo):
    main = CODEQL.get(repo.language)
    added = add_workflow(repo, ([main] if main else []) + ["actions"])
    if not added:
        add_language(repo, "actions")
        set_version(repo, "v4")
    repo.set_var("added", added)
    repo.set_var("languages", ", ".join(languages(repo)))
