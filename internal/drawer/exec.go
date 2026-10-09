package drawer

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"
)

const (
	maxOutputBytes = 64 * 1024
	killGrace      = 500 * time.Millisecond
	scriptPath     = "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
)

// ScriptOutput is what a finished script left behind.
type ScriptOutput struct {
	Stdout   string
	Stderr   string
	ExitCode int
	TimedOut bool
	Err      error // the script could not be started
}

// ScriptRunner runs one script; tests replace it.
type ScriptRunner func(ctx context.Context, script string) ScriptOutput

// scriptEnvironment is deliberately small: NickelMenu starts scripts from Nickel, not from a
// login shell, so scripts must not depend on anything else. extra carries the few protocol
// variables a drawer event passes (NICKELDRAWER_VALUE).
func scriptEnvironment(extra []string) []string {
	return append([]string{"PATH=" + scriptPath, "HOME=/"}, extra...)
}

// ExecScript runs script from "/" with the minimal environment.
func ExecScript(ctx context.Context, script string) ScriptOutput {
	return ExecScriptWithEnv(ctx, script, nil)
}

// ExecScriptWithEnv runs script from "/" with the minimal environment plus extra. The whole
// process group is killed on timeout, so a script's background children cannot keep the pipe
// open.
func ExecScriptWithEnv(ctx context.Context, script string, extra []string) ScriptOutput {
	command := exec.CommandContext(ctx, script)
	command.Dir = "/"
	command.Env = scriptEnvironment(extra)
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Cancel = func() error { return syscall.Kill(-command.Process.Pid, syscall.SIGKILL) }
	command.WaitDelay = killGrace
	stdout, stderr := &limitedBuffer{}, &limitedBuffer{}
	command.Stdout, command.Stderr = stdout, stderr
	err := command.Run()
	output := ScriptOutput{Stdout: stdout.String(), Stderr: stderr.String()}
	if ctx.Err() != nil {
		output.TimedOut = true
		output.ExitCode = -1
		return output
	}
	var exitErr *exec.ExitError
	switch {
	case err == nil:
	case errors.As(err, &exitErr):
		output.ExitCode = exitErr.ExitCode()
	default:
		output.Err = err
		output.ExitCode = -1
	}
	return output
}

// StartDetached starts script in its own session with no output and does not wait for it.
func StartDetached(script string) error {
	return StartDetachedWithEnv(script, nil)
}

// StartDetachedWithEnv is StartDetached with extra environment variables.
func StartDetachedWithEnv(script string, extra []string) error {
	command := exec.Command(script)
	command.Dir = "/"
	command.Env = scriptEnvironment(extra)
	command.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	devNull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer devNull.Close()
	command.Stdin, command.Stdout, command.Stderr = devNull, devNull, devNull
	if err := command.Start(); err != nil {
		return err
	}
	go func() { _ = command.Wait() }() // reap the child while appsctl is still alive
	return nil
}

// limitedBuffer keeps the first maxOutputBytes and drops the rest, so a runaway script
// cannot fill memory.
type limitedBuffer struct{ buffer bytes.Buffer }

func (b *limitedBuffer) Write(data []byte) (int, error) {
	if room := maxOutputBytes - b.buffer.Len(); room > 0 {
		b.buffer.Write(data[:min(room, len(data))])
	}
	return len(data), nil
}

func (b *limitedBuffer) String() string { return b.buffer.String() }
