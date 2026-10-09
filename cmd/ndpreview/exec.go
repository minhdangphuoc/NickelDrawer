package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"koboapps/internal/manifest"
)

const (
	maxOutputBytes   = 16 << 10
	minExecTimeout   = time.Second
	maxExecTimeout   = 10 * time.Second
	execDataDirName  = "data"
	shellInterpreter = "sh"
)

// actionRunner runs an app's action script on this computer inside a temporary folder that
// stands in for the Kobo storage. Scripts that hard-code /mnt/onboard paths will fail here and
// say so in their output; that is a signal, not a bug of the preview.
type actionRunner struct {
	root string
}

func newActionRunner() (*actionRunner, error) {
	root, err := os.MkdirTemp("", "ndpreview-root-")
	if err != nil {
		return nil, fmt.Errorf("cannot create the temporary fake root for --exec: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(root, execDataDirName), 0o755); err != nil {
		return nil, fmt.Errorf("cannot create %s in the fake root: %w", execDataDirName, err)
	}
	return &actionRunner{root: root}, nil
}

func (r *actionRunner) close() { os.RemoveAll(r.root) }

type execResult struct {
	Exit     int    `json:"exit"`
	Output   string `json:"output"`
	TimedOut bool   `json:"timedOut"`
	Error    string `json:"error,omitempty"`
	Root     string `json:"root"`
}

func (r *actionRunner) run(app manifest.App, action manifest.Action, eventLine string) execResult {
	result := execResult{Root: r.root}
	script, err := scriptPath(app.Dir, action.Run)
	if err != nil {
		result.Exit, result.Error = -1, err.Error()
		return result
	}
	timeout := time.Duration(action.Timeout()) * time.Millisecond
	timeout = max(minExecTimeout, min(timeout, maxExecTimeout))
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	command := exec.CommandContext(ctx, shellInterpreter, script)
	command.Dir = r.root
	command.Env = []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + r.root,
		"KOBO_ROOT=" + r.root,
		"ND_PREVIEW=1",
		"ND_APP_DIR=" + app.Dir,
		"ND_DATA_DIR=" + filepath.Join(r.root, execDataDirName),
		"ND_EVENT=" + eventLine,
	}
	var output limitedBuffer
	command.Stdout, command.Stderr = &output, &output
	runErr := command.Run()
	result.Output = output.String()
	result.TimedOut = ctx.Err() == context.DeadlineExceeded
	if exitErr, isExit := runErr.(*exec.ExitError); isExit {
		result.Exit = exitErr.ExitCode()
	} else if runErr != nil {
		result.Exit, result.Error = -1, runErr.Error()
	}
	return result
}

// scriptPath resolves run relative to the app folder and refuses to leave it.
func scriptPath(appDir, run string) (string, error) {
	path := filepath.Join(appDir, run)
	relative, err := filepath.Rel(appDir, path)
	if err != nil || strings.HasPrefix(relative, "..") {
		return "", fmt.Errorf("action script %q is outside the app folder", run)
	}
	if _, statErr := os.Stat(path); statErr != nil {
		return "", fmt.Errorf("action script %s is missing: %w", relative, statErr)
	}
	return path, nil
}

// limitedBuffer keeps the first maxOutputBytes of output.
type limitedBuffer struct{ buffer bytes.Buffer }

func (l *limitedBuffer) Write(data []byte) (int, error) {
	if room := maxOutputBytes - l.buffer.Len(); room > 0 {
		l.buffer.Write(data[:min(room, len(data))])
	}
	return len(data), nil
}

func (l *limitedBuffer) String() string { return l.buffer.String() }
