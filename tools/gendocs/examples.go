package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"go.yaml.in/yaml/v3"
)

// Markers around the generated example entries in mkdocs.yml.
const (
	examplesNavBegin = "# gendocs:examples:begin"
	examplesNavEnd   = "# gendocs:examples:end"
)

// Example is one example campaign directory, read for its docs page.
type Example struct {
	// Name is the directory name, which is also the campaign name.
	Name        string
	Description string
	// Files are the campaign's own files: the Starlark scripts first, as
	// they do the work, then fleet.yaml, then everything else, by path.
	Files []exampleFile
	// Cases are the test cases under tests/, by name.
	Cases []exampleCase
	// Uses lists what the scripts load from the library, as
	// "module.star/name".
	Uses []string
	// Funcs holds the Uses entries that are documented functions, which
	// get a link to their section; constants link to the module page.
	Funcs map[string]bool
}

// exampleFile is one file of an example, with its path in the example.
type exampleFile struct {
	Path, Content string
}

// exampleCase is one test case: the repo before the change and, when the
// change edits it, after.
type exampleCase struct {
	Name          string
	Before, After []exampleFile
	// NoAfter is true when the case expects the change to leave the repo
	// alone.
	NoAfter bool
}

// readExamples reads every directory under dir that holds a fleet.yaml.
// used maps "module.star/function" to the examples that call it.
func readExamples(dir string, used map[string][]string) ([]Example, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*", "fleet.yaml"))
	if err != nil {
		return nil, err
	}
	uses := map[string][]string{}
	for fn, exs := range used {
		for _, e := range exs {
			uses[e] = append(uses[e], fn)
		}
	}
	var out []Example
	for _, f := range files {
		root := filepath.Dir(f)
		ex, err := readExample(root)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", root, err)
		}
		ex.Uses = uses[ex.Name]
		sort.Strings(ex.Uses)
		out = append(out, ex)
	}
	return out, nil
}

// readExample reads one example directory.
func readExample(root string) (Example, error) {
	ex := Example{Name: filepath.Base(root)}
	b, err := os.ReadFile(filepath.Join(root, "fleet.yaml"))
	if err != nil {
		return ex, err
	}
	var meta struct {
		Description string `yaml:"description"`
	}
	if err := yaml.Unmarshal(b, &meta); err != nil {
		return ex, err
	}
	ex.Description = meta.Description
	tests := filepath.Join(root, "tests")
	ex.Files, err = readTree(root, func(p string) bool { return p == tests })
	if err != nil {
		return ex, err
	}
	sort.SliceStable(ex.Files, func(i, j int) bool { return fileRank(ex.Files[i].Path) < fileRank(ex.Files[j].Path) })
	// Windows reports a file read as a directory as missing, so check what
	// tests is before listing it.
	st, err := os.Stat(tests)
	if os.IsNotExist(err) {
		return ex, nil
	}
	if err == nil && !st.IsDir() {
		err = fmt.Errorf("%s is not a directory", tests)
	}
	var cases []fs.DirEntry
	if err == nil {
		cases, err = os.ReadDir(tests)
	}
	if err != nil {
		return ex, err
	}
	for _, c := range cases {
		if !c.IsDir() {
			continue
		}
		tc := exampleCase{Name: c.Name()}
		if tc.Before, err = readTree(filepath.Join(tests, c.Name(), "before"), nil); err != nil {
			return ex, err
		}
		after := filepath.Join(tests, c.Name(), "after")
		if _, err := os.Stat(after); err != nil {
			tc.NoAfter = true
		} else if tc.After, err = readTree(after, nil); err != nil {
			return ex, err
		}
		ex.Cases = append(ex.Cases, tc)
	}
	return ex, nil
}

// fileRank orders an example's files: change.star, other Starlark
// scripts, fleet.yaml, then the rest.
func fileRank(p string) int {
	switch {
	case p == "change.star":
		return 0
	case strings.HasSuffix(p, ".star"):
		return 1
	case p == "fleet.yaml":
		return 2
	}
	return 3
}

// readTree reads every file under root, sorted by slash-separated path,
// skipping the directories skip reports. A file that isn't text is listed
// without its content.
func readTree(root string, skip func(string) bool) ([]exampleFile, error) {
	var out []exampleFile
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if skip != nil && skip(p) {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		content := string(b)
		if !utf8.Valid(b) || bytes.IndexByte(b, 0) >= 0 {
			content = ""
		}
		out = append(out, exampleFile{Path: filepath.ToSlash(rel), Content: content})
		return nil
	})
	return out, err
}

