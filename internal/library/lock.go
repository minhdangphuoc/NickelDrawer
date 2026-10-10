package library

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"syscall"

	"koboapps/internal/i18n"
)

// Acquire takes the single-run lock (data/lock holds the pid). A lock whose process is gone
// is stale and is taken over. The file is closed at once; nothing stays open on /mnt/onboard.
func Acquire(paths Paths) (release func(), err error) {
	if err := os.MkdirAll(paths.Data(), 0o755); err != nil {
		return nil, i18n.NewError("msg.err_mkdir", "path", paths.Data()).WithCause(err)
	}
	for attempt := 0; attempt < 2; attempt++ {
		file, err := os.OpenFile(paths.Lock(), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err == nil {
			_, writeErr := file.WriteString(strconv.Itoa(os.Getpid()))
			if err := errors.Join(writeErr, file.Close()); err != nil {
				_ = os.Remove(paths.Lock())
				return nil, i18n.NewError("msg.err_write", "path", paths.Lock()).WithCause(err)
			}
			_ = os.Remove(paths.Cancel()) // a stop request from before this run does not apply to it
			return func() {
				_ = os.Remove(paths.Lock())
				_ = os.Remove(paths.Cancel())
			}, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, i18n.NewError("msg.err_write", "path", paths.Lock()).WithCause(err)
		}
		if IsRunning(paths) {
			return nil, i18n.NewError("msg.busy")
		}
		_ = os.Remove(paths.Lock())
	}
	return nil, i18n.NewError("msg.busy")
}

// IsRunning reports whether a live process holds the lock.
func IsRunning(paths Paths) bool {
	content, err := os.ReadFile(paths.Lock())
	if err != nil {
		return false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(content)))
	if err != nil || pid <= 0 {
		return false
	}
	err = syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}
