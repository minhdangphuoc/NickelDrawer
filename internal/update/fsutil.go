package update

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// WriteFileAtomically writes via a temp file in the same folder and renames it, so a reader
// never sees a half-written file.
func WriteFileAtomically(target string, content []byte, mode os.FileMode) error {
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
	return finishReplace(temp.Name(), target, mode)
}

// copyFileAtomically copies source over target through a temp file and a rename.
// Renaming over a running executable is fine on Linux, which is how appsctl updates itself.
func copyFileAtomically(source, target string, mode os.FileMode) error {
	input, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("cannot open %s: %w", source, err)
	}
	defer input.Close()
	dir := filepath.Dir(target)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("cannot create folder %s: %w", dir, err)
	}
	temp, err := os.CreateTemp(dir, ".tmp-"+filepath.Base(target)+"-")
	if err != nil {
		return fmt.Errorf("cannot create temp file in %s: %w", dir, err)
	}
	defer os.Remove(temp.Name())
	if _, err := io.Copy(temp, input); err != nil {
		temp.Close()
		return fmt.Errorf("cannot copy %s to %s: %w", source, target, err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("cannot close %s: %w", temp.Name(), err)
	}
	return finishReplace(temp.Name(), target, mode)
}

func finishReplace(tempPath, target string, mode os.FileMode) error {
	if err := setMode(tempPath, mode); err != nil {
		return err
	}
	if err := os.Rename(tempPath, target); err != nil {
		return fmt.Errorf("cannot replace %s: %w", target, err)
	}
	return nil
}

// setMode applies permission bits. FAT-style mounts may refuse chmod yet still decide
// executability from the mount options, so a refusal is an error only if the executable
// bit then differs from what was asked for.
func setMode(path string, mode os.FileMode) error {
	chmodErr := os.Chmod(path, mode)
	if chmodErr == nil {
		return nil
	}
	info, err := os.Stat(path)
	if err == nil && (info.Mode()&0o111 != 0) == (mode&0o111 != 0) {
		return nil
	}
	return fmt.Errorf("cannot set mode %o on %s: %w", mode, path, chmodErr)
}

// hashIfExists returns the sha256 of a file, or "" when it does not exist.
func hashIfExists(path string, hash func(string) (string, error)) (string, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", nil
	}
	return hash(path)
}
