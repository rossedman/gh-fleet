package gitops

import "testing"

func TestSubcommand(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"plain", []string{"clone", "--quiet", "url"}, "clone"},
		{"config option", []string{"-c", "color.ui=never", "diff", "--cached"}, "diff"},
		{"dir option", []string{"-C", "/tmp", "status"}, "status"},
		{"leading flag", []string{"--no-pager", "log"}, "log"},
		{"only options", []string{"-c", "a=b"}, "?"},
		{"empty", nil, "?"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := subcommand(tc.args); got != tc.want {
				t.Fatalf("subcommand(%q) = %q, want %q", tc.args, got, tc.want)
			}
		})
	}
}
