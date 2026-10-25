// Package runner executes a campaign's change script in a workspace, with
// a timeout, the configured shell and environment, and captured output.
//
// Scripts run in their own process group and are not canceled when the
// parent context is, so Ctrl-C lets in-flight scripts finish or hit their
// timeout instead of leaving a repo half-changed.
package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Spec says what to run.
type Spec struct {
	// Script is an inline script, run as `<shell> -c <script>`.
	Script string
	// File is a script file, run as `<shell> <file>`.
	File string
	// Argv runs a command directly, without a shell (used by exec).
	Argv []string
	// Shell defaults to bash.
	Shell string
	// Env is added to the environment.
	Env map[string]string
	// Timeout of zero means no timeout.
	Timeout time.Duration
}

// Result is what a run produced.
type Result struct {
	ExitCode int
	Stdout   []byte
	Stderr   []byte
	// CommitMessage is the override the script wrote to
	// $FLEET_OUTPUT/commit-message, if any.
	CommitMessage string
	Duration      time.Duration
}

// ExitError is returned when the command exits non-zero or times out.
type ExitError struct {
	Code     int
	TimedOut bool
	Timeout  time.Duration
	Tail     string
}

// Error describes the failure with the last lines of output.
func (e *ExitError) Error() string {
	msg := fmt.Sprintf("exited with status %d", e.Code)
	if e.TimedOut {
		msg = fmt.Sprintf("timed out after %s", e.Timeout)
	}
	if e.Tail != "" {
		msg += ": " + e.Tail
	}
	return msg
}

// tailLines is how many output lines an ExitError keeps.
const tailLines = 5

// Run executes spec in dir. env is added after spec.Env, and every byte of
// output is also copied to log when it is not nil.
func Run(ctx context.Context, dir string, spec Spec, env []string, log io.Writer) (Result, error) {
	outDir, err := os.MkdirTemp("", "gh-fleet-output-")
	if err != nil {
		return Result{}, err
	}
	defer func() { _ = os.RemoveAll(outDir) }()

	argv, err := spec.argv()
	if err != nil {
		return Result{}, err
	}
	// Detach from cancellation so Ctrl-C does not kill the script; the
	// timeout still applies.
	runCtx := context.WithoutCancel(ctx)
	if spec.Timeout > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(runCtx, spec.Timeout)
		defer cancel()
	}
	cmd := exec.CommandContext(runCtx, argv[0], argv[1:]...) //nolint:gosec // running the user's script is the point
	cmd.Dir = dir
	cmd.Env = os.Environ()
	for k, v := range spec.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	cmd.Env = append(cmd.Env, env...)
	cmd.Env = append(cmd.Env, "FLEET_OUTPUT="+outDir)
	cmd.Stdin = nil
	setProcessGroup(cmd)
	cmd.WaitDelay = 5 * time.Second

	var stdout, stderr bytes.Buffer
	if log == nil {
		log = io.Discard
	}
	// stdout and stderr are copied by separate goroutines.
	shared := &lockedWriter{w: log}
	cmd.Stdout = io.MultiWriter(&stdout, shared)
	cmd.Stderr = io.MultiWriter(&stderr, shared)

	start := time.Now()
	runErr := cmd.Run()
	res := Result{Stdout: stdout.Bytes(), Stderr: stderr.Bytes(), Duration: time.Since(start)}
	if b, err := os.ReadFile(filepath.Join(outDir, "commit-message")); err == nil {
		res.CommitMessage = strings.TrimSpace(string(b))
	}
	if runErr == nil {
		return res, nil
	}
	ee := &ExitError{Code: -1, Tail: tail(stderr.Bytes(), stdout.Bytes())}
	var xe *exec.ExitError
	if errors.As(runErr, &xe) {
		ee.Code = xe.ExitCode()
	}
	if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
		ee.TimedOut, ee.Timeout = true, spec.Timeout
	} else if ee.Code == -1 {
		return res, runErr
	}
	res.ExitCode = ee.Code
	return res, ee
}

// lockedWriter serializes writes to w.
type lockedWriter struct {
	mu sync.Mutex
	w  io.Writer
}

// Write writes p to the underlying writer under the lock.
func (l *lockedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}

// argv builds the command line for spec.
func (s Spec) argv() ([]string, error) {
	shell := s.Shell
	if shell == "" {
		shell = "bash"
	}
	switch {
	case len(s.Argv) > 0:
		return s.Argv, nil
	case s.File != "":
		return []string{shell, s.File}, nil
	case s.Script != "":
		return []string{shell, "-c", s.Script}, nil
	}
	return nil, errors.New("nothing to run")
}

// tail returns the last lines of the first non-empty output.
func tail(outputs ...[]byte) string {
	for _, o := range outputs {
		s := strings.TrimSpace(string(o))
		if s == "" {
			continue
		}
		lines := strings.Split(s, "\n")
		if len(lines) > tailLines {
			lines = lines[len(lines)-tailLines:]
		}
		return strings.Join(lines, " | ")
	}
	return ""
}
