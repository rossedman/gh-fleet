package star

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"go.starlark.net/starlark"
)

// RepoInfo is what a script knows about the repo it is changing.
type RepoInfo struct {
	Owner         string
	Name          string
	DefaultBranch string
	Topics        []string
	Language      string
	Properties    map[string]string
	Vars          map[string]any
}

// repo is the Starlark value passed to change(repo). Files are reached
// through root, which refuses any path that leaves the clone.
type repo struct {
	root *os.Root
	info RepoInfo
	log  io.Writer
	vars map[string]any
}

// newRepo returns the repo value for one run.
func newRepo(root *os.Root, info RepoInfo, log io.Writer) *repo {
	return &repo{root: root, info: info, log: log, vars: map[string]any{}}
}

// String describes the repo.
func (r *repo) String() string { return "<repo " + r.info.Owner + "/" + r.info.Name + ">" }

// Type names the value's type.
func (r *repo) Type() string { return "repo" }

// Freeze does nothing: the repo has no Starlark-visible mutable state.
func (r *repo) Freeze() {}

// Truth reports the repo as true.
func (r *repo) Truth() starlark.Bool { return true }

// Hash refuses to hash the repo.
func (r *repo) Hash() (uint32, error) { return 0, errors.New("unhashable type: repo") }

// methods are the repo's callable attributes.
var methods = map[string]func(r *repo, th *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error){
	"read":    (*repo).read,
	"write":   (*repo).write,
	"append":  (*repo).appendFile,
	"exists":  (*repo).exists,
	"delete":  (*repo).delete,
	"glob":    (*repo).glob,
	"set_var": (*repo).setVar,
}

// Attr returns a repo attribute or method.
func (r *repo) Attr(name string) (starlark.Value, error) {
	switch name {
	case "owner":
		return starlark.String(r.info.Owner), nil
	case "name":
		return starlark.String(r.info.Name), nil
	case "full_name":
		return starlark.String(r.info.Owner + "/" + r.info.Name), nil
	case "default_branch":
		return starlark.String(r.info.DefaultBranch), nil
	case "language":
		return starlark.String(r.info.Language), nil
	case "topics":
		return toStarlark(r.info.Topics)
	case "properties":
		return frozen(toStarlark(r.info.Properties))
	case "vars":
		merged := map[string]any{}
		for k, v := range r.info.Vars {
			merged[k] = v
		}
		for k, v := range r.vars {
			merged[k] = v
		}
		return frozen(toStarlark(merged))
	}
	if m, ok := methods[name]; ok {
		return starlark.NewBuiltin("repo."+name, func(th *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			return m(r, th, b, args, kwargs)
		}), nil
	}
	return nil, nil
}

// AttrNames lists the repo's attributes.
func (r *repo) AttrNames() []string {
	names := []string{"owner", "name", "full_name", "default_branch", "language", "topics", "properties", "vars"}
	for m := range methods {
		names = append(names, m)
	}
	sort.Strings(names)
	return names
}

// frozen freezes a converted value so scripts can't mutate shared state.
func frozen(v starlark.Value, err error) (starlark.Value, error) {
	if err == nil {
		v.Freeze()
	}
	return v, err
}

// localPath checks that a script-given path stays inside the repo and
// returns it in OS form.
func localPath(fn, p string) (string, error) {
	local := filepath.FromSlash(p)
	if p == "" || !filepath.IsLocal(local) {
		return "", fmt.Errorf("%s: %q is not a path inside the repo", fn, p)
	}
	// Clean before looking at the first segment, so ./.git and a/../.git
	// are caught too.
	local = filepath.Clean(local)
	if first, _, _ := strings.Cut(filepath.ToSlash(local), "/"); first == ".git" {
		return "", fmt.Errorf("%s: %q is inside .git", fn, p)
	}
	return local, nil
}

// caller returns "file:line" of the script line calling a builtin.
func caller(th *starlark.Thread) string {
	return th.CallFrame(1).Pos.String()
}

// read returns a file's content, or default when it doesn't exist.
func (r *repo) read(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var path string
	var def starlark.Value
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "path", &path, "default?", &def); err != nil {
		return nil, err
	}
	local, err := localPath(b.Name(), path)
	if err != nil {
		return nil, err
	}
	data, err := r.root.ReadFile(local)
	if errors.Is(err, fs.ErrNotExist) && def != nil {
		return def, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %s: %w", b.Name(), path, unwrapPath(err))
	}
	return starlark.String(data), nil
}

// write replaces a file's content, creating it and its directories as
// needed and keeping an existing file's mode.
func (r *repo) write(th *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var path, content string
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "path", &path, "content", &content); err != nil {
		return nil, err
	}
	return starlark.None, r.writeFile(th, b.Name(), path, func(old []byte) []byte { return []byte(content) })
}

// appendFile adds text to the end of a file, creating it if needed.
func (r *repo) appendFile(th *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var path, content string
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "path", &path, "content", &content); err != nil {
		return nil, err
	}
	return starlark.None, r.writeFile(th, b.Name(), path, func(old []byte) []byte { return append(old, content...) })
}