// ExamplePage renders one example's page: what it does, how to start from
// it, every file of the campaign in tabs, and its test cases.
func ExamplePage(ex Example) []byte {
	var b bytes.Buffer
	fmt.Fprintf(&b, "---\ntitle: %s\n---\n\n# %s\n\n", ex.Name, ex.Name)
	if ex.Description != "" {
		b.WriteString(sentence(ex.Description) + "\n\n")
	}
	if len(ex.Uses) > 0 {
		links := make([]string, len(ex.Uses))
		for i, u := range ex.Uses {
			mod, name, _ := strings.Cut(u, "/")
			mod = strings.TrimSuffix(mod, ".star")
			anchor := ""
			if ex.Funcs[u] {
				anchor = "#" + name
			}
			links[i] = fmt.Sprintf("[`%s.%s`](../starlark/%s.md%s)", mod, name, mod, anchor)
		}
		fmt.Fprintf(&b, "**Uses:** %s\n\n", strings.Join(links, ", "))
	}
	fmt.Fprintf(&b, "Start a campaign from it, then point it at your organization:\n\n```sh\ngh fleet init --from %s --org your-org\n```\n\n", ex.Name)
	b.WriteString("## Campaign\n\n")
	writeTabs(&b, ex.Files, "")
	if len(ex.Cases) > 0 {
		b.WriteString("## Test cases\n\n")
		fmt.Fprintf(&b, "`gh fleet test` runs the change on each case's `before/` files and checks the result against `after/`. See [Testing a campaign](../../guides/testing.md).\n\n")
		for _, c := range ex.Cases {
			fmt.Fprintf(&b, "### %s\n\n", c.Name)
			b.WriteString("=== \"Before\"\n\n")
			writeFiles(&b, c.Before, "    ")
			b.WriteString("=== \"After\"\n\n")
			if c.NoAfter {
				b.WriteString("    No changes: the campaign leaves this repo alone.\n\n")
			} else {
				writeFiles(&b, c.After, "    ")
			}
		}
	}
	fmt.Fprintf(&b, "[Source on GitHub](%s/tree/main/examples/%s)\n", repoURL, ex.Name)
	return b.Bytes()
}

// writeTabs writes one content tab per file, indented by indent.
func writeTabs(b *bytes.Buffer, files []exampleFile, indent string) {
	for _, f := range files {
		fmt.Fprintf(b, "%s=== \"%s\"\n\n", indent, f.Path)
		writeContent(b, f, indent+"    ")
	}
}

// writeFiles writes each file's path and content in sequence.
func writeFiles(b *bytes.Buffer, files []exampleFile, indent string) {
	for _, f := range files {
		fmt.Fprintf(b, "%s`%s`\n\n", indent, f.Path)
		writeContent(b, f, indent)
	}
}

// writeContent writes a file's content as a code block, or a note for a
// file that isn't text or is empty.
func writeContent(b *bytes.Buffer, f exampleFile, indent string) {
	if strings.TrimSpace(f.Content) == "" {
		fmt.Fprintf(b, "%s_Empty or not text._\n\n", indent)
		return
	}
	b.WriteString(fence(lang(f.Path), f.Content, indent))
	b.WriteString("\n")
}

// writeExamples writes a page per example under out/examples and the
// matching nav entries into nav.
func writeExamples(out, nav, examples string, used map[string][]string) error {
	exs, err := readExamples(examples, used)
	if err != nil {
		return err
	}
	docs, err := libDocs()
	if err != nil {
		return err
	}
	funcs := map[string]bool{}
	for _, d := range docs {
		for _, f := range d.Funcs {
			funcs[d.Name+"/"+f.Name] = true
		}
	}
	dir := filepath.Join(out, "examples")
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	entries := []string{"Overview: examples.md"}
	for _, ex := range exs {
		ex.Funcs = funcs
		if err := os.WriteFile(filepath.Join(dir, ex.Name+".md"), ExamplePage(ex), 0o644); err != nil { //nolint:gosec // docs are public
			return err
		}
		entries = append(entries, fmt.Sprintf("%s: reference/examples/%s.md", ex.Name, ex.Name))
	}
	if nav == "" {
		return nil
	}
	return updateNavBlock(nav, examplesNavBegin, examplesNavEnd, entries)
}
