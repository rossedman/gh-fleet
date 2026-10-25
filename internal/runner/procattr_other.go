//go:build !unix

package runner

import "os/exec"

// setProcessGroup is a no-op where process groups are not available; the
// command is killed directly on timeout.
func setProcessGroup(*exec.Cmd) {}
