package runner

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunInlineFileAndArgv(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	var log bytes.Buffer
	res, err := Run(context.Background(), dir, Spec{Script: `echo "$A $B" ; echo err >&2; pwd; echo msg > "$FLEET_OUTPUT/commit-message"`, Env: map[string]string{"A": "a"}}, []string{"B=b"}, &log)
	if err != nil {
		t.Fatal(err)
	}
	out := string(res.Stdout)
	if !strings.HasPrefix(out, "a b\n") || !strings.Contains(out, filepath.Base(dir)) || string(res.Stderr) != "err\n" || res.CommitMessage != "msg" {
		t.Fatalf("res = %+v", res)
	}
	if !strings.Contains(log.String(), "a b") || !strings.Contains(log.String(), "err") {
		t.Fatalf("log = %q", log.String())
	}

	script := filepath.Join(t.TempDir(), "s.sh")
	if err := os.WriteFile(script, []byte("echo from-file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if res, err := Run(context.Background(), dir, Spec{File: script, Shell: "sh"}, nil, nil); err != nil || string(res.Stdout) != "from-file\n" {
		t.Fatalf("file: %q, %v", res.Stdout, err)
	}
	if res, err := Run(context.Background(), dir, Spec{Argv: []string{"echo", "argv"}}, nil, nil); err != nil || string(res.Stdout) != "argv\n" {
		t.Fatalf("argv: %q, %v", res.Stdout, err)
	}
	if _, err := Run(context.Background(), dir, Spec{}, nil, nil); err == nil {
		t.Fatal("expected nothing to run")
	}
	if _, err := Run(context.Background(), dir, Spec{Argv: []string{"/nonexistent/binary"}}, nil, nil); err == nil {
		t.Fatal("expected start error")
	}
}

func TestRunExitAndTimeout(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	_, err := Run(context.Background(), dir, Spec{Script: "for i in 1 2 3 4 5 6 7; do echo line$i; done >&2; exit 4"}, nil, nil)
	var ee *ExitError
	if !errors.As(err, &ee) || ee.Code != 4 || ee.Tail != "line3 | line4 | line5 | line6 | line7" {
		t.Fatalf("err = %#v", err)
	}
	if !strings.HasPrefix(err.Error(), "exited with status 4: line3") {
		t.Fatal(err.Error())
	}

	start := time.Now()
	_, err = Run(context.Background(), dir, Spec{Script: "sleep 30 & sleep 30", Timeout: 200 * time.Millisecond}, nil, nil)
	if !errors.As(err, &ee) || !ee.TimedOut || !strings.Contains(err.Error(), "timed out after 200ms") {
		t.Fatalf("err = %v", err)
	}
	if time.Since(start) > 10*time.Second {
		t.Fatal("timeout did not kill the process group")
	}
}

func TestRunSurvivesCancellation(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	res, err := Run(ctx, t.TempDir(), Spec{Script: "sleep 0.1; echo finished"}, nil, nil)
	if err != nil || string(res.Stdout) != "finished\n" {
		t.Fatalf("canceled parent killed the script: %q, %v", res.Stdout, err)
	}
}

func TestTail(t *testing.T) {
	t.Parallel()
	if tail(nil, []byte("  \n"), []byte("x\n")) != "x" || tail() != "" {
		t.Fatal("tail")
	}
	if (&ExitError{Code: 1}).Error() != "exited with status 1" {
		t.Fatal("message without tail")
	}
}

func TestScriptVars(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	res, err := Run(context.Background(), dir, Spec{Script: `printf '# comment\n\nOLD_GO=1.22\nurl=https://x/?a=b\r\n' > "$FLEET_OUTPUT/vars"`}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Vars["OLD_GO"] != "1.22" || res.Vars["url"] != "https://x/?a=b" || len(res.Vars) != 2 {
		t.Fatalf("vars = %v", res.Vars)
	}
	_, err = Run(context.Background(), dir, Spec{Script: `echo 'bad line' > "$FLEET_OUTPUT/vars"`}, nil, nil)
	if err == nil || !strings.Contains(err.Error(), `$FLEET_OUTPUT/vars line 1: want NAME=value, got "bad line"`) {
		t.Fatalf("err = %v", err)
	}
	if res, err := Run(context.Background(), dir, Spec{Script: "true"}, nil, nil); err != nil || res.Vars != nil {
		t.Fatalf("no file: %v %v", res.Vars, err)
	}
	if _, err := Run(context.Background(), dir, Spec{Script: `mkdir "$FLEET_OUTPUT/vars"`}, nil, nil); err == nil {
		t.Fatal("unreadable vars file accepted")
	}
}
