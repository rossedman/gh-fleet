package app

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/rossedman/gh-fleet/internal/ghapi"
)

func TestApplyReviewersFrom(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	f := h.campaign(`apiVersion: fleet.gh/v1
name: owners
targets:
  orgs: [acme]
  include:
    topics: [service]
change:
  steps:
    - starlark: ./change.star
pr:
  title: notice
  reviewers: [octocat]
  reviewersFrom: [codeowners, catalog]
`, map[string]string{"change.star": `def change(repo):
    repo.write(".github/CODEOWNERS", "* @acme/platform\nNOTICE @OctoCat @acme/legal\n*.md docs@example.com\n")
    repo.write("catalog-info.yaml", "kind: Component\nspec:\n  owner: group:default/payments\n")
    repo.write("NOTICE", "x\n")
`})
	ctx := context.Background()
	h.gh.Errors["RequestReviewers:acme/web"] = errors.New("422 not a collaborator")
	err := h.app.Apply(ctx, ApplyOptions{Common: Common{File: f, Yes: true}})
	// acme/web fails on the campaign's own reviewer, which stays strict.
	if err == nil || !strings.Contains(h.errOut.String(), "failed   acme/web") {
		t.Fatalf("err = %v\n%s", err, h.errOut)
	}
	// Owners of every changed file, deduplicated against reviewers and
	// each other, after the campaign's own.
	if got := strings.Join(h.gh.Reviewers["acme/api#1"], ","); got != "octocat,acme/platform,acme/legal,acme/payments" {
		t.Fatalf("reviewers = %q", got)
	}
	log, _ := os.ReadFile(h.app.Store.LogPath("owners", "acme/api"))
	if !strings.Contains(string(log), "## reviewers from codeowners, catalog: acme/platform, acme/legal, acme/payments") {
		t.Errorf("log:\n%s", log)
	}

	// An owner GitHub rejects is skipped without failing the repo.
	h2 := newHarness(t)
	f2 := h2.campaign(strings.Replace(string(must(os.ReadFile(f))), "  reviewers: [octocat]\n", "", 1), map[string]string{"change.star": string(must(os.ReadFile(strings.Replace(f, "fleet.yaml", "change.star", 1))))})
	h2.gh.Errors["RequestReviewers:acme/web"] = errors.New("422 team not found")
	if err := h2.app.Apply(ctx, ApplyOptions{Common: Common{File: f2, Yes: true}}); err != nil {
		t.Fatalf("owner errors failed the run: %v\n%s", err, h2.errOut)
	}
	if prs := h2.gh.OpenPRs(); len(prs) != 2 || prs[0].State != ghapi.StateOpen {
		t.Fatalf("prs = %+v", prs)
	}
	log, _ = os.ReadFile(h2.app.Store.LogPath("owners", "acme/web"))
	if strings.Count(string(log), "skipped: 422 team not found") != 4 {
		t.Errorf("log:\n%s", log)
	}
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
