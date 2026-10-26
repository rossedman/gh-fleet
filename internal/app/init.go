package app

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/rossedman/gh-fleet/examples"
	"github.com/rossedman/gh-fleet/internal/campaign"
	"github.com/rossedman/gh-fleet/internal/render"
	"github.com/rossedman/gh-fleet/internal/ui"
)

// InitOptions configures Init.
type InitOptions struct {
	Name  string
	File  string
	Org   string
	Force bool
	// From copies this example campaign, with its scripts and test cases,
	// instead of writing a blank one.
	From string
	// List prints the examples instead of writing anything.
	List bool
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

// Init writes a commented campaign file, or copies an example campaign.
func (a *App) Init(o InitOptions) error {
	if o.List {
		return a.listExamples()
	}
	if o.From != "" {
		return a.initFrom(o)
	}
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
	fmt.Fprintf(a.Err, "%s %s for campaign %s\n", s.OK.Render("Wrote"), file, s.Bold.Render(name))
	fmt.Fprintf(a.Err, "%s edit the targets and change, then run %s\n", s.Dim.Render("Next:"), s.Brand.Render("gh fleet targets"))
	return nil
}

// listExamples prints each example and its description.
func (a *App) listExamples() error {
	names := examples.Names()
	rows := make([][]string, len(names))
	for i, n := range names {
		b, err := fs.ReadFile(examples.FS, n+"/fleet.yaml")
		if err != nil {
			return err
		}
		rows[i] = []string{n, campaignDescription(b)}
	}
	p := render.Printer{Out: a.Out, TTY: a.TTY, Styles: ui.For(a.Color)}
	p.Table([]string{"EXAMPLE", "WHAT IT DOES"}, rows, nil)
	fmt.Fprintf(a.Err, "\ngh fleet init --from <example> copies one, with its test cases.\n")
	return nil
}

// descriptionLine matches a campaign file's top-level description.
var descriptionLine = regexp.MustCompile(`(?m)^description:\s*(.*)$`)

// campaignDescription returns a campaign file's description line.
func campaignDescription(b []byte) string {
	if m := descriptionLine.FindSubmatch(b); m != nil {
		return strings.Trim(string(m[1]), `"'`)
	}
	return ""
}

// initFrom copies an example campaign next to file, renaming it and
// pointing it at an org when asked.
func (a *App) initFrom(o InitOptions) error {
	src := strings.TrimSuffix(strings.TrimPrefix(o.From, "examples/"), "/")
	if _, err := fs.Stat(examples.FS, src+"/fleet.yaml"); err != nil {
		var near []string
		for _, n := range examples.Names() {
			if strings.Contains(n, src) || strings.Contains(src, n) || editDistance(n, src) <= 2 {
				near = append(near, n)
			}
		}
		if len(near) > 0 {
			return fmt.Errorf("no example %s; close names: %s", src, strings.Join(near, ", "))
		}
		return fmt.Errorf("no example %s; gh fleet init --list shows them", src)
	}
	file := o.File
	if file == "" {
		file = campaign.DefaultFile
	}
	dir := filepath.Dir(file)

	// Every path is checked before anything is written, so a refusal
	// leaves the directory as it was.
	type out struct {
		path string
		data []byte
	}
	var files []out
	err := fs.WalkDir(examples.FS, src, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := fs.ReadFile(examples.FS, p)
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(p, src+"/")
		target := filepath.Join(dir, filepath.FromSlash(rel))
		if rel == "fleet.yaml" {
			target = file
			b = customize(b, o.Name, o.Org)
		}
		files = append(files, out{target, b})
		return nil
	})
	if err != nil {
		return err
	}
	if !o.Force {
		for _, f := range files {
			if _, err := os.Stat(f.path); err == nil {
				return fmt.Errorf("%s already exists; pass --force to overwrite it", f.path)
			}
		}
	}
	for _, f := range files {
		if err := os.MkdirAll(filepath.Dir(f.path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(f.path, f.data, 0o644); err != nil { //nolint:gosec // campaign files are meant to be shared
			return err
		}
	}
	s := ui.For(a.Color)
	fmt.Fprintf(a.Err, "%s %d files from example %s into %s\n", s.OK.Render("Copied"), len(files), s.Bold.Render(src), dir)
	fmt.Fprintf(a.Err, "%s %s checks it against its test cases; edit the targets, then run %s\n", s.Dim.Render("Next:"), s.Brand.Render("gh fleet test"), s.Brand.Render("gh fleet targets"))
	return nil
}

// nameLine and orgsLine match the lines customize rewrites.
var (
	nameLine = regexp.MustCompile(`(?m)^name:.*$`)
	orgsLine = regexp.MustCompile(`(?m)^(\s+orgs:\s*)\[[^\]]*\](.*)$`)
)

// customize sets an example's name and target org.
func customize(b []byte, name, org string) []byte {
	if name != "" {
		b = nameLine.ReplaceAll(b, []byte("name: "+SlugName(name)))
	}
	if org != "" {
		b = orgsLine.ReplaceAll(b, []byte("${1}["+org+"]${2}"))
	}
	return b
}
