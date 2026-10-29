package app

import (
	"context"
	"strings"
	"testing"
)

func TestTargetNotices(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	f := h.campaign(strings.Replace(basic, "    topics: [service]", "    topics: [service]\n    repos: [acme/docs]\n    names: [api, other/web]", 1), files)
	ctx := context.Background()
	wantWarn := `warning: include.names "other/web" names an owner that isn't in orgs`
	wantNote := "note: include.repos adds its repos to what include.names and include.topics select in acme"
	for name, run := range map[string]func(Common) error{
		"targets": func(c Common) error { return h.app.Targets(ctx, c) },
		"plan":    func(c Common) error { return h.app.Plan(ctx, PlanOptions{Common: c}) },
		"apply":   func(c Common) error { return h.app.Apply(ctx, ApplyOptions{Common: c}) },
	} {
		h.reset()
		_ = run(Common{File: f, Yes: true})
		if got := h.errOut.String(); strings.Count(got, wantWarn) != 1 || strings.Count(got, wantNote) != 1 {
			t.Errorf("%s stderr:\n%s", name, got)
		}
	}
	// Machine output with nobody watching stderr stays quiet.
	h.reset()
	if err := h.app.Targets(ctx, Common{File: f, JSON: true}); err != nil {
		t.Fatal(err)
	}
	if got := h.errOut.String(); strings.Contains(got, "warning:") || strings.Contains(got, "note:") {
		t.Errorf("--json stderr:\n%s", got)
	}
}
