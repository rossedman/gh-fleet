package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/rossedman/gh-fleet/internal/campaign"
	"github.com/rossedman/gh-fleet/internal/engine"
	"github.com/rossedman/gh-fleet/internal/ghapi"
	"github.com/rossedman/gh-fleet/internal/target"
)

// TestDir is the directory, next to the campaign file, that holds test
// cases.
const TestDir = "tests"

// TestOptions configures gh fleet test.
type TestOptions struct {
	Common
	// Run only runs cases whose name matches this regular expression.
	Run string
	// Update rewrites each case's after/ from what the change produced.
	Update bool
	// Dir runs the change against this directory instead of the test
	// cases and prints the diff.
	Dir string
	// Verbose prints each case's step output and PR title.
	Verbose bool
}

// TestCase is the optional case.yaml of a test case: the repo the change
// runs as and what to expect besides files.
type TestCase struct {
	Repo          string            `yaml:"repo" json:"repo,omitempty"`
	DefaultBranch string            `yaml:"defaultBranch" json:"defaultBranch,omitempty"`
	Language      string            `yaml:"language" json:"language,omitempty"`
	Topics        []string          `yaml:"topics" json:"topics,omitempty"`
	Properties    map[string]string `yaml:"properties" json:"properties,omitempty"`
	// Vars are set on top of the campaign's variables for the repo.
	Vars map[string]any `yaml:"vars" json:"vars,omitempty"`
	// Env overrides the steps' environment.
	Env map[string]string `yaml:"env" json:"env,omitempty"`
	// WantVars are variables the steps must set, compared as text.
	WantVars map[string]any `yaml:"wantVars" json:"wantVars,omitempty"`
	// WantTitle is the PR title the case must render.
	WantTitle string `yaml:"wantTitle" json:"wantTitle,omitempty"`
	// WantError is text the change's error must contain; the case then
	// expects the change to fail.
	WantError string `yaml:"wantError" json:"wantError,omitempty"`
}

// TestResult is the outcome of one case.
type TestResult struct {
	Name string `json:"name"`
	// Status is pass, fail or updated.
	Status string         `json:"status"`
	Errors []string       `json:"errors,omitempty"`
	Diff   string         `json:"diff,omitempty"`
	Vars   map[string]any `json:"vars,omitempty"`
	Title  string         `json:"title,omitempty"`
	Log    string         `json:"-"`
}

// Test runs a campaign's change against its test cases, or against a
// local directory with --dir.
func (a *App) Test(ctx context.Context, o TestOptions) error {
	camp, err := a.LoadCampaign(o.Common)
	if err != nil {
		return err
	}
	if o.Dir != "" {
		return a.testDir(ctx, camp, o)
	}
	root := filepath.Join(camp.Dir(), TestDir)
	names, err := testCases(root)
	if err != nil {
		return err
	}
	if o.Run != "" {
		re, err := regexp.Compile(o.Run)
		if err != nil {
			return fmt.Errorf("--run: %w", err)
		}
		names = slices.DeleteFunc(names, func(n string) bool { return !re.MatchString(n) })
	}
	if len(names) == 0 {
		return fmt.Errorf("no test cases in %s; each case is a directory with before/ and, when the change edits files, after/", root)
	}
	p := a.printer(o.Common)
	var results []TestResult
	count := map[string]int{}
	for _, name := range names {
		r := a.testCase(ctx, camp, filepath.Join(root, name), name, o.Update)
		count[r.Status]++
		results = append(results, r)
		if !p.Machine() {
			a.printTest(o, r)
		}
	}
	if p.Machine() {
		if err := p.PrintJSON(results); err != nil {
			return err
		}
	} else {
		summary := fmt.Sprintf("%d passed, %d failed", count["pass"], count["fail"])
		if count["updated"] > 0 {
			summary += fmt.Sprintf(", %d updated", count["updated"])
		}
		fmt.Fprintln(a.Err, summary)
	}
	if count["fail"] > 0 {
		return &FailedError{N: count["fail"], Noun: "test case"}
	}
	return nil
}

// testCases lists the case directories under root.
func testCases(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	return out, nil
}

// loadCase reads a case's case.yaml, if any.
func loadCase(dir string) (TestCase, error) {
	var tc TestCase
	b, err := os.ReadFile(filepath.Join(dir, "case.yaml"))
	if errors.Is(err, fs.ErrNotExist) {
		return tc, nil
	}
	if err != nil {
		return tc, err
	}
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	if err := dec.Decode(&tc); err != nil && !errors.Is(err, io.EOF) {
		return tc, fmt.Errorf("case.yaml: %w", err)
	}
	return tc, nil
}

