package state

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestStoreRoundTrip(t *testing.T) {
	t.Parallel()
	now := time.Unix(1000, 0).UTC()
	s := &Store{Dir: filepath.Join(t.TempDir(), "state"), Now: func() time.Time { return now }}
	if s.Exists("x") {
		t.Fatal("exists before save")
	}
	c, err := s.Load("x")
	if err != nil || len(c.Repos) != 0 {
		t.Fatalf("empty load: %v", err)
	}
	c.File = "/tmp/fleet.yaml"
	c.Set(RepoState{Repo: "a/b", Status: Failed, LastError: "boom", Diff: "d", Output: "o"}, s.Time())
	now = now.Add(time.Hour)
	c.Set(RepoState{Repo: "a/b", Status: PROpen, PRNumber: 3}, s.Time())
	c.Set(RepoState{Repo: "a/a", Status: Noop}, s.Time())
	if err := s.Save(c); err != nil {
		t.Fatal(err)
	}
	got, err := s.Load("x")
	if err != nil {
		t.Fatal(err)
	}
	rs := got.Get("a/b")
	if rs.Status != PROpen || rs.PRNumber != 3 || !rs.CreatedAt.Equal(time.Unix(1000, 0)) || !rs.UpdatedAt.Equal(now) || rs.Diff != "" {
		t.Fatalf("state = %+v", rs)
	}
	if sorted := got.Sorted(); sorted[0].Repo != "a/a" || got.File != "/tmp/fleet.yaml" {
		t.Fatalf("sorted = %+v", sorted)
	}
	if got.Get("nope") != nil {
		t.Fatal("missing repo returned state")
	}
	if !s.Exists("x") || s.LogPath("x", "a/b") != filepath.Join(s.Dir, "x", "logs", "a__b.log") {
		t.Fatal("paths")
	}

	sums, err := s.List()
	if err != nil || len(sums) != 1 || sums[0].Name != "x" || sums[0].Counts[PROpen] != 1 || sums[0].Counts[Noop] != 1 {
		t.Fatalf("list = %+v, %v", sums, err)
	}
}

func TestStoreErrors(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s := &Store{Dir: dir}
	if err := os.WriteFile(filepath.Join(dir, "bad.json"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Load("bad"); err == nil {
		t.Fatal("expected parse error")
	}
	if _, err := s.List(); err == nil {
		t.Fatal("expected list error")
	}
	if err := os.WriteFile(filepath.Join(dir, "null.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if c, err := s.Load("null"); err != nil || c.Repos == nil {
		t.Fatalf("null repos: %v", err)
	}
	if err := os.Mkdir(filepath.Join(dir, "unreadable.json"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Load("unreadable"); err == nil {
		t.Fatal("expected read error")
	}
	if sums, err := (&Store{Dir: filepath.Join(dir, "none")}).List(); err != nil || sums != nil {
		t.Fatal("missing dir should list nothing")
	}
	// Windows reports a file used as a directory as missing, which lists
	// nothing.
	if _, err := (&Store{Dir: filepath.Join(dir, "null.json")}).List(); err == nil && runtime.GOOS != "windows" {
		t.Fatal("expected error listing a file")
	}
	if err := (&Store{Dir: filepath.Join(dir, "null.json", "x")}).Save(&Campaign{Name: "x"}); err == nil {
		t.Fatal("expected save error")
	}
	ro := filepath.Join(dir, "ro")
	if err := os.Mkdir(ro, 0o500); err != nil {
		t.Fatal(err)
	}
	// Root ignores the mode, and Windows has no read-only directories.
	if os.Geteuid() != 0 && runtime.GOOS != "windows" {
		if err := (&Store{Dir: ro}).Save(&Campaign{Name: "x"}); err == nil {
			t.Fatal("expected temp file error")
		}
	}
	if (&Store{}).Time().IsZero() {
		t.Fatal("default clock")
	}
}

func TestDefaultDir(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/xdg")
	if d, err := DefaultDir(); err != nil || d != filepath.Join("/xdg", "gh-fleet") {
		t.Fatalf("dir = %q, %v", d, err)
	}
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("HOME", "/home/x")
	t.Setenv("USERPROFILE", "/home/x")
	if d, err := DefaultDir(); err != nil || d != filepath.Join("/home/x", ".local", "state", "gh-fleet") {
		t.Fatalf("dir = %q, %v", d, err)
	}
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	if _, err := DefaultDir(); err == nil {
		t.Fatal("expected error without HOME")
	}
}
