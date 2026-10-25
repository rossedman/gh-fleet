package app

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/rossedman/gh-fleet/internal/campaign"
	"github.com/rossedman/gh-fleet/internal/ui"
)

// InitOptions configures Init.
type InitOptions struct {
	Name  string
	File  string
	Org   string
	Force bool
}

// nonName matches runs of characters not allowed in a campaign name.
var nonName = regexp.MustCompile(`[^a-z0-9]+`)

// SlugName turns free text into a valid campaign name.
func SlugName(s string) string {
	s = strings.Trim(nonName.ReplaceAllString(strings.ToLower(s), "-"), "-")
	if len(s) > 63 {
		s = strings.TrimRight(s[:63], "-")
	}
	if s == "" {
		return "my-campaign"
	}
	return s
}

// Init writes a commented campaign file.
func (a *App) Init(o InitOptions) error {
	file := o.File
	if file == "" {
		file = campaign.DefaultFile
	}
	name := o.Name
	if name == "" {
		abs, err := filepath.Abs(filepath.Dir(file))
		if err != nil {
			return err
		}
		name = filepath.Base(abs)
	}
	name = SlugName(name)
	if _, err := os.Stat(file); err == nil && !o.Force {
		return fmt.Errorf("%s already exists; pass --force to overwrite it", file)
	} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	b, err := campaign.Scaffold(name, o.Org)
	if err != nil {
		return err
	}
	if dir := filepath.Dir(file); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	if err := os.WriteFile(file, b, 0o644); err != nil { //nolint:gosec // campaign files are meant to be shared
		return err
	}
	s := ui.For(a.Color)
	fmt.Fprintf(a.Out, "%s %s for campaign %s\n", s.OK.Render("Wrote"), file, s.Bold.Render(name))
	fmt.Fprintf(a.Out, "%s edit the targets and change, then run %s\n", s.Dim.Render("Next:"), s.Brand.Render("gh fleet targets"))
	return nil
}