// testTarget builds the target a case runs as.
func testTarget(camp *campaign.Campaign, tc TestCase) (target.Target, error) {
	repo := ghapi.Repo{Owner: "acme", Name: "test", DefaultBranch: tc.DefaultBranch, Language: tc.Language, Topics: tc.Topics, Properties: tc.Properties}
	if tc.Repo != "" {
		owner, name, ok := strings.Cut(tc.Repo, "/")
		if !ok || owner == "" || name == "" || strings.Contains(name, "/") {
			return target.Target{}, fmt.Errorf("case.yaml: repo %q is not owner/name", tc.Repo)
		}
		repo.Owner, repo.Name = owner, name
	}
	if repo.DefaultBranch == "" {
		repo.DefaultBranch = "main"
	}
	vars := target.Vars(&camp.Vars, repo)
	maps.Copy(vars, tc.Vars)
	return target.Target{Repo: repo, Vars: vars}, nil
}

// withEnv returns a copy of camp whose steps all see env on top of their
// own.
func withEnv(camp *campaign.Campaign, env map[string]string) *campaign.Campaign {
	if len(env) == 0 {
		return camp
	}
	c := *camp
	c.Change.Env = maps.Clone(camp.Change.Env)
	if c.Change.Env == nil {
		c.Change.Env = map[string]string{}
	}
	maps.Copy(c.Change.Env, env)
	c.Change.Steps = slices.Clone(camp.Change.Steps)
	for i := range c.Change.Steps {
		st := &c.Change.Steps[i]
		st.Env = maps.Clone(st.Env)
		for k := range env {
			delete(st.Env, k)
		}
	}
	return &c
}

// testCase runs one case: the change on a copy of before/, compared with
// after/, then a second run that must change nothing.
func (a *App) testCase(ctx context.Context, camp *campaign.Campaign, dir, name string, update bool) TestResult {
	r := TestResult{Name: name, Status: "pass"}
	fail := func(format string, args ...any) TestResult {
		r.Status = "fail"
		r.Errors = append(r.Errors, fmt.Sprintf(format, args...))
		return r
	}
	tc, err := loadCase(dir)
	if err != nil {
		return fail("%v", err)
	}
	t, err := testTarget(camp, tc)
	if err != nil {
		return fail("%v", err)
	}
	before := filepath.Join(dir, "before")
	if _, err := os.Stat(before); err != nil {
		return fail("no before/ directory")
	}
	work, err := os.MkdirTemp("", "fleet-test-")
	if err != nil {
		return fail("%v", err)
	}
	defer func() { _ = os.RemoveAll(work) }()
	if err := copyTree(before, work); err != nil {
		return fail("%v", err)
	}
	if err := a.gitSnapshot(ctx, work, true); err != nil {
		return fail("%v", err)
	}
	c := withEnv(camp, tc.Env)
	pipe, err := a.localPipeline(ctx, c)
	if err != nil {
		return fail("%v", err)
	}
	var log bytes.Buffer
	res, runErr := pipe.RunLocal(ctx, work, t, &log)
	r.Log, r.Vars = log.String(), res.Vars
	if runErr == nil {
		// Like apply, render the PR only for a repo the change edited.
		edited, err := treeDiff(ctx, before, work)
		if err != nil {
			return fail("%v", err)
		}
		if edited != "" {
			res, runErr = pipe.RenderLocal(t, res)
			r.Title = res.Title
		}
	}
	if tc.WantError != "" {
		switch {
		case runErr == nil:
			return fail("the change succeeded; want an error containing %q", tc.WantError)
		case !strings.Contains(runErr.Error(), tc.WantError):
			return fail("the error does not contain %q: %v", tc.WantError, runErr)
		}
		return r
	}
	if runErr != nil {
		return fail("%v", runErr)
	}
	for _, k := range slices.Sorted(maps.Keys(tc.WantVars)) {
		want, got := campaign.FormatValue(tc.WantVars[k]), ""
		v, ok := res.Vars[k]
		if ok {
			got = campaign.FormatValue(v)
		}
		if !ok || got != want {
			fail("var %s = %q, want %q", k, got, want)
		}
	}
	if tc.WantTitle != "" && res.Title != tc.WantTitle {
		if res.Title == "" {
			fail("the change edited nothing, so there is no PR title; want %q", tc.WantTitle)
		} else {
			fail("PR title = %q, want %q", res.Title, tc.WantTitle)
		}
	}
	after := filepath.Join(dir, "after")
	if update {
		if err := writeAfter(before, work, after); err != nil {
			return fail("%v", err)
		}
		if r.Status == "pass" {
			r.Status = "updated"
		}
	} else {
		want := after
		if _, err := os.Stat(after); err != nil {
			want = before
		}
		diff, err := treeDiff(ctx, want, work)
		if err != nil {
			return fail("%v", err)
		}
		if diff != "" {
			r.Diff = diff
			if want == before {
				fail("the change edited files, but the case has no after/")
			} else {
				fail("files differ from after/")
			}
		}
	}

	// A second run on the result must be a no-op, as it is when gh fleet
	// re-runs a campaign on a repo whose PR is already open.
	if err := a.gitSnapshot(ctx, work, false); err != nil {
		return fail("%v", err)
	}
	if _, err := pipe.RunLocal(ctx, work, t, io.Discard); err != nil {
		return fail("second run: %v", err)
	}
	if again, err := a.Git.StagedDiff(ctx, work, "HEAD"); err != nil {
		return fail("%v", err)
	} else if again != "" {
		r.Diff += again + "\n"
		fail("a second run changed files again; a re-run must change nothing")
	}
	return r
}

