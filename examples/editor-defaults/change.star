load("@fleet//editorconfig.star", "set")
load("@fleet//gitattributes.star", gitattributes = "set")

def change(repo):
    set(repo, "", "root", True)
    set(repo, "*", "end_of_line", "lf")
    set(repo, "*", "insert_final_newline", True)
    set(repo, "Makefile", "indent_style", "tab")
    gitattributes(repo, "*", "text=auto")
    gitattributes(repo, "*.sh", "text eol=lf")
