// Package campaign loads, defaults and validates gh fleet campaign files.
//
// A campaign is one YAML file that says which repositories to target, what
// to run in each and how to open the pull request. Files are validated
// against a JSON Schema embedded in the binary, and every error carries the
// line and column it refers to.
package campaign

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

// APIVersion is the only campaign apiVersion this binary understands.
const APIVersion = "fleet.gh/v1"

// DefaultFile is the campaign file commands read when none is given.
const DefaultFile = "fleet.yaml"

// Campaign is a parsed, defaulted campaign file.
type Campaign struct {
	APIVersion  string  `yaml:"apiVersion" json:"apiVersion"`
	Name        string  `yaml:"name" json:"name"`
	Description string  `yaml:"description,omitempty" json:"description,omitempty"`
	Targets     Targets `yaml:"targets" json:"targets"`
	Change      Change  `yaml:"change" json:"change"`
	PR          PR      `yaml:"pr" json:"pr"`
	Limits      Limits  `yaml:"limits" json:"limits"`

	// Path is the absolute path of the file the campaign was loaded from.
	Path string `yaml:"-" json:"path,omitempty"`
}

// Targets selects the repositories a campaign touches.
type Targets struct {
	Orgs    []string `yaml:"orgs,omitempty" json:"orgs,omitempty"`
	Include Include  `yaml:"include,omitempty" json:"include"`
	Exclude Exclude  `yaml:"exclude,omitempty" json:"exclude"`
}

// Include holds the filters a candidate repo must pass.
type Include struct {
	Topics    []string `yaml:"topics,omitempty" json:"topics,omitempty"`
	Languages []string `yaml:"languages,omitempty" json:"languages,omitempty"`
	Files     []string `yaml:"files,omitempty" json:"files,omitempty"`
	Search    string   `yaml:"search,omitempty" json:"search,omitempty"`
	Repos     []string `yaml:"repos,omitempty" json:"repos,omitempty"`
}

// Exclude holds the rules that drop a repo. Exclusion always wins.
type Exclude struct {
	Repos    []string `yaml:"repos,omitempty" json:"repos,omitempty"`
	Archived *bool    `yaml:"archived,omitempty" json:"archived,omitempty"`
	Forks    *bool    `yaml:"forks,omitempty" json:"forks,omitempty"`
	Topics   []string `yaml:"topics,omitempty" json:"topics,omitempty"`
}

// Change describes the script run in each repository.
type Change struct {
	Run     string            `yaml:"run" json:"run"`
	Shell   string            `yaml:"shell,omitempty" json:"shell,omitempty"`
	Env     map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
	Timeout Duration          `yaml:"timeout,omitempty" json:"timeout,omitempty"`
}

// PR describes the pull request opened in each repository.
type PR struct {
	Title     string   `yaml:"title" json:"title"`
	Body      string   `yaml:"body,omitempty" json:"body,omitempty"`
	Branch    string   `yaml:"branch,omitempty" json:"branch,omitempty"`
	Base      string   `yaml:"base,omitempty" json:"base,omitempty"`
	Draft     bool     `yaml:"draft,omitempty" json:"draft,omitempty"`
	Labels    []string `yaml:"labels,omitempty" json:"labels,omitempty"`
	Reviewers []string `yaml:"reviewers,omitempty" json:"reviewers,omitempty"`
	Commit    Commit   `yaml:"commit,omitempty" json:"commit"`
}

// Commit describes the commit made in each repository.
type Commit struct {
	Message string `yaml:"message,omitempty" json:"message,omitempty"`
	Signoff bool   `yaml:"signoff,omitempty" json:"signoff,omitempty"`
}

// Limits caps how much a single run does.
type Limits struct {
	Concurrency int `yaml:"concurrency,omitempty" json:"concurrency,omitempty"`
	MaxOpenPRs  int `yaml:"maxOpenPRs,omitempty" json:"maxOpenPRs,omitempty"`
}

// Duration is a time.Duration written as a Go duration string in YAML.
type Duration time.Duration

// UnmarshalText parses a Go duration string such as "5m".
func (d *Duration) UnmarshalText(b []byte) error {
	v, err := time.ParseDuration(string(b))
	if err != nil {
		return err
	}
	*d = Duration(v)
	return nil
}

// MarshalText writes the duration as a Go duration string.
func (d Duration) MarshalText() ([]byte, error) {
	return []byte(time.Duration(d).String()), nil
}

// Std returns the duration as a time.Duration.
func (d Duration) Std() time.Duration { return time.Duration(d) }

