package examples_test

import "testing"

func TestBackstageExamples(t *testing.T) {
	t.Parallel()
	dir := scratch(t, map[string]string{"mkdocs.yml": "site_name: web\n"})
	vars := runTwice(t, "backstage-onboarding", dir)
	if vars["owner"] != "group:platform" {
		t.Errorf("owner = %v", vars["owner"])
	}
	contains(t, dir, "catalog-info.yaml", "name: web\n", "github.com/project-slug: acme/web", "backstage.io/techdocs-ref: dir:.", "owner: group:platform")

	dir = scratch(t, map[string]string{"catalog-info.yaml": "kind: Component\nmetadata:\n  name: api\nspec:\n  owner: group:payments\n---\nkind: Component\nmetadata:\n  name: search\nspec:\n  owner: discovery\n"})
	vars = runTwice(t, "backstage-team-rename", dir)
	if vars["entities"] != "api" {
		t.Errorf("entities = %v", vars["entities"])
	}
	contains(t, dir, "catalog-info.yaml", "owner: group:billing\n", "owner: discovery\n")
}