// localPipeline compiles a campaign's scripts into a plan-mode pipeline.
func (a *App) localPipeline(ctx context.Context, camp *campaign.Campaign) (*engine.Pipeline, error) {
	scripts, err := a.scripts(ctx, camp)
	if err != nil {
		return nil, err
	}
	return &engine.Pipeline{Mode: engine.Plan, Campaign: camp, Git: a.Git, Scripts: scripts}, nil
}

// gitSnapshot commits everything in dir, first creating the repo when init
// is set, so steps that call git see a clone-like checkout.
func (a *App) gitSnapshot(ctx context.Context, dir string, init bool) error {
	if init {
		if _, err := a.Git.Run(ctx, dir, "init", "--quiet", "--initial-branch=main"); err != nil {
			return err
		}
	}
	if _, err := a.Git.Run(ctx, dir, "add", "--all"); err != nil {
		return err
	}
	_, err := a.Git.Run(ctx, dir, "-c", "user.name=gh fleet", "-c", "user.email=fleet@localhost", "-c", "commit.gpgsign=false",
		"commit", "--quiet", "--no-verify", "--allow-empty", "-m", "snapshot")
	return err
}

// snapshot reads every file under dir except .git.
func snapshot(dir string) (map[string][]byte, error) {
	out := map[string][]byte{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" && p != dir {
				return filepath.SkipDir
			}
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		out[filepath.ToSlash(rel)] = b
		return nil
	})
	return out, err
}

// treeDiff returns a unified diff from want to got, file by file; empty
// when the trees match.
func treeDiff(ctx context.Context, want, got string) (string, error) {
	w, err := snapshot(want)
	if err != nil {
		return "", err
	}
	g, err := snapshot(got)
	if err != nil {
		return "", err
	}
	paths := slices.Sorted(maps.Keys(w))
	for p := range g {
		if _, ok := w[p]; !ok {
			paths = append(paths, p)
		}
	}
	sort.Strings(paths)
	var b strings.Builder
	for _, p := range paths {
		wb, inW := w[p]
		gb, inG := g[p]
		if inW && inG && bytes.Equal(wb, gb) {
			continue
		}
		from, to := filepath.Join(want, filepath.FromSlash(p)), filepath.Join(got, filepath.FromSlash(p))
		if !inW {
			from = os.DevNull
		}
		if !inG {
			to = os.DevNull
		}
		d, err := fileDiff(ctx, from, to, p)
		if err != nil {
			return "", err
		}
		b.WriteString(d)
	}
	return b.String(), nil
}

// fileDiff diffs two files with git, labeling them want/<path> and
// got/<path>.
func fileDiff(ctx context.Context, from, to, label string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-c", "color.ui=never", "diff", "--no-index", "--no-ext-diff",
		"--src-prefix=want/", "--dst-prefix=got/", "--", from, to)
	var out, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &stderr
	err := cmd.Run()
	var exit *exec.ExitError
	if err != nil && (!errors.As(err, &exit) || exit.ExitCode() != 1) {
		return "", fmt.Errorf("git diff: %s", strings.TrimSpace(stderr.String()))
	}
	// git names the files by their temporary paths; show the repo path.
	lines := strings.SplitAfter(out.String(), "\n")
	for i, l := range lines {
		if strings.HasPrefix(l, "@@") {
			break
		}
		switch {
		case strings.HasPrefix(l, "diff --git "):
			lines[i] = "diff --git want/" + label + " got/" + label + "\n"
		case strings.HasPrefix(l, "--- ") && l != "--- /dev/null\n":
			lines[i] = "--- want/" + label + "\n"
		case strings.HasPrefix(l, "+++ ") && l != "+++ /dev/null\n":
			lines[i] = "+++ got/" + label + "\n"
		}
	}
	return strings.Join(lines, ""), nil
}