// Defaults used when a campaign leaves a field empty.
const (
	DefaultShell       = "bash"
	DefaultTimeout     = 5 * time.Minute
	DefaultConcurrency = 8
	BranchPrefix       = "fleet/"
)

// ApplyDefaults fills every empty optional field with its default value.
func (c *Campaign) ApplyDefaults() {
	t := true
	if c.Targets.Exclude.Archived == nil {
		c.Targets.Exclude.Archived = &t
	}
	if c.Targets.Exclude.Forks == nil {
		f := true
		c.Targets.Exclude.Forks = &f
	}
	if c.Change.Shell == "" {
		c.Change.Shell = DefaultShell
	}
	if c.Change.Timeout == 0 {
		c.Change.Timeout = Duration(DefaultTimeout)
	}
	if c.PR.Branch == "" {
		c.PR.Branch = BranchPrefix + c.Name
	}
	if c.PR.Commit.Message == "" {
		c.PR.Commit.Message = c.PR.Title
	}
	if c.Limits.Concurrency == 0 {
		c.Limits.Concurrency = DefaultConcurrency
	}
}

// Marker is the hidden comment that identifies a campaign's PRs.
func (c *Campaign) Marker() string {
	return Marker(c.Name)
}

// Marker returns the hidden PR body marker for the named campaign.
func Marker(name string) string {
	return "<!-- gh-fleet:campaign=" + name + " -->"
}

// Dir is the directory relative paths in the campaign resolve against.
func (c *Campaign) Dir() string {
	if c.Path == "" {
		return "."
	}
	return filepath.Dir(c.Path)
}

// ScriptFile returns the absolute path of change.run when it names a file
// next to the campaign, or "" when change.run is an inline script.
func (c *Campaign) ScriptFile() string {
	return c.fileRef(c.Change.Run)
}

// fileRef returns the absolute path of v when it is a single-line value
// naming an existing file relative to the campaign directory, else "".
func (c *Campaign) fileRef(v string) string {
	if v == "" || strings.ContainsAny(v, "\n") {
		return ""
	}
	p := v
	if !filepath.IsAbs(p) {
		p = filepath.Join(c.Dir(), p)
	}
	if st, err := os.Stat(p); err == nil && st.Mode().IsRegular() {
		abs, err := filepath.Abs(p)
		if err != nil {
			return p
		}
		return abs
	}
	return ""
}

// looksLikePath reports whether a single-line value is clearly meant as a
// relative file path rather than inline text.
func looksLikePath(v string) bool {
	if strings.ContainsAny(v, "\n ") {
		return false
	}
	return strings.HasPrefix(v, "./") || strings.HasPrefix(v, "../")
}

// RepoData is the repository view passed to PR templates as .Repo.
type RepoData struct {
	Owner         string
	Name          string
	FullName      string
	DefaultBranch string
}

// templateData is the root object passed to PR title and body templates.
type templateData struct {
	Repo     RepoData
	Campaign *Campaign
}

// RenderTitle renders the PR title template for one repo.
func (c *Campaign) RenderTitle(repo RepoData) (string, error) {
	return c.render("title", c.PR.Title, repo)
}

// RenderCommitMessage renders the commit message template for one repo.
func (c *Campaign) RenderCommitMessage(repo RepoData) (string, error) {
	return c.render("commit", c.PR.Commit.Message, repo)
}

// RenderBody renders the PR body for one repo, reading it from a file when
// pr.body names one, and appends the campaign marker.
func (c *Campaign) RenderBody(repo RepoData) (string, error) {
	src := c.PR.Body
	if p := c.fileRef(src); p != "" {
		b, err := os.ReadFile(p)
		if err != nil {
			return "", fmt.Errorf("read pr.body: %w", err)
		}
		src = string(b)
	}
	body, err := c.render("body", src, repo)
	if err != nil {
		return "", err
	}
	body = strings.TrimRight(body, "\n")
	if body != "" {
		body += "\n\n"
	}
	return body + c.Marker(), nil
}

// render executes one template against the repo and campaign.
func (c *Campaign) render(name, src string, repo RepoData) (string, error) {
	t, err := template.New(name).Option("missingkey=error").Parse(src)
	if err != nil {
		return "", fmt.Errorf("pr.%s template: %w", name, err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, templateData{Repo: repo, Campaign: c}); err != nil {
		return "", fmt.Errorf("pr.%s template: %w", name, err)
	}
	return buf.String(), nil
}
