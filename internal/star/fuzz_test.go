package star

import (
	"path/filepath"
	"strings"
	"testing"
)

func FuzzLocalPath(f *testing.F) {
	for _, s := range []string{"a.txt", "dir/b", "../x", "/etc/passwd", "", ".git/config", "a/../../b", `a\..\b`, "./.git", "x/.git/y", ".gitignore", "con", "a//b"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, p string) {
		local, err := localPath("fuzz", p)
		if err != nil {
			return
		}
		// Anything accepted stays inside the repo and out of .git.
		if !filepath.IsLocal(local) {
			t.Fatalf("%q accepted as %q, which is not local", p, local)
		}
		clean := filepath.ToSlash(filepath.Clean(local))
		if clean == ".git" || strings.HasPrefix(clean, ".git/") {
			t.Fatalf("%q reaches .git", p)
		}
		root := filepath.FromSlash("/repo")
		joined := filepath.Join(root, local)
		if joined != root && !strings.HasPrefix(joined, root+string(filepath.Separator)) {
			t.Fatalf("%q escapes to %q", p, joined)
		}
	})
}
