package star

import (
	"errors"
	"io"
	"os"

	"go.starlark.net/starlark"
)

// campaignDir is the predeclared campaign value: read-only access to the
// files next to the campaign file, for changes that copy or template them
// into each repo. Like repo, it cannot reach outside its directory.
type campaignDir struct {
	dir string
}

// campaignMethods are the repo methods campaign shares; none write.
var campaignMethods = []string{"exists", "glob", "read"}

// String describes the value.
func (c *campaignDir) String() string { return "<campaign>" }

// Type names the value's type.
func (c *campaignDir) Type() string { return "campaign" }

// Freeze does nothing: campaign has no mutable state.
func (c *campaignDir) Freeze() {}

// Truth reports campaign as true.
func (c *campaignDir) Truth() starlark.Bool { return true }

// Hash refuses to hash campaign.
func (c *campaignDir) Hash() (uint32, error) { return 0, errors.New("unhashable type: campaign") }

// Attr returns one of campaign's methods. Each call opens the directory
// afresh, so the value holds no file handles between runs.
func (c *campaignDir) Attr(name string) (starlark.Value, error) {
	for _, m := range campaignMethods {
		if m != name {
			continue
		}
		fn := methods[name]
		return starlark.NewBuiltin("campaign."+name, func(th *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			root, err := os.OpenRoot(c.dir)
			if err != nil {
				return nil, err
			}
			defer func() { _ = root.Close() }()
			return fn(&repo{root: root, log: io.Discard}, th, b, args, kwargs)
		}), nil
	}
	return nil, nil
}

// AttrNames lists campaign's methods.
func (c *campaignDir) AttrNames() []string { return campaignMethods }
