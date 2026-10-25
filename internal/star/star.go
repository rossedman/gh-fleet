// Package star runs Starlark change scripts.
//
// A script defines change(repo) and edits files through the repo object.
// It runs inside gh fleet with no shell, network or clock: the only things
// it can touch are the files of the clone it is given, which are opened
// through an os.Root so no path, symlink included, can escape it. Every
// script, and everything it loads, is parsed, resolved and run to the top
// level when the campaign loads, before any repo is cloned.
package star

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"go.starlark.net/starlark"
	"go.starlark.net/syntax"

	"github.com/rossedman/gh-fleet/internal/gitops"
)

// MaxSteps bounds how much work one run may do, so a runaway loop fails
// instead of hanging a worker.
const MaxSteps = 200_000_000

// fileOptions are the Starlark dialect scripts are written in: Python-like
// sets, while loops and top-level control flow are allowed; recursion is not.
var fileOptions = &syntax.FileOptions{Set: true, While: true, TopLevelControl: true, GlobalReassign: false}

// Options configures Load.
type Options struct {
	// CacheDir holds libraries loaded from other repositories.
	CacheDir string
	// Git fetches those libraries.
	Git gitops.Git
	// RemoteURL returns the clone URL of a library repository; defaults to
	// https://<host>/<owner>/<repo>.git.
	RemoteURL func(host, owner, repo string) string
	// Env is the step's environment (change.env plus the step's env),
	// available to scripts as env.
	Env map[string]string
}

// Script is a compiled change script and every module it loads.
type Script struct {
	// Path is the absolute path of the script.
	Path    string
	root    string
	modules map[string]*module
	// byName maps the file names Starlark reports in positions back to
	// paths: campaign-relative for local files, host/owner/repo@sha:path
	// for libraries.
	byName map[string]string
	opts   Options
}

// module is one compiled .star file.
type module struct {
	prog *starlark.Program
	// root is the directory loads from this module must stay inside: the
	// campaign directory, or the checkout of a library repository.
	root string
}

// predeclared are the names every script and module can use: the
// modules, env (the step's environment as a frozen dict) and campaign
// (read-only access to the files next to the campaign file).
func (s *Script) predeclared() starlark.StringDict {
	env := starlark.NewDict(len(s.opts.Env))
	for _, k := range sortedKeys(s.opts.Env) {
		_ = env.SetKey(starlark.String(k), starlark.String(s.opts.Env[k]))
	}
	env.Freeze()
	return starlark.StringDict{
		"json":     jsonModule,
		"yaml":     yamlModule,
		"semver":   semverModule,
		"re":       reModule,
		"env":      env,
		"campaign": &campaignDir{dir: s.root},
	}
}

// Load compiles the script at path and everything it loads, fetching
// libraries from other repositories into the cache, and checks that it
// defines change(repo). campaignDir is the directory local loads must stay
// inside.
func Load(ctx context.Context, path, campaignDir string, opts Options) (*Script, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	root, err := filepath.Abs(campaignDir)
	if err != nil {
		return nil, err
	}
	s := &Script{Path: abs, root: root, modules: map[string]*module{}, byName: map[string]string{}, opts: opts}
	if err := s.compile(ctx, abs, root, relTo(root, abs), nil); err != nil {
		return nil, err
	}
	// Run the top level once, with no repo, to check that it defines a
	// change function and that every load resolves.
	thread, stop := s.thread(ctx, io.Discard)
	defer stop()
	globals, err := s.exec(thread, abs)
	if err != nil {
		return nil, describe(err)
	}
	fn, ok := globals["change"].(*starlark.Function)
	if !ok {
		return nil, fmt.Errorf("%s: no change(repo) function defined", relTo(root, abs))
	}
	if fn.NumParams() != 1 {
		return nil, fmt.Errorf("%s: change must take exactly one parameter, repo; it takes %d", relTo(root, abs), fn.NumParams())
	}
	return s, nil
}

// compile parses and resolves one file and, recursively, the files it
// loads. name is how the file appears in errors; stack detects load cycles.
func (s *Script) compile(ctx context.Context, path, root, name string, stack []string) error {
	if _, ok := s.modules[path]; ok {
		return nil
	}
	for _, p := range stack {
		if p == path {
			return fmt.Errorf("load cycle: %s", strings.Join(append(stack, path), " -> "))
		}
	}
	src, err := s.source(path)
	if err != nil {
		return err
	}
	pre := s.predeclared()
	f, prog, err := starlark.SourceProgramOptions(fileOptions, name, src, pre.Has)
	if err != nil {
		return err
	}
	s.modules[path] = &module{prog: prog, root: root}
	s.byName[name] = path
	for _, stmt := range f.Stmts {
		load, ok := stmt.(*syntax.LoadStmt)
		if !ok {
			continue
		}
		target, targetRoot, err := s.resolveLoad(ctx, load.ModuleName(), path, root)
		if err != nil {
			pos := load.Module.TokenPos
			return fmt.Errorf("%s:%d:%d: load %q: %w", name, pos.Line, pos.Col, load.ModuleName(), err)
		}
		if err := s.compile(ctx, target, targetRoot, s.displayName(load.ModuleName(), target), append(stack, path)); err != nil {
			return err
		}
	}
	return nil
}

