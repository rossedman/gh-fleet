package ui

import (
	"bytes"
	"strings"
	"testing"
)

func TestStylesAndColor(t *testing.T) {
	if Plain().Bad.Render("x") != "x" {
		t.Fatal("plain styles add codes")
	}
	_ = Color().Header.Render("x")
	if For(false).OK.Render("y") != "y" {
		t.Fatal("For(false)")
	}
	t.Setenv("NO_COLOR", "1")
	if ColorEnabled(true) {
		t.Fatal("NO_COLOR ignored")
	}
	t.Setenv("NO_COLOR", "")
	t.Setenv("CLICOLOR", "0")
	if ColorEnabled(true) {
		t.Fatal("CLICOLOR=0 ignored")
	}
	t.Setenv("CLICOLOR", "")
	t.Setenv("CLICOLOR_FORCE", "1")
	if !ColorEnabled(false) {
		t.Fatal("CLICOLOR_FORCE ignored")
	}
	t.Setenv("CLICOLOR_FORCE", "")
	if !ColorEnabled(true) || ColorEnabled(false) {
		t.Fatal("tty detection")
	}
	t.Setenv("GH_FLEET_ACCESSIBLE", "1")
	if !Accessible() {
		t.Fatal("accessible")
	}
	_ = Theme().Theme(true)
}

func TestConfirmAccessible(t *testing.T) {
	t.Setenv("GH_FLEET_ACCESSIBLE", "1")
	var out bytes.Buffer
	ok, err := Confirm(strings.NewReader("y\n"), &out, "Proceed?")
	if err != nil || !ok {
		t.Fatalf("confirm = %v, %v (%s)", ok, err, out.String())
	}
	ok, err = Confirm(strings.NewReader("n\n"), &out, "Proceed?")
	if err != nil || ok {
		t.Fatalf("confirm = %v, %v", ok, err)
	}
}

func TestLogger(t *testing.T) {
	t.Parallel()
	var b bytes.Buffer
	NewLogger(&b, true).Debug("hello", "repo", "a/b")
	NewLogger(&b, false).Debug("hidden")
	if !strings.Contains(b.String(), "hello") || strings.Contains(b.String(), "hidden") {
		t.Fatalf("log = %q", b.String())
	}
}
