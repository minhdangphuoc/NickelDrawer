package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// writeFileAtomically writes via a temp file in the same folder and renames it,
// so NickelMenu never reads a half-written config.
func writeFileAtomically(target string, content []byte) error {
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
	if err := os.Chmod(temp.Name(), 0o644); err != nil {
		return fmt.Errorf("cannot set mode on %s: %w", temp.Name(), err)
	}
	if err := os.Rename(temp.Name(), target); err != nil {
		return fmt.Errorf("cannot replace %s: %w", target, err)
	}
	return nil
}
