package release

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// VerifyZip checks that the zip holds exactly the manifest's files and that each one matches
// its sha256 and size. Nothing is extracted.
func VerifyZip(manifest *Manifest, zipPath string) error {
	archive, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("cannot open release zip %s: %w", zipPath, err)
	}
	defer archive.Close()
	entries, err := indexEntries(archive)
	if err != nil {
		return err
	}
	for _, file := range manifest.Files {
		entry, found := entries[file.Path]
		if !found {
			return fmt.Errorf("%s is listed in release.json but missing from the zip", file.Path)
		}
		if err := verifyEntry(file, entry, io.Discard); err != nil {
			return err
		}
		delete(entries, file.Path)
	}
	for name := range entries {
		return fmt.Errorf("%s is in the zip but not listed in release.json", name)
	}
	return nil
}

func indexEntries(archive *zip.ReadCloser) (map[string]*zip.File, error) {
	entries := map[string]*zip.File{}
	for _, entry := range archive.File {
		if entry.FileInfo().IsDir() {
			continue
		}
		if _, duplicate := entries[entry.Name]; duplicate {
			return nil, fmt.Errorf("%s appears twice in the zip", entry.Name)
		}
		entries[entry.Name] = entry
	}
	return entries, nil
}

// ExtractFile writes one manifest file from the zip to destination after checking its
// sha256 and size; a mismatch leaves no file behind.
func ExtractFile(manifest *Manifest, zipPath, relativePath, destination string) error {
	file, found := manifest.Find(relativePath)
	if !found {
		return fmt.Errorf("%s is not listed in release.json", relativePath)
	}
	archive, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("cannot open release zip %s: %w", zipPath, err)
	}
	defer archive.Close()
	for _, entry := range archive.File {
		if entry.Name == relativePath {
			return extractEntry(file, entry, destination)
		}
	}
	return fmt.Errorf("%s is missing from the zip", relativePath)
}

func extractEntry(file File, entry *zip.File, destination string) (err error) {
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return fmt.Errorf("cannot create folder for %s: %w", destination, err)
	}
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("cannot create %s: %w", destination, err)
	}
	defer func() {
		if closeErr := output.Close(); err == nil {
			err = closeErr
		}
		if err != nil {
			os.Remove(destination)
		}
	}()
	return verifyEntry(file, entry, output)
}

// verifyEntry streams an entry into sink while hashing it; reading stops one byte past the
// declared size so a lying archive cannot fill the disk.
func verifyEntry(file File, entry *zip.File, sink io.Writer) error {
	reader, err := entry.Open()
	if err != nil {
		return fmt.Errorf("cannot read %s from the zip: %w", file.Path, err)
	}
	defer reader.Close()
	hash := sha256.New()
	written, err := io.Copy(io.MultiWriter(hash, sink), io.LimitReader(reader, file.Size+1))
	if err != nil {
		return fmt.Errorf("cannot read %s from the zip: %w", file.Path, err)
	}
	if written != file.Size {
		return fmt.Errorf("%s has the wrong size (release.json says %d bytes)", file.Path, file.Size)
	}
	if hex.EncodeToString(hash.Sum(nil)) != file.SHA256 {
		return fmt.Errorf("%s does not match the sha256 in release.json", file.Path)
	}
	return nil
}

// Find returns the manifest entry for a path.
func (manifest *Manifest) Find(relativePath string) (File, bool) {
	for _, file := range manifest.Files {
		if file.Path == relativePath {
			return file, true
		}
	}
	return File{}, false
}