// source returns a module's source: from the embedded library for
// @fleet// paths, else from disk.
func (s *Script) source(path string) ([]byte, error) {
	if name, ok := strings.CutPrefix(path, LibPrefix); ok {
		return fs.ReadFile(libFS, "lib/"+name)
	}
	return os.ReadFile(path)
}

// displayName is how a loaded file appears in errors: the library
// reference with a short SHA, the built-in library path, or the path
// relative to the campaign.
func (s *Script) displayName(loadName, path string) string {
	if strings.HasPrefix(path, LibPrefix) {
		return path
	}
	if m := remoteLoad.FindStringSubmatch(loadName); m != nil {
		dir := filepath.Join(s.opts.CacheDir, m[1], m[2], m[3], m[4])
		return m[1] + "/" + m[2] + "/" + m[3] + "@" + m[4][:12] + ":" + relTo(dir, path)
	}
	return relTo(s.root, path)
}

// remoteLoad matches host/owner/repo@sha:path.star.
var remoteLoad = regexp.MustCompile(`^([a-z0-9.-]+\.[a-z]{2,})/([A-Za-z0-9_.-]+)/([A-Za-z0-9_.-]+)@([0-9a-f]{40}):(.+\.star)$`)

// resolveLoad turns a load() module name into a file path and the root
// that file's own loads must stay inside.
func (s *Script) resolveLoad(ctx context.Context, name, from, root string) (string, string, error) {
	if root == LibPrefix && strings.HasPrefix(name, "./") {
		name = LibPrefix + strings.TrimPrefix(name, "./")
	}
	if strings.HasPrefix(name, LibPrefix) {
		return resolveLib(name)
	}
	if root == LibPrefix {
		return "", "", fmt.Errorf("the built-in library can only load %s modules", LibPrefix)
	}
	if m := remoteLoad.FindStringSubmatch(name); m != nil {
		dir, err := s.fetch(ctx, m[1], m[2], m[3], m[4])
		if err != nil {
			return "", "", err
		}
		return within(dir, m[5])
	}
	if strings.Contains(name, "@") && !strings.HasPrefix(name, ".") {
		return "", "", errors.New("libraries from other repositories are loaded as host/owner/repo@<full commit SHA>:path.star")
	}
	if !strings.HasPrefix(name, "./") && !strings.HasPrefix(name, "../") {
		return "", "", errors.New(`load a file relative to this one ("./lib.star"), a built-in module ("@fleet//files.star") or host/owner/repo@<sha>:path.star`)
	}
	if !strings.HasSuffix(name, ".star") {
		return "", "", errors.New("only .star files can be loaded")
	}
	p := filepath.Join(filepath.Dir(from), filepath.FromSlash(name))
	rel, err := filepath.Rel(root, p)
	if err != nil || !filepath.IsLocal(rel) {
		return "", "", fmt.Errorf("%s is outside %s", name, relTo(s.root, root))
	}
	return within(root, filepath.ToSlash(rel))
}

// within joins a slash-separated relative path onto dir, refusing paths
// that leave it.
func within(dir, rel string) (string, string, error) {
	local := filepath.FromSlash(rel)
	if !filepath.IsLocal(local) {
		return "", "", fmt.Errorf("%s leaves its repository", rel)
	}
	return filepath.Join(dir, local), dir, nil
}

// fetch makes sure a library repository is checked out at sha in the cache
// and returns its directory. A checkout is immutable once made, since the
// commit is pinned.
func (s *Script) fetch(ctx context.Context, host, owner, repo, sha string) (string, error) {
	if s.opts.CacheDir == "" {
		return "", errors.New("no cache directory for libraries")
	}
	dir := filepath.Join(s.opts.CacheDir, host, owner, repo, sha)
	if st, err := os.Stat(dir); err == nil && st.IsDir() {
		return dir, nil
	}
	url := "https://" + host + "/" + owner + "/" + repo + ".git"
	if s.opts.RemoteURL != nil {
		url = s.opts.RemoteURL(host, owner, repo)
	}
	if err := os.MkdirAll(filepath.Dir(dir), 0o700); err != nil {
		return "", err
	}
	tmp, err := os.MkdirTemp(filepath.Dir(dir), sha+".tmp-")
	if err != nil {
		return "", err
	}
	defer func() { _ = os.RemoveAll(tmp) }()
	g := s.opts.Git
	for _, args := range [][]string{
		{"init", "--quiet"},
		{"fetch", "--quiet", "--depth", "1", url, sha},
		{"checkout", "--quiet", "--detach", "FETCH_HEAD"},
	} {
		if _, err := g.Run(ctx, tmp, args...); err != nil {
			return "", fmt.Errorf("fetch %s/%s/%s@%s: %w", host, owner, repo, sha[:12], err)
		}
	}
	got, err := g.Rev(ctx, tmp, "HEAD")
	if err != nil {
		return "", err
	}
	if got != sha {
		return "", fmt.Errorf("fetch %s/%s/%s: got commit %s, want %s", host, owner, repo, got, sha)
	}
	if err := os.Rename(tmp, dir); err != nil && !errors.Is(err, os.ErrExist) {
		// Another worker may have won the race; its checkout is identical.
		if st, statErr := os.Stat(dir); statErr != nil || !st.IsDir() {
			return "", err
		}
	}
	return dir, nil
}

