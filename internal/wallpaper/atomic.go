package wallpaper

import (
	"errors"
	"os"
	"path/filepath"

	"koboapps/internal/i18n"
)

// writeFileAtomic writes a temp file next to target (name starting with tempPrefix) and renames it.
// Nothing stays open on /mnt/onboard and readers never see a half-written file.
func writeFileAtomic(target, tempPrefix string, content []byte) error {
	temp, err := os.CreateTemp(filepath.Dir(target), tempPrefix+"*.tmp")
	if err != nil {
		return i18n.NewError("msg.err_tempfile", "path", target).WithCause(err)
	}
	defer os.Remove(temp.Name())
	_, writeErr := temp.Write(content)
	syncErr := temp.Sync()
	closeErr := temp.Close()
	if err := errors.Join(writeErr, syncErr, closeErr); err != nil {
		return i18n.NewError("msg.err_write", "path", target).WithCause(err)
	}
	// CreateTemp makes 0600; vfat ignores modes, other file systems should be readable.
	_ = os.Chmod(temp.Name(), 0o644)
	if err := os.Rename(temp.Name(), target); err != nil {
		return i18n.NewError("msg.err_write", "path", target).WithCause(err)
	}
	return nil
}
