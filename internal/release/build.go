package release

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// BuildManifest lists the updatable files under distRoot (the tree that mirrors the Kobo's
// onboard root) and hashes them. Files outside the allowed folders, such as apps and data/, are
// skipped. App zips are added separately (AddAppPackages).
func BuildManifest(distRoot, version, minAppsctl string, created time.Time) (*Manifest, error) {
	manifest := &Manifest{
		Schema: SchemaVersion, Version: version, MinAppsctl: minAppsctl,
		Created: created.UTC().Format(time.RFC3339), AppPackages: []AppPackage{},
	}
	err := filepath.WalkDir(distRoot, func(fullPath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() {
			return walkErr
		}
		relative, err := filepath.Rel(distRoot, fullPath)
		if err != nil {
			return err
		}
		return manifest.addIfUpdatable(fullPath, filepath.ToSlash(relative), entry)
	})
	if err != nil {
		return nil, fmt.Errorf("cannot list %s: %w", distRoot, err)
	}
	sort.Slice(manifest.Files, func(i, j int) bool { return manifest.Files[i].Path < manifest.Files[j].Path })
	if err := manifest.Validate(); err != nil {
		return nil, fmt.Errorf("built manifest is invalid: %w", err)
	}
	return manifest, nil
}

func (manifest *Manifest) addIfUpdatable(fullPath, relative string, entry fs.DirEntry) error {
	if CheckPath(relative) != nil || isJunk(entry.Name()) {
		return nil
	}
	if !entry.Type().IsRegular() {
		return fmt.Errorf("%s is not a regular file; the release only ships regular files", relative)
	}
	file, err := describeFile(fullPath, relative)
	if err != nil {
		return err
	}
	manifest.Files = append(manifest.Files, file)
	return nil
}

func isJunk(name string) bool {
	return name == ".DS_Store" || strings.HasPrefix(name, "._")
}

func describeFile(fullPath, relative string) (File, error) {
	info, err := os.Stat(fullPath)
	if err != nil {
		return File{}, err
	}
	hash, err := HashFile(fullPath)
	if err != nil {
		return File{}, err
	}
	return File{Path: relative, SHA256: hash, Size: info.Size(), Mode: fmt.Sprintf("%04o", info.Mode().Perm())}, nil
}

// HashFile returns the lowercase hex SHA-256 of a file.
func HashFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("cannot read %s: %w", filePath, err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// Marshal renders release.json with a stable layout; the signature covers exactly these bytes.
func (manifest *Manifest) Marshal() ([]byte, error) {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

// WriteZip stores every manifest file, under its manifest path, in a zip.
func WriteZip(distRoot string, manifest *Manifest, output io.Writer) error {
	archive := zip.NewWriter(output)
	created, err := time.Parse(time.RFC3339, manifest.Created)
	if err != nil {
		return err
	}
	for _, file := range manifest.Files {
		if err := addToZip(archive, distRoot, file, created); err != nil {
			return err
		}
	}
	return archive.Close()
}

func addToZip(archive *zip.Writer, distRoot string, file File, modified time.Time) error {
	mode, err := file.ParseMode()
	if err != nil {
		return err
	}
	header := &zip.FileHeader{Name: path.Clean(file.Path), Method: zip.Deflate, Modified: modified}
	header.SetMode(fs.FileMode(mode))
	entry, err := archive.CreateHeader(header)
	if err != nil {
		return err
	}
	source, err := os.Open(filepath.Join(distRoot, filepath.FromSlash(file.Path)))
	if err != nil {
		return err
	}
	defer source.Close()
	if _, err := io.Copy(entry, source); err != nil {
		return fmt.Errorf("cannot add %s to the zip: %w", file.Path, err)
	}
	return nil
}
