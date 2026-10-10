package library

import (
	"os"

	"koboapps/internal/i18n"
)

// ErrCancelled ends a scan the user stopped. A stopped scan saves nothing, so the report of
// the last complete scan stays valid.
var ErrCancelled = i18n.NewError("msg.cancelled")

// Cancelled reports whether the user asked the running command to stop. Loops check it
// between books, so the book being written always finishes and no file is left half done.
func (service *Service) Cancelled() bool {
	_, err := os.Stat(service.Paths.Cancel())
	return err == nil
}

// RequestCancel asks the running command to stop at the next book. It returns false when
// nothing is running, so a stale request cannot cancel the next run (Acquire clears it too).
func RequestCancel(paths Paths) (bool, error) {
	if !IsRunning(paths) {
		return false, nil
	}
	if err := writeFileAtomic(paths.Cancel(), []byte("1")); err != nil {
		return false, err
	}
	return true, nil
}

// IsCancelling is true while a stop request waits for the running command to notice it.
func IsCancelling(paths Paths) bool {
	_, err := os.Stat(paths.Cancel())
	return err == nil && IsRunning(paths)
}
