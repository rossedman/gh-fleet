package owners

import (
	"strings"
	"testing"
)

// codeowners is GitHub's documented example file, cut down.
const codeowners = `# Default owners for everything
*       @global-owner1 @global-owner2

# Order matters: the last match wins.
*.js    @js-owner #This is an inline comment.
*.go docs@example.com
/build/logs/ @doctocat
docs/*  docs@example.com @docs-team
apps/ @octocat
/docs/ @acme/doc-writers
**/logs @acme/loggers
/scripts/ @doctocat @octocat
/apps/github
!negated @nobody
[abc].txt @nobody
`

func TestCodeowners(t *testing.T) {
	t.Parallel()
	c := ParseCodeowners(codeowners)
	for _, tt := range []struct {
		paths []string
		want  string
	}{
		{[]string{"README.md"}, "global-owner1,global-owner2"},
		{[]string{"web/app.js"}, "js-owner"},
		{[]string{"main.go"}, ""},
		{[]string{"build/logs/out.txt"}, "acme/loggers"},
		{[]string{"x/build/logs/out.txt"}, "acme/loggers"},
		{[]string{"docs/getting-started.md"}, "acme/doc-writers"},
		{[]string{"src/docs/a.md"}, "global-owner1,global-owner2"},
		{[]string{"src/docs/deep/a.md"}, "global-owner1,global-owner2"},
		{[]string{"apps/web/x"}, "octocat"},
		{[]string{"apps/github/x"}, ""},
		{[]string{"scripts/run.sh", "app.js", "main.js"}, "doctocat,octocat,js-owner"},
		{nil, ""},
	} {
		if got := strings.Join(c.Owners(tt.paths), ","); got != tt.want {
			t.Errorf("%v: got %q, want %q", tt.paths, got, tt.want)
		}
	}
	// docs/* is anchored and owns only files directly in docs.
	d := ParseCodeowners("docs/* @docs-team\n")
	if got := strings.Join(d.Owners([]string{"docs/a.md", "docs/deep/b.md", "src/docs/c.md"}), ","); got != "docs-team" {
		t.Errorf("docs/* = %q", got)
	}
	if d.Owners([]string{"docs/deep/b.md"}) != nil || d.Owners([]string{"src/docs/c.md"}) != nil {
		t.Error("docs/* matched too much")
	}
	if got := ParseCodeowners("").Owners([]string{"a"}); got != nil {
		t.Errorf("empty file: %v", got)
	}
}

func TestCatalogOwners(t *testing.T) {
	t.Parallel()
	catalog := "kind: Component\nspec:\n  owner: group:default/payments\n---\nkind: API\nspec:\n  owner: payments\n---\nkind: Resource\nspec:\n  owner: user:jdoe\n---\nkind: System\nspec:\n  owner: system:billing\n---\nkind: Location\n"
	if got := strings.Join(CatalogOwners([]byte(catalog), "acme"), ","); got != "acme/payments,jdoe" {
		t.Errorf("owners = %q", got)
	}
	if got := CatalogOwners([]byte("- not\n- an entity\n"), "acme"); len(got) != 0 {
		t.Errorf("list = %v", got)
	}
}