// writeFile writes the result of edit(old content) when it differs, and
// logs which script line made the change.
func (r *repo) writeFile(th *starlark.Thread, fn, path string, edit func(old []byte) []byte) error {
	local, err := localPath(fn, path)
	if err != nil {
		return err
	}
	mode := fs.FileMode(0o644)
	old, err := r.root.ReadFile(local)
	switch {
	case err == nil:
		if st, err := r.root.Stat(local); err == nil {
			mode = st.Mode().Perm()
		}
	case errors.Is(err, fs.ErrNotExist):
		old = nil
	default:
		return fmt.Errorf("%s: %s: %w", fn, path, unwrapPath(err))
	}
	data := edit(old)
	if old != nil && string(old) == string(data) {
		return nil
	}
	if dir := filepath.Dir(local); dir != "." {
		if err := r.root.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("%s: %s: %w", fn, path, unwrapPath(err))
		}
	}
	if err := r.root.WriteFile(local, data, mode); err != nil {
		return fmt.Errorf("%s: %s: %w", fn, path, unwrapPath(err))
	}
	fmt.Fprintf(r.log, "starlark: %s %s (%s)\n", strings.TrimPrefix(fn, "repo."), path, caller(th))
	return nil
}

// exists reports whether any of the paths exist.
func (r *repo) exists(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	paths, err := stringArgs(b, args, kwargs)
	if err != nil {
		return nil, err
	}
	for _, p := range paths {
		local, err := localPath(b.Name(), p)
		if err != nil {
			return nil, err
		}
		if _, err := r.root.Lstat(local); err == nil {
			return starlark.True, nil
		}
	}
	return starlark.False, nil
}

// delete removes files and directories; missing ones are skipped.
func (r *repo) delete(th *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	paths, err := stringArgs(b, args, kwargs)
	if err != nil {
		return nil, err
	}
	for _, p := range paths {
		local, err := localPath(b.Name(), p)
		if err != nil {
			return nil, err
		}
		if _, err := r.root.Lstat(local); err != nil {
			continue
		}
		if err := r.root.RemoveAll(local); err != nil {
			return nil, fmt.Errorf("%s: %s: %w", b.Name(), p, unwrapPath(err))
		}
		fmt.Fprintf(r.log, "starlark: delete %s (%s)\n", p, caller(th))
	}
	return starlark.None, nil
}

// glob returns the files matching any pattern (with ** for any depth),
// minus those matching exclude, sorted. .git is never included.
func (r *repo) glob(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var exclude *starlark.List
	var rest []starlark.Tuple
	for _, kv := range kwargs {
		if string(kv[0].(starlark.String)) == "exclude" {
			l, ok := kv[1].(*starlark.List)
			if !ok {
				return nil, fmt.Errorf("%s: exclude must be a list", b.Name())
			}
			exclude = l
			continue
		}
		rest = append(rest, kv)
	}
	patterns, err := stringArgs(b, args, rest)
	if err != nil {
		return nil, err
	}
	var excl []string
	if exclude != nil {
		for i := range exclude.Len() {
			s, ok := starlark.AsString(exclude.Index(i))
			if !ok {
				return nil, fmt.Errorf("%s: exclude must be a list of strings", b.Name())
			}
			excl = append(excl, s)
		}
	}
	for _, p := range append(append([]string{}, patterns...), excl...) {
		if !doublestar.ValidatePattern(p) {
			return nil, fmt.Errorf("%s: invalid pattern %q", b.Name(), p)
		}
	}
	seen := map[string]bool{}
	fsys := r.root.FS()
	err = fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		for _, x := range excl {
			if ok, _ := doublestar.Match(x, p); ok {
				return nil
			}
		}
		for _, pat := range patterns {
			if ok, _ := doublestar.Match(pat, p); ok {
				seen[p] = true
				break
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", b.Name(), err)
	}
	out := make([]string, 0, len(seen))
	for p := range seen {
		out = append(out, p)
	}
	sort.Strings(out)
	return toStarlark(out)
}

// setVar records a variable for the PR templates.
func (r *repo) setVar(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var name string
	var v starlark.Value
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "name", &name, "value", &v); err != nil {
		return nil, err
	}
	if !validName(name) {
		return nil, fmt.Errorf("%s: %q is not a valid variable name", b.Name(), name)
	}
	val, err := fromStarlark(v)
	if err != nil {
		return nil, fmt.Errorf("%s: %s: %w", b.Name(), name, err)
	}
	if _, ok := val.(map[string]any); ok {
		return nil, fmt.Errorf("%s: %s: a variable is a string, number, boolean or list", b.Name(), name)
	}
	r.vars[name] = val
	return starlark.None, nil
}

// stringArgs unpacks positional string arguments, refusing keywords.
func stringArgs(b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) ([]string, error) {
	if len(kwargs) > 0 {
		return nil, fmt.Errorf("%s: unexpected keyword argument %s", b.Name(), kwargs[0][0])
	}
	if len(args) == 0 {
		return nil, fmt.Errorf("%s: needs at least one path", b.Name())
	}
	out := make([]string, len(args))
	for i, a := range args {
		s, ok := starlark.AsString(a)
		if !ok {
			return nil, fmt.Errorf("%s: argument %d is %s, want string", b.Name(), i+1, a.Type())
		}
		out[i] = s
	}
	return out, nil
}

// unwrapPath strips the os.Root path prefix from an error so messages
// show the script's path, not the clone's location on disk.
func unwrapPath(err error) error {
	var pe *fs.PathError
	if errors.As(err, &pe) {
		return pe.Err
	}
	return err
}
