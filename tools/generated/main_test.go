package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	t.Parallel()
	var out, errOut bytes.Buffer
	if code := run([]string{"../doccheck/..."}, &out, &errOut); code != 0 {
		t.Fatalf("code %d: %s", code, errOut.String())
	}
	// testdata is skipped by the walker, so nothing is generated here.
	if out.String() != "" {
		t.Fatalf("out = %q", out.String())
	}
	if code := run([]string{"../doccheck/testdata/gen"}, &out, &errOut); code != 0 || !strings.HasSuffix(strings.TrimSpace(out.String()), "testdata/gen/gen.go") {
		t.Fatalf("out = %q", out.String())
	}
	if code := run([]string{"./nope"}, &out, &errOut); code != 1 {
		t.Fatal("missing dir passed")
	}
}
