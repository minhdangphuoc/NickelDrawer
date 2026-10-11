package tweaks

import (
	"fmt"
	"os"
	"path/filepath"
)

// writeFileAtomically writes through a temp file in the same folder and renames it, so a reader
// never sees a half-written file. A missing folder is created.
func writeFileAtomically(target string, content []byte, mode os.FileMode) error {
	dir := filepath.Dir(target)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("cannot create folder %s: %w", dir, err)
	}
	temp, err := os.CreateTemp(dir, ".tmp-"+filepath.Base(target)+"-")
	if err != nil {
		return fmt.Errorf("cannot create temp file in %s: %w", dir, err)
	}
	defer os.Remove(temp.Name()) // no-op after a successful rename
	if _, err := temp.Write(content); err != nil {
		temp.Close()
		return fmt.Errorf("cannot write %s: %w", temp.Name(), err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("cannot close %s: %w", temp.Name(), err)
	}
	// FAT-style mounts may refuse chmod; the content is what matters for these files.
	_ = os.Chmod(temp.Name(), mode)
	if err := os.Rename(temp.Name(), target); err != nil {
		return fmt.Errorf("cannot replace %s: %w", target, err)
	}
	return nil
}
