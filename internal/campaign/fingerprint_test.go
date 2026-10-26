package campaign

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFingerprint(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, "camp")
	write := func(rel, content string) {
		t.Helper()
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("camp/fleet.yaml", "name: x\n")
	write("camp/change.star", `load("../shared/lib.star", "f")`+"\n")
	write("shared/lib.star", "def f(): pass\n")
	write("shared/run.sh", "echo hi\n")
	write("camp/.git/HEAD", "ref\n")
	c := &Campaign{Path: filepath.Join(dir, "fleet.yaml"), Change: Change{Steps: []Step{{Starlark: "change.star"}, {Run: "../shared/run.sh"}}}}

	fp := c.Fingerprint()
	if len(fp) != 64 || c.Fingerprint() != fp {
		t.Fatalf("fingerprint %q isn't stable", fp)
	}
	// Hidden directories don't count; files in the campaign directory do,
	// and so do scripts and loads outside it.
	write("camp/.git/HEAD", "other\n")
	if c.Fingerprint() != fp {
		t.Fatal("a hidden directory changed the fingerprint")
	}
	for _, rel := range []string{"camp/notes.md", "shared/lib.star", "shared/run.sh"} {
		before := c.Fingerprint()
		write(rel, "changed "+rel+"\n")
		if c.Fingerprint() == before {
			t.Errorf("changing %s kept the fingerprint", rel)
		}
	}

	// A remote load at a version or commit is fine; at a branch, the code
	// can change without the files changing, so there's no fingerprint.
	for ref, ok := range map[string]bool{"v1.2.0": true, "1.4": true, "0a1b2c3d": true, "main": false, "feature-x": false} {
		write("camp/change.star", `load("github.com/acme/lib/docker.star@`+ref+`", "f")`+"\n")
		if got := c.Fingerprint() != ""; got != ok {
			t.Errorf("load @%s: fingerprint %v, want %v", ref, got, ok)
		}
	}
	// An unpinned load reached through a local load counts too.
	write("camp/change.star", `load("../shared/lib.star", "f")`+"\n")
	write("shared/lib.star", `load("github.com/acme/lib/x.star@main", "g")`+"\n")
	if c.Fingerprint() != "" {
		t.Error("unpinned load outside the directory ignored")
	}
	write("shared/lib.star", "def f(): pass\n")

	// A directory too big to be a campaign has none.
	big := &Campaign{Path: filepath.Join(root, "big", "fleet.yaml")}
	write("big/fleet.yaml", strings.Repeat("x", maxFingerprintBytes+1))
	if big.Fingerprint() != "" {
		t.Error("oversized directory fingerprinted")
	}
	many := &Campaign{Path: filepath.Join(root, "many", "fleet.yaml")}
	for i := range maxFingerprintFiles + 1 {
		write(filepath.Join("many", fmt.Sprintf("f%04d", i)), "")
	}
	if many.Fingerprint() != "" {
		t.Error("directory with too many files fingerprinted")
	}
	// A missing directory has none.
	if (&Campaign{Path: filepath.Join(root, "nope", "fleet.yaml")}).Fingerprint() != "" {
		t.Error("missing directory fingerprinted")
	}
	// A load cycle ends.
	write("shared/a.star", `load("./b.star", "x")`)
	write("shared/b.star", `load("./a.star", "y")`)
	if got := starlarkFiles(filepath.Join(root, "shared/a.star"), map[string]bool{}); len(got) != 2 {
		t.Errorf("cycle: %v", got)
	}
	if got := starlarkFiles(filepath.Join(root, "shared/none.star"), map[string]bool{}); len(got) != 1 {
		t.Errorf("missing: %v", got)
	}
}
