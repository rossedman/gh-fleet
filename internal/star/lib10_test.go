package star

import (
	"strings"
	"testing"
)

func TestLibBackstage(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]libCase{
		"filters and tag forms": {
			repo: map[string]string{
				"catalog-info.yaml":    "---\n# billing\nkind: component\nmetadata:\n  name: api\n  tags:\n    - java  # lang\n    - \"pci\"\nspec:\n  owner: user:jdoe\n\n---\nkind: Component\nmetadata:\n  name: web\n  tags: ['ui']\nspec:\n  owner: payments\n",
				"svc/catalog-info.yml": "kind: Group\nmetadata:\n  name: payments\n",
			},
			script: `load("@fleet//backstage.star", "add_tag", "entities", "set_annotation", "set_owner")
def change(repo):
    repo.set_var("tag", add_tag(repo, "pci", kind = "Component"))
    repo.set_var("ui", add_tag(repo, "ui", name = "web"))
    repo.set_var("owner", set_owner(repo, "group:billing", from_owner = "group:default/payments"))
    set_annotation(repo, "backstage.io/techdocs-ref", "dir:.", name = "api")
    set_annotation(repo, "backstage.io/techdocs-ref", "url:x", only_missing = True, kind = "COMPONENT")
    repo.set_var("names", [e["name"] for e in entities(repo)])
`,
			want: map[string]string{
				"catalog-info.yaml":    "---\n# billing\nkind: component\nmetadata:\n  name: api\n  tags:\n    - java  # lang\n    - \"pci\"\n  annotations:\n    backstage.io/techdocs-ref: dir:.\nspec:\n  owner: user:jdoe\n\n---\nkind: Component\nmetadata:\n  name: web\n  tags: ['ui', pci]\n  annotations:\n    backstage.io/techdocs-ref: url:x\nspec:\n  owner: group:billing\n",
				"svc/catalog-info.yml": "kind: Group\nmetadata:\n  name: payments\n",
			},
			vars: map[string]string{"tag": "[catalog-info.yaml]", "ui": "[]", "owner": "[catalog-info.yaml]", "names": "[api web payments]"},
		},
		"no metadata and existing file": {
			repo: map[string]string{"catalog-info.yaml": "kind: Location\nspec:\n  targets: [./a.yaml]\n"},
			script: `load("@fleet//backstage.star", "add_tag", "ensure")
def change(repo):
    add_tag(repo, "infra")
    repo.set_var("ensured", ensure(repo, "x"))
`,
			want: map[string]string{"catalog-info.yaml": "kind: Location\nspec:\n  targets: [./a.yaml]\nmetadata:\n  tags: [infra]\n"},
			vars: map[string]string{"ensured": "false"},
		},
		"yaml quoting around colons": {
			script: `load("@fleet//yamlpath.star", "set_in")
def change(repo):
    repo.set_var("q", set_in("", "a", "x: y") + set_in("", "b", "x:") + set_in("", "c", "https://example.com/a"))
`,
			vars: map[string]string{"q": "a: \"x: y\"\nb: \"x:\"\nc: https://example.com/a\n"},
		},
	} {
		runLib(t, name, tc)
	}
	_, _, _, err := run(t, "load('@fleet//backstage.star', 'add_tag')\ndef change(repo):\n  add_tag(repo, 'Has Space')\n", nil, RepoInfo{})
	if err == nil || !strings.Contains(err.Error(), "isn't a valid Backstage tag") {
		t.Errorf("bad tag: %v", err)
	}
}
