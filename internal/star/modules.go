package star

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
	"go.yaml.in/yaml/v3"
	"golang.org/x/mod/semver"
)

// validName matches variable names, as in campaign vars.
var validName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`).MatchString

// builtin is the signature of a Starlark builtin's Go function.
type builtin = func(*starlark.Thread, *starlark.Builtin, starlark.Tuple, []starlark.Tuple) (starlark.Value, error)

// module builds a Starlark module from named functions.
func newModule(name string, fns map[string]builtin) *starlarkstruct.Module {
	m := &starlarkstruct.Module{Name: name, Members: starlark.StringDict{}}
	for k, fn := range fns {
		m.Members[k] = starlark.NewBuiltin(name+"."+k, fn)
	}
	return m
}

// yamlModule decodes and encodes YAML. Mapping order is kept both ways.
var yamlModule = newModule("yaml", map[string]builtin{
	"decode": func(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var src string
		if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &src); err != nil {
			return nil, err
		}
		var n yaml.Node
		if err := yaml.Unmarshal([]byte(src), &n); err != nil {
			return nil, fmt.Errorf("%s: %w", b.Name(), err)
		}
		if len(n.Content) == 0 {
			return starlark.None, nil
		}
		return fromYAML(n.Content[0], 0)
	},
	"encode": func(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var v starlark.Value
		if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &v); err != nil {
			return nil, err
		}
		n, err := toYAML(v)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", b.Name(), err)
		}
		var sb strings.Builder
		enc := yaml.NewEncoder(&sb)
		enc.SetIndent(2)
		if err := enc.Encode(n); err != nil {
			return nil, fmt.Errorf("%s: %w", b.Name(), err)
		}
		return starlark.String(sb.String()), nil
	},
})

// maxYAMLDepth bounds alias expansion and nesting in yaml.decode.
const maxYAMLDepth = 100

// fromYAML converts a YAML node to a Starlark value.
func fromYAML(n *yaml.Node, depth int) (starlark.Value, error) {
	if depth > maxYAMLDepth {
		return nil, errors.New("yaml.decode: document nested too deeply")
	}
	switch n.Kind {
	case yaml.AliasNode:
		return fromYAML(n.Alias, depth+1)
	case yaml.MappingNode:
		d := starlark.NewDict(len(n.Content) / 2)
		for i := 0; i+1 < len(n.Content); i += 2 {
			k, err := fromYAML(n.Content[i], depth+1)
			if err != nil {
				return nil, err
			}
			v, err := fromYAML(n.Content[i+1], depth+1)
			if err != nil {
				return nil, err
			}
			if err := d.SetKey(k, v); err != nil {
				return nil, err
			}
		}
		return d, nil
	case yaml.SequenceNode:
		items := make([]starlark.Value, 0, len(n.Content))
		for _, c := range n.Content {
			v, err := fromYAML(c, depth+1)
			if err != nil {
				return nil, err
			}
			items = append(items, v)
		}
		return starlark.NewList(items), nil
	case yaml.ScalarNode:
		var v any
		if err := n.Decode(&v); err != nil {
			return nil, err
		}
		return toStarlark(v)
	}
	return starlark.None, nil
}

// toYAML converts a Starlark value to a YAML node, keeping dict order.
func toYAML(v starlark.Value) (*yaml.Node, error) {
	switch t := v.(type) {
	case *starlark.Dict:
		n := &yaml.Node{Kind: yaml.MappingNode}
		for _, item := range t.Items() {
			k, err := toYAML(item[0])
			if err != nil {
				return nil, err
			}
			val, err := toYAML(item[1])
			if err != nil {
				return nil, err
			}
			n.Content = append(n.Content, k, val)
		}
		return n, nil
	case *starlark.List, starlark.Tuple:
		n := &yaml.Node{Kind: yaml.SequenceNode}
		iter := starlark.Iterate(v)
		defer iter.Done()
		var x starlark.Value
		for iter.Next(&x) {
			c, err := toYAML(x)
			if err != nil {
				return nil, err
			}
			n.Content = append(n.Content, c)
		}
		return n, nil
	}
	g, err := fromStarlark(v)
	if err != nil {
		return nil, err
	}
	n := &yaml.Node{}
	if err := n.Encode(g); err != nil {
		return nil, err
	}
	return n, nil
}

// canonical prefixes v so golang.org/x/mod/semver accepts it.
func canonical(v string) string {
	if strings.HasPrefix(v, "v") {
		return v
	}
	return "v" + v
}

// semverArg unpacks one or two version arguments and checks them.
func semverArgs(b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple, n int) ([]string, error) {
	vals := make([]string, n)
	ptrs := make([]any, n)
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, n, ptrs...); err != nil {
		return nil, err
	}
	for i, v := range vals {
		vals[i] = canonical(v)
		if !semver.IsValid(vals[i]) {
			return nil, fmt.Errorf("%s: %q is not a semantic version", b.Name(), v)
		}
	}
	return vals, nil
}

// semverModule compares semantic versions; a leading "v" is optional.
var semverModule = newModule("semver", map[string]builtin{
	"valid": func(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var v string
		if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &v); err != nil {
			return nil, err
		}
		return starlark.Bool(semver.IsValid(canonical(v))), nil
	},
	"compare": func(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		v, err := semverArgs(b, args, kwargs, 2)
		if err != nil {
			return nil, err
		}
		return starlark.MakeInt(semver.Compare(v[0], v[1])), nil
	},
	"less": func(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		v, err := semverArgs(b, args, kwargs, 2)
		if err != nil {
			return nil, err
		}
		return starlark.Bool(semver.Compare(v[0], v[1]) < 0), nil
	},
	"major": func(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		v, err := semverArgs(b, args, kwargs, 1)
		if err != nil {
			return nil, err
		}
		return starlark.String(strings.TrimPrefix(semver.Major(v[0]), "v")), nil
	},
	"major_minor": func(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		v, err := semverArgs(b, args, kwargs, 1)
		if err != nil {
			return nil, err
		}
		return starlark.String(strings.TrimPrefix(semver.MajorMinor(v[0]), "v")), nil
	},
	"lowers": func(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var current, target string
		if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 2, &current, &target); err != nil {
			return nil, err
		}
		return starlark.Bool(lowers(current, target)), nil
	},
	"max": func(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if len(args) == 0 || len(kwargs) > 0 {
			return nil, fmt.Errorf("%s: needs one or more versions", b.Name())
		}
		var best, bestRaw string
		for i, a := range args {
			s, ok := starlark.AsString(a)
			if !ok || !semver.IsValid(canonical(s)) {
				return nil, fmt.Errorf("%s: argument %d is not a semantic version", b.Name(), i+1)
			}
			if best == "" || semver.Compare(canonical(s), best) > 0 {
				best, bestRaw = canonical(s), s
			}
		}
		return starlark.String(bestRaw), nil
	},
})

// versionNumber finds the first dotted number in a version string, such
// as 13.17 in "^13.17" or 3.12 in "python3.12".
var versionNumber = regexp.MustCompile(`\d+(?:\.\d+)*`)

// lowers reports whether moving from current to target would lower a
// version. Only the first dotted number in each string counts, so
// constraints, prefixes and suffixes ("^8.3", "go1.22", "22-alpine")
// compare by their versions. It is false whenever either string has no
// number, such as a branch name, or is a commit SHA or digest, whose
// digits mean nothing.
func lowers(current, target string) bool {
	if hashLike(current) || hashLike(target) {
		return false
	}
	c, t := versionNumber.FindString(current), versionNumber.FindString(target)
	if c == "" || t == "" {
		return false
	}
	cs, ts := strings.Split(c, "."), strings.Split(t, ".")
	for i := range max(len(cs), len(ts)) {
		var a, b int
		if i < len(cs) {
			a, _ = strconv.Atoi(cs[i])
		}
		if i < len(ts) {
			b, _ = strconv.Atoi(ts[i])
		}
		if a != b {
			return b < a
		}
	}
	return false
}

// hexRef matches an abbreviated or full commit SHA.
var hexRef = regexp.MustCompile(`^[0-9a-f]{7,64}$`)

// hashLike reports whether s is a commit SHA or a content digest rather
// than a version.
func hashLike(s string) bool {
	s = strings.TrimSpace(s)
	return strings.Contains(s, "sha256:") || strings.Contains(s, "sha512") || hexRef.MatchString(s) && strings.ContainsAny(s, "abcdef")
}

// regexps caches compiled patterns; they are immutable and safe to share.
var regexps sync.Map

// compileRE returns a compiled RE2 pattern.
func compileRE(fn, pattern string) (*regexp.Regexp, error) {
	if re, ok := regexps.Load(pattern); ok {
		return re.(*regexp.Regexp), nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", fn, err)
	}
	regexps.Store(pattern, re)
	return re, nil
}

// reArgs unpacks a pattern and a string.
func reArgs(b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (*regexp.Regexp, string, error) {
	var pattern, s string
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 2, &pattern, &s); err != nil {
		return nil, "", err
	}
	re, err := compileRE(b.Name(), pattern)
	return re, s, err
}

// reModule is regular expressions in Go's RE2 syntax. Replacements use
// $1 or ${name} for groups.
var reModule = newModule("re", map[string]builtin{
	"escape": func(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var s string
		if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &s); err != nil {
			return nil, err
		}
		return starlark.String(regexp.QuoteMeta(s)), nil
	},
	"matches": func(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		re, s, err := reArgs(b, args, kwargs)
		if err != nil {
			return nil, err
		}
		return starlark.Bool(re.MatchString(s)), nil
	},
	"search": func(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		re, s, err := reArgs(b, args, kwargs)
		if err != nil {
			return nil, err
		}
		loc := re.FindStringIndex(s)
		if loc == nil {
			return starlark.None, nil
		}
		return starlark.String(s[loc[0]:loc[1]]), nil
	},
	"groups": func(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		re, s, err := reArgs(b, args, kwargs)
		if err != nil {
			return nil, err
		}
		m := re.FindStringSubmatch(s)
		if m == nil {
			return starlark.None, nil
		}
		return toStarlark(m[1:])
	},
	"find_all": func(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		re, s, err := reArgs(b, args, kwargs)
		if err != nil {
			return nil, err
		}
		return toStarlark(re.FindAllString(s, -1))
	},
	"split": func(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		re, s, err := reArgs(b, args, kwargs)
		if err != nil {
			return nil, err
		}
		return toStarlark(re.Split(s, -1))
	},
	"sub": func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var pattern, s string
		var repl starlark.Value
		count := -1
		if err := starlark.UnpackArgs(b.Name(), args, kwargs, "pattern", &pattern, "repl", &repl, "s", &s, "count?", &count); err != nil {
			return nil, err
		}
		re, err := compileRE(b.Name(), pattern)
		if err != nil {
			return nil, err
		}
		fn, isFn := repl.(starlark.Callable)
		str, isStr := starlark.AsString(repl)
		if !isFn && !isStr {
			return nil, fmt.Errorf("%s: repl must be a string or a function, not %s", b.Name(), repl.Type())
		}
		var out strings.Builder
		last, n := 0, 0
		for _, m := range re.FindAllStringSubmatchIndex(s, -1) {
			if count >= 0 && n == count {
				break
			}
			out.WriteString(s[last:m[0]])
			if isFn {
				// A function gets the match and its groups, as re.groups
				// returns them after the match, and returns the text to use.
				groups := make([]starlark.Value, 0, len(m)/2)
				for i := 0; i < len(m); i += 2 {
					if m[i] < 0 {
						groups = append(groups, starlark.String(""))
					} else {
						groups = append(groups, starlark.String(s[m[i]:m[i+1]]))
					}
				}
				v, err := starlark.Call(thread, fn, starlark.Tuple{starlark.NewList(groups)}, nil)
				if err != nil {
					return nil, err
				}
				r, ok := starlark.AsString(v)
				if !ok {
					return nil, fmt.Errorf("%s: repl function returned %s, want string", b.Name(), v.Type())
				}
				out.WriteString(r)
			} else {
				out.Write(re.ExpandString(nil, str, s, m))
			}
			last, n = m[1], n+1
		}
		out.WriteString(s[last:])
		return starlark.String(out.String()), nil
	},
})

// toStarlark converts a Go value to a Starlark value.
func toStarlark(v any) (starlark.Value, error) {
	switch t := v.(type) {
	case nil:
		return starlark.None, nil
	case starlark.Value:
		return t, nil
	case string:
		return starlark.String(t), nil
	case bool:
		return starlark.Bool(t), nil
	case int:
		return starlark.MakeInt(t), nil
	case int64:
		return starlark.MakeInt64(t), nil
	case uint64:
		return starlark.MakeUint64(t), nil
	case float64:
		return starlark.Float(t), nil
	case *big.Int:
		return starlark.MakeBigInt(t), nil
	case []string:
		items := make([]starlark.Value, len(t))
		for i, s := range t {
			items[i] = starlark.String(s)
		}
		return starlark.NewList(items), nil
	case []any:
		items := make([]starlark.Value, len(t))
		for i, e := range t {
			x, err := toStarlark(e)
			if err != nil {
				return nil, err
			}
			items[i] = x
		}
		return starlark.NewList(items), nil
	case map[string]string:
		d := starlark.NewDict(len(t))
		for _, k := range sortedKeys(t) {
			_ = d.SetKey(starlark.String(k), starlark.String(t[k]))
		}
		return d, nil
	case map[string]any:
		d := starlark.NewDict(len(t))
		for _, k := range sortedKeys(t) {
			x, err := toStarlark(t[k])
			if err != nil {
				return nil, err
			}
			_ = d.SetKey(starlark.String(k), x)
		}
		return d, nil
	}
	return nil, fmt.Errorf("cannot convert %T to Starlark", v)
}

// fromStarlark converts a Starlark value to plain Go data.
func fromStarlark(v starlark.Value) (any, error) {
	switch t := v.(type) {
	case starlark.NoneType:
		return nil, nil
	case starlark.String:
		return string(t), nil
	case starlark.Bool:
		return bool(t), nil
	case starlark.Int:
		if i, ok := t.Int64(); ok {
			return i, nil
		}
		return t.BigInt(), nil
	case starlark.Float:
		f := float64(t)
		if math.IsInf(f, 0) || math.IsNaN(f) {
			return nil, fmt.Errorf("%s is not a finite number", strconv.FormatFloat(f, 'g', -1, 64))
		}
		return f, nil
	case *starlark.List, starlark.Tuple:
		var out []any
		iter := starlark.Iterate(v)
		defer iter.Done()
		var x starlark.Value
		for iter.Next(&x) {
			g, err := fromStarlark(x)
			if err != nil {
				return nil, err
			}
			out = append(out, g)
		}
		if out == nil {
			out = []any{}
		}
		return out, nil
	case *starlark.Dict:
		out := map[string]any{}
		for _, item := range t.Items() {
			k, ok := starlark.AsString(item[0])
			if !ok {
				return nil, fmt.Errorf("dict key %s is not a string", item[0])
			}
			g, err := fromStarlark(item[1])
			if err != nil {
				return nil, err
			}
			out[k] = g
		}
		return out, nil
	}
	return nil, fmt.Errorf("cannot use a %s here", v.Type())
}

// sortedKeys returns map keys in order, so converted dicts are stable.
func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
