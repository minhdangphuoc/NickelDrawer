package ghsync

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

const maxLogBytes = 512 * 1024

// Logger writes timestamped lines and never lets the token through.
type Logger struct {
	out   io.Writer
	token Token
}

func NewLogger(out io.Writer, token Token) *Logger {
	return &Logger{out: out, token: token}
}

// appendWriter opens, appends and closes on every write, so no handle on
// /mnt/onboard outlives a single log line (USB mass storage must stay ejectable).
type appendWriter struct{ path string }

// NewLogFile returns a writer for data/log.txt, starting the file over when it grew large.
func NewLogFile(logPath string) (io.Writer, error) {
	if info, err := os.Stat(logPath); err == nil && info.Size() > maxLogBytes {
		if err := os.Truncate(logPath, 0); err != nil {
			return nil, err
		}
	}
	return appendWriter{path: logPath}, nil
}

func (writer appendWriter) Write(content []byte) (int, error) {
	file, err := os.OpenFile(writer.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return 0, err
	}
	written, writeErr := file.Write(content)
	return written, errors.Join(writeErr, file.Close())
}

func (logger *Logger) Printf(format string, args ...any) {
	line := fmt.Sprintf(format, args...)
	if logger.token != "" {
		line = strings.ReplaceAll(line, string(logger.token), "***")
	}
	fmt.Fprintf(logger.out, "%s %s\n", time.Now().Format("2006-01-02 15:04:05"), line)
}
