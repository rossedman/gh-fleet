package app

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestParseRepoList(t *testing.T) {
	t.Parallel()
	in := "# services\nacme/api\n\n  acme/web  # the site\r\n" +
		"https://github.com/acme/docs.git\ngit@github.com:acme/old.git\n" +
		"acme/cli\tA command line tool\tpublic\t2d ago\nacme/lib,extra\ngithub.com/acme/sdk/\n"
	got, err := ParseRepoList(strings.NewReader(in), "list")
	want := []string{"acme/api", "acme/web", "acme/docs", "acme/old", "acme/cli", "acme/lib", "acme/sdk"}
	if err != nil || !slices.Equal(got, want) {
		t.Fatalf("got %v, %v", got, err)
	}
	for _, bad := range []string{"acme\n", "acme/a/b\n", "/api\n"} {
		if _, err := ParseRepoList(strings.NewReader("acme/ok\n"+bad), "list"); err == nil || !strings.Contains(err.Error(), "list:2:") {
			t.Errorf("%q: err = %v", bad, err)
		}
	}
	if _, err := ParseRepoList(strings.NewReader(strings.Repeat("x", 70000)), "big"); err == nil {
		t.Error("over-long line accepted")
	}
}

func TestReadRepoFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	a := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(a, []byte("acme/api\nacme/web\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := ReadRepoFiles([]string{a, "-"}, strings.NewReader("ACME/API\nacme/docs\n"))
	if err != nil || !slices.Equal(got, []string{"acme/api", "acme/web", "acme/docs"}) {
		t.Fatalf("got %v, %v", got, err)
	}
	if _, err := ReadRepoFiles([]string{filepath.Join(dir, "missing")}, nil); err == nil {
		t.Error("missing file accepted")
	}
	if _, err := ReadRepoFiles([]string{"-"}, strings.NewReader("nope\n")); err == nil || !strings.Contains(err.Error(), "stdin:1:") {
		t.Errorf("bad stdin: %v", err)
	}
}

func TestExclude(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	f := h.campaign(basic, files)
	ctx := context.Background()
	if err := h.app.Targets(ctx, Common{File: f, Exclude: []string{"ACME/web"}}); err != nil {
		t.Fatal(err)
	}
	if out := h.out.String(); !strings.Contains(out, "acme/api") || strings.Contains(out, "acme/web") {
		t.Fatalf("targets:\n%s", out)
	}
	// A repo both listed and excluded is left out, not an error.
	h.reset()
	if err := h.app.Targets(ctx, Common{File: f, Repos: []string{"acme/api", "acme/web"}, Exclude: []string{"acme/web"}, JQ: "[.targets[].repo | .owner + \"/\" + .name]"}); err != nil {
		t.Fatal(err)
	}
	if out := h.out.String(); strings.Contains(out, "acme/web") || !strings.Contains(out, "acme/api") {
		t.Fatalf("listed and excluded:\n%s", out)
	}

	// Status leaves out an excluded repo's PR from earlier runs too.
	if err := h.app.Apply(ctx, ApplyOptions{Common: Common{File: f, Yes: true}}); err != nil {
		t.Fatal(err)
	}
	h.reset()
	if err := h.app.Status(ctx, StatusOptions{Common: Common{File: f, Exclude: []string{"acme/web"}}}); err != nil {
		t.Fatal(err)
	}
	if out := h.out.String(); !strings.Contains(out, "acme/api") || strings.Contains(out, "acme/web") {
		t.Fatalf("status:\n%s", out)
	}
}
