package wallpaper

import (
	"fmt"
	"io"
	"os"
	"time"
)

const maxLogBytes = 256 * 1024

// Logger writes timestamped lines; a nil Logger discards them.
type Logger struct{ out io.Writer }

// NewLogger wraps a writer.
func NewLogger(out io.Writer) *Logger { return &Logger{out: out} }

// NewLogFile appends to path, reopening for every line so no handle stays open on /mnt/onboard.
func NewLogFile(path string) (*Logger, error) {
	if info, err := os.Stat(path); err == nil && info.Size() > maxLogBytes {
		if err := os.Truncate(path, 0); err != nil {
			return nil, err
		}
	}
	return &Logger{out: appendWriter{path: path}}, nil
}

type appendWriter struct{ path string }

func (writer appendWriter) Write(content []byte) (int, error) {
	file, err := os.OpenFile(writer.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return 0, err
	}
	written, writeErr := file.Write(content)
	if closeErr := file.Close(); writeErr == nil {
		writeErr = closeErr
	}
	return written, writeErr
}

// Printf logs one line.
func (logger *Logger) Printf(format string, args ...any) {
	if logger == nil || logger.out == nil {
		return
	}
	fmt.Fprintf(logger.out, "%s %s\n", time.Now().Format("2006-01-02 15:04:05"), fmt.Sprintf(format, args...))
}