// copyTree copies the files under src into dst, skipping .git.
func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			if d.Name() == ".git" && p != src {
				return filepath.SkipDir
			}
			return os.MkdirAll(target, 0o755)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, b, info.Mode().Perm())
	})
}

// writeAfter replaces after/ with the change's result, or removes it when
// the result is the same as before/.
func writeAfter(before, work, after string) error {
	b, err := snapshot(before)
	if err != nil {
		return err
	}
	w, err := snapshot(work)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(after); err != nil {
		return err
	}
	if maps.EqualFunc(b, w, bytes.Equal) {
		return nil
	}
	return copyTree(work, after)
}

// printTest prints one case's result for people.
func (a *App) printTest(o TestOptions, r TestResult) {
	p := a.printer(o.Common)
	s := p.Styles
	mark := map[string]string{"pass": s.OK.Render("ok  "), "updated": s.Info.Render("upd "), "fail": s.Bad.Render("FAIL")}[r.Status]
	fmt.Fprintf(a.Out, "%s %s\n", mark, r.Name)
	for _, e := range r.Errors {
		fmt.Fprintf(a.Out, "     %s\n", strings.ReplaceAll(e, "\n", "\n     "))
	}
	if r.Diff != "" {
		p.Diff(r.Name, r.Diff)
	}
	if o.Verbose && r.Title != "" {
		fmt.Fprintf(a.Out, "     %s %s\n", s.Dim.Render("title:"), r.Title)
	}
	if (o.Verbose || r.Status == "fail") && r.Log != "" {
		fmt.Fprintln(a.Out, s.Dim.Render(strings.TrimRight(r.Log, "\n")))
	}
}

// testDir runs the change on a copy of a local directory and prints the
// diff it would make, the variables it set and the PR it would open.
func (a *App) testDir(ctx context.Context, camp *campaign.Campaign, o TestOptions) error {
	if o.Update || o.Run != "" {
		return errors.New("--dir can't be combined with --update or --run")
	}
	info, err := os.Stat(o.Dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("--dir %s is not a directory", o.Dir)
	}
	work, err := os.MkdirTemp("", "fleet-test-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(work) }()
	if err := copyTree(o.Dir, work); err != nil {
		return err
	}
	if err := a.gitSnapshot(ctx, work, true); err != nil {
		return err
	}
	abs, _ := filepath.Abs(o.Dir)
	t, err := testTarget(camp, TestCase{Repo: localRepo(o.Repos, abs)})
	if err != nil {
		return err
	}
	pipe, err := a.localPipeline(ctx, camp)
	if err != nil {
		return err
	}
	var log bytes.Buffer
	res, runErr := pipe.RunLocal(ctx, work, t, &log)
	diff, err := a.Git.StagedDiff(ctx, work, "HEAD")
	if err != nil {
		return err
	}
	if runErr == nil && diff != "" {
		res, runErr = pipe.RenderLocal(t, res)
	}
	p := a.printer(o.Common)
	if p.Machine() {
		out := TestResult{Name: t.Repo.FullName(), Status: "pass", Diff: diff, Vars: res.Vars, Title: res.Title}
		if runErr != nil {
			out.Status, out.Errors = "fail", []string{runErr.Error()}
		}
		if err := p.PrintJSON(out); err != nil {
			return err
		}
		return runErr
	}
	if o.Verbose && log.Len() > 0 {
		fmt.Fprintln(a.Err, p.Styles.Dim.Render(strings.TrimRight(log.String(), "\n")))
	}
	if runErr != nil {
		return runErr
	}
	if diff == "" {
		fmt.Fprintf(a.Err, "%s: no changes\n", t.Repo.FullName())
	} else {
		p.Diff(t.Repo.FullName(), diff)
	}
	for _, k := range slices.Sorted(maps.Keys(res.Vars)) {
		fmt.Fprintf(a.Out, "%s %s=%s\n", p.Styles.Dim.Render("var"), k, campaign.FormatValue(res.Vars[k]))
	}
	if diff != "" {
		fmt.Fprintf(a.Out, "%s %s\n", p.Styles.Dim.Render("title"), res.Title)
	}
	return nil
}

// localRepo names the repo a --dir run acts as: --repo when given, else
// acme/<directory name>.
func localRepo(repos []string, dir string) string {
	if len(repos) > 0 {
		return repos[0]
	}
	return "acme/" + filepath.Base(dir)
}
