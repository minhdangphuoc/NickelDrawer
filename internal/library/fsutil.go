package library

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"koboapps/internal/i18n"
)

// writeFileAtomic writes a temp file next to target and renames it, so readers never see a
// half-written file and nothing stays open on /mnt/onboard.
func writeFileAtomic(target string, content []byte) error {
	return writeAtomic(target, func(temp *os.File) error {
		_, err := temp.Write(content)
		return err
	})
}

func writeAtomic(target string, fill func(temp *os.File) error) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return i18n.NewError("msg.err_mkdir", "path", filepath.Dir(target)).WithCause(err)
	}
	temp, err := os.CreateTemp(filepath.Dir(target), ".library-*.tmp")
	if err != nil {
		return i18n.NewError("msg.err_tempfile", "path", target).WithCause(err)
	}
	defer os.Remove(temp.Name())
	fillErr := fill(temp)
	syncErr := temp.Sync()
	closeErr := temp.Close()
	if err := errors.Join(fillErr, syncErr, closeErr); err != nil {
		return i18n.NewError("msg.err_write", "path", target).WithCause(err)
	}
	_ = os.Chmod(temp.Name(), 0o644)
	if err := os.Rename(temp.Name(), target); err != nil {
		return i18n.NewError("msg.err_write", "path", target).WithCause(err)
	}
	return nil
}

// copyFileAtomic copies source to target through a temp file next to target.
func copyFileAtomic(source, target string) error {
	return writeAtomic(target, func(temp *os.File) error {
		input, err := os.Open(source)
		if err != nil {
			return err
		}
		defer input.Close()
		_, err = io.Copy(temp, input)
		return err
	})
}

// moveFile renames, and copies then removes when the two folders are on different file systems.
func moveFile(source, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return i18n.NewError("msg.err_mkdir", "path", filepath.Dir(target)).WithCause(err)
	}
	if err := os.Rename(source, target); err == nil {
		return nil
	}
	if err := copyFileAtomic(source, target); err != nil {
		return err
	}
	if err := os.Remove(source); err != nil {
		return i18n.NewError("msg.err_remove", "path", source).WithCause(err)
	}
	return nil
}

// ensureFreeSpace fails when the file system holding dir has less than needed bytes free.
func ensureFreeSpace(dir string, needed int64) error {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(dir, &stat); err != nil {
		return i18n.NewError("msg.err_read", "path", dir).WithCause(err)
	}
	free := int64(stat.Bavail) * int64(stat.Bsize)
	if free < needed {
		return i18n.NewError("msg.err_no_space", "path", dir, "need", megabytes(needed), "free", megabytes(free))
	}
	return nil
}

// Logger appends timestamped lines to data/log.txt, reopening per line; a nil Logger discards.
type Logger struct{ path string }

const maxLogBytes = 256 * 1024

// NewLogger truncates an oversized log and returns a logger appending to path.
func NewLogger(path string) *Logger {
	if info, err := os.Stat(path); err == nil && info.Size() > maxLogBytes {
		_ = os.Truncate(path, 0)
	}
	return &Logger{path: path}
}

// Printf logs one line; failures to log are ignored because logging must never stop the work.
func (logger *Logger) Printf(format string, args ...any) {
	if logger == nil {
		return
	}
	file, err := os.OpenFile(logger.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer file.Close()
	line := time.Now().Format("2006-01-02 15:04:05") + " " + fmt.Sprintf(format, args...) + "\n"
	_, _ = file.WriteString(line)
}

// readSmallFile returns the trimmed text of a tiny state file.
func readSmallFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	return strings.TrimSpace(string(content)), err
}

func fileSize(relative string) string {
	info, err := os.Lstat(absoluteOf(relative))
	if err != nil {
		return "?"
	}
	return megabytes(info.Size())
}