// thread returns a Starlark thread whose loads come from the compiled
// modules, whose print goes to log and which is canceled when ctx is done.
// Call stop when the thread is finished.
func (s *Script) thread(ctx context.Context, log io.Writer) (*starlark.Thread, func() bool) {
	cache := map[string]*loadResult{}
	thread := &starlark.Thread{
		Name:  "gh-fleet",
		Print: func(_ *starlark.Thread, msg string) { fmt.Fprintln(log, msg) },
	}
	thread.SetMaxExecutionSteps(MaxSteps)
	thread.Load = func(th *starlark.Thread, name string) (starlark.StringDict, error) {
		from := th.CallFrame(0).Pos.Filename()
		fromPath := s.byName[from]
		if m, ok := s.modules[fromPath]; ok {
			target, _, err := s.resolveLoad(ctx, name, fromPath, m.root)
			if err != nil {
				return nil, err
			}
			if r, ok := cache[target]; ok {
				return r.globals, r.err
			}
			cache[target] = &loadResult{err: fmt.Errorf("load cycle through %s", name)}
			g, err := s.exec(th, target)
			cache[target] = &loadResult{globals: g, err: err}
			return g, err
		}
		return nil, fmt.Errorf("load %q from unknown module %s", name, from)
	}
	stop := context.AfterFunc(ctx, func() { thread.Cancel("canceled") })
	return thread, stop
}

// loadResult is a module executed once per thread.
type loadResult struct {
	globals starlark.StringDict
	err     error
}

// exec runs one compiled module's top level.
func (s *Script) exec(thread *starlark.Thread, path string) (starlark.StringDict, error) {
	m, ok := s.modules[path]
	if !ok {
		return nil, fmt.Errorf("%s was not compiled", path)
	}
	g, err := m.prog.Init(thread, s.predeclared())
	if err != nil {
		return nil, err
	}
	g.Freeze()
	return g, nil
}

// Result is what a run produced besides file changes.
type Result struct {
	// Vars are the values set with repo.set_var.
	Vars map[string]any
}

// Run calls change(repo) for the clone in dir. Every file write is logged
// with the script line that made it. timeout of zero means none.
func (s *Script) Run(ctx context.Context, dir string, info RepoInfo, log io.Writer, timeout time.Duration) (Result, error) {
	if log == nil {
		log = io.Discard
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return Result{}, err
	}
	defer func() { _ = root.Close() }()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	thread, stop := s.thread(ctx, log)
	defer stop()
	globals, err := s.exec(thread, s.Path)
	if err != nil {
		return Result{}, s.runError(ctx, err, timeout)
	}
	repo := newRepo(root, info, log)
	if _, err := starlark.Call(thread, globals["change"], starlark.Tuple{repo}, nil); err != nil {
		return Result{}, s.runError(ctx, err, timeout)
	}
	return Result{Vars: repo.vars}, nil
}

// runError explains a failed run, naming a timeout when that was the cause.
func (s *Script) runError(ctx context.Context, err error, timeout time.Duration) error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("starlark change timed out after %s", timeout)
	}
	return describe(err)
}

// describe turns a Starlark error into "file:line:col: message" on the
// first line, where one-line progress output shows it, followed by the
// full traceback.
func describe(err error) error {
	var ee *starlark.EvalError
	if !errors.As(err, &ee) {
		return err
	}
	where := ""
	for i := len(ee.CallStack) - 1; i >= 0; i-- {
		if pos := ee.CallStack[i].Pos; pos.Filename() != "<builtin>" {
			where = pos.String() + ": "
			break
		}
	}
	return fmt.Errorf("%s%s\n%s", where, ee.Msg, strings.TrimSpace(ee.Backtrace()))
}

// relTo returns path relative to base for messages, or path itself.
func relTo(base, path string) string {
	if rel, err := filepath.Rel(base, path); err == nil && filepath.IsLocal(rel) {
		return filepath.ToSlash(rel)
	}
	return path
}
