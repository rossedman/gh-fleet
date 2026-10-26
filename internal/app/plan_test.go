package app

import (
	"context"
	"strings"
	"testing"

	"github.com/rossedman/gh-fleet/internal/state"
)

func TestGroupDiffs(t *testing.T) {
	t.Parallel()
	a := "diff --git a/x b/x\nindex 1111111..2222222 100644\n--- a/x\n+++ b/x\n@@ -1,2 +1,3 @@ func\n a\n+b\n"
	a2 := strings.NewReplacer("1111111..2222222", "3333333..4444444", "@@ -1,2 +1,3 @@", "@@ -10,2 +10,3 @@").Replace(a)
	b := strings.Replace(a, "+b", "+c", 1)
	results := []state.RepoState{
		{Repo: "acme/one", Status: state.Changed, Diff: b},
		{Repo: "acme/two", Status: state.Changed, Diff: a},
		{Repo: "acme/three", Status: state.Changed, Diff: a2},
		{Repo: "acme/noop", Status: state.Noop},
		{Repo: "acme/empty", Status: state.Changed},
	}
	groups := groupDiffs(results)
	if len(groups) != 2 || strings.Join(groups[0].repos, ",") != "acme/two,acme/three" || groups[0].diff != a || strings.Join(groups[1].repos, ",") != "acme/one" {
		t.Fatalf("groups = %+v", groups)
	}
	if h := groupHeader([]string{"acme/a"}); h != "acme/a" {
		t.Errorf("one = %q", h)
	}
	if h := groupHeader([]string{"a", "b"}); h != "2 repos: a, b" {
		t.Errorf("two = %q", h)
	}
	many := strings.Split("a b c d e f g h i j k l", " ")
	if h := groupHeader(many); h != "12 repos: a, b, c, d, e, f, g, h, i, j and 2 more" {
		t.Errorf("many = %q", h)
	}
}

func TestPlanGroupsIdenticalDiffs(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	f := h.campaign(`apiVersion: fleet.gh/v1
name: add-file
targets:
  orgs: [acme]
  include:
    topics: [service]
change:
  steps:
    - starlark: ./change.star
pr:
  title: add a file
`, map[string]string{"change.star": "def change(repo):\n    repo.write(\"NOTICE\", \"managed\\n\")\n"})
	ctx := context.Background()
	if err := h.app.Plan(ctx, PlanOptions{Common: Common{File: f}}); err != nil {
		t.Fatal(err)
	}
	out := h.out.String()
	if strings.Count(out, "+managed") != 1 || !strings.Contains(out, "── 2 repos: acme/api, acme/web") {
		t.Fatalf("grouped plan:\n%s", out)
	}
	if !strings.Contains(h.errOut.String(), "changed  acme/api") {
		t.Errorf("no progress line for a changed repo:\n%s", h.errOut)
	}
	h.reset()
	if err := h.app.Plan(ctx, PlanOptions{Common: Common{File: f}, Ungrouped: true}); err != nil {
		t.Fatal(err)
	}
	if strings.Count(h.out.String(), "+managed") != 2 || !strings.Contains(h.out.String(), "── acme/web") {
		t.Fatalf("ungrouped plan:\n%s", h.out)
	}
}
