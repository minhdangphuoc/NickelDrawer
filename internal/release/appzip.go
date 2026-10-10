package release

import (
	"archive/zip"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Limits of an app zip (docs/spec/app-updates.md, "The zip").
const (
	maxAppZipEntries     = 4096
	maxAppUnpackedBytes  = 64 << 20
	previewStatePrefix   = "preview-state"
	appZipRootDirPattern = appsPrefix + "/"
)

// ErrUnsafeZip marks a zip that breaks a safety rule; the message says which entry and rule.
var ErrUnsafeZip = errors.New("unsafe app zip")

// AppZipEntry is one file or folder of a checked app zip.
type AppZipEntry struct {
	Name         string // path below the app folder, forward slashes; "" is the app folder itself
	IsDir        bool
	IsExecutable bool
	Size         int64 // declared uncompressed size; extraction still enforces it
	File         *zip.File
}

func unsafeZip(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrUnsafeZip, fmt.Sprintf(format, args...))
}

// CheckAppZip applies the zip rules to an opened archive and returns its entries below the app
// folder: regular files and folders only, every path inside .adds/apps/<id>/, nothing under
// data/, no dev-only preview-state files, NFC names without case-fold duplicates, limits on
// the entry count and the unpacked size, and an app.json. Nothing is read from the archive.
func CheckAppZip(archive *zip.Reader, id string) ([]AppZipEntry, error) {
	if len(archive.File) > maxAppZipEntries {
		return nil, unsafeZip("%d entries; at most %d", len(archive.File), maxAppZipEntries)
	}
	var entries []AppZipEntry
	seen := map[string]bool{}
	var total uint64
	hasManifest := false
	for _, file := range archive.File {
		entry, err := checkAppZipEntry(file, id)
		if err != nil {
			return nil, err
		}
		if entry.Name == "" {
			continue // a folder above the app folder
		}
		key := strings.ToLower(entry.Name)
		if seen[key] {
			return nil, unsafeZip("%s appears twice (names are compared ignoring case)", file.Name)
		}
		seen[key] = true
		total += file.UncompressedSize64
		if total > maxAppUnpackedBytes {
			return nil, unsafeZip("more than %d bytes unpacked", maxAppUnpackedBytes)
		}
		hasManifest = hasManifest || entry.Name == "app.json"
		entries = append(entries, entry)
	}
	if !hasManifest {
		return nil, unsafeZip("no %s/%s/app.json", appsPrefix, id)
	}
	return entries, nil
}

func checkAppZipEntry(file *zip.File, id string) (AppZipEntry, error) {
	mode := file.Mode()
	isDir := file.FileInfo().IsDir()
	if !isDir && !mode.IsRegular() {
		return AppZipEntry{}, unsafeZip("%s is not a regular file or folder (links and devices are refused)", file.Name)
	}
	name := strings.TrimSuffix(file.Name, "/")
	if !norm.NFC.IsNormalString(name) {
		return AppZipEntry{}, unsafeZip("%s is not NFC-normalized Unicode", file.Name)
	}
	if hasControlRune(name) {
		return AppZipEntry{}, unsafeZip("%q contains a control character", file.Name)
	}
	relative, err := appRelativeName(name, id)
	if err != nil {
		return AppZipEntry{}, unsafeZip("%s: %v", file.Name, err)
	}
	if strings.HasPrefix(path.Base(relative), previewStatePrefix) {
		return AppZipEntry{}, unsafeZip("%s is a preview-state file; they are dev-only", file.Name)
	}
	return AppZipEntry{Name: relative, IsDir: isDir, IsExecutable: mode.Perm()&0o111 != 0,
		Size: int64(file.UncompressedSize64), File: file}, nil
}

// appRelativeName returns name below .adds/apps/<id>/. Folders above the app folder
// (.adds, .adds/apps) are accepted as bare directory entries and map to "".
func appRelativeName(name, id string) (string, error) {
	if name == appsPrefix || name == strings.Split(appsPrefix, "/")[0] || name == appsPrefix+"/"+id {
		return "", nil
	}
	if err := CheckAppEntry(id, name); err != nil {
		return "", err
	}
	return strings.TrimPrefix(name, appZipRootDirPattern+id+"/"), nil
}

func hasControlRune(text string) bool {
	return strings.IndexFunc(text, unicode.IsControl) >= 0
}

// ModeOf returns the permission bits an entry is installed with: 0755 or 0644, never setuid.
func (entry AppZipEntry) ModeOf() fs.FileMode {
	switch {
	case entry.IsDir, entry.IsExecutable:
		return 0o755
	}
	return 0o644
}

// IdentifyAppZip reads the app id and version a zip declares, refusing entries outside its app.
func IdentifyAppZip(zipPath string) (id, version string, err error) {
	return identifyAppZip(zipPath)
}
