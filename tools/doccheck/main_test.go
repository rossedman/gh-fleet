package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	t.Parallel()
	var out, errOut bytes.Buffer
	if code := run([]string{"./testdata/good"}, &out, &errOut); code != 0 {
		t.Fatalf("good: %d %s", code, out.String())
	}
	if code := run([]string{"./testdata/gen"}, &out, &errOut); code != 0 {
		t.Fatalf("generated files must be skipped: %s", out.String())
	}
	out.Reset()
	if code := run([]string{"./testdata/bad"}, &out, &errOut); code != 1 {
		t.Fatal("bad package passed")
	}
	for _, want := range []string{"bad.go:5: T.M has no doc comment", "bad.go:7: init has no doc comment", "bad.go:9: F has no doc comment", "package bad has no package comment"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("missing %q in:\n%s", want, out.String())
		}
	}
	if code := run([]string{"./nonexistent/..."}, &out, &errOut); code != 1 {
		t.Fatal("missing dir passed")
	}
	if recvName(nil) != "?" {
		t.Fatal("recvName fallback")
	}
}
