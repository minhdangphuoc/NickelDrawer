// Package library scans the book folders, finds Vietnamese EPUBs tagged with the wrong language
// and duplicate files, and fixes or moves them with a backup (see docs/spec/library.md).
package library

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"koboapps/internal/i18n"
	"koboapps/internal/storage"
)

const (
	// maxScanBytes is the largest file the scan reads (same cap as the inbox router for books).
	maxScanBytes = 200 << 20
	// maxFiles stops a runaway tree from keeping the Kobo busy.
	maxFiles = 20000
	// maxDepth is how deep below a library folder the scan goes (Books/GitHub/repo/...).
	maxDepth = 8
	// spaceMargin is kept free on top of the two copies (backup and rewritten file) a fix needs.
	spaceMargin = 16 << 20
	// trashRetention is how long a trashed duplicate stays restorable.
	trashRetention = 30 * 24 * time.Hour

	dateLayout = "2006-01-02"
)

// maxFixBytes is the largest EPUB the fix rewrites; bigger ones are skipped. A variable so tests can lower it.
var maxFixBytes int64 = 100 << 20

var datePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// scannedFolders is the allowlist of library folders; nothing else is ever read or written.
var scannedFolders = []storage.Folder{storage.Books, storage.Documents, storage.Comics}

// Paths locates the app's own state, all inside its data/ folder.
type Paths struct{ AppDir string }

func (paths Paths) Data() string      { return filepath.Join(paths.AppDir, "data") }
func (paths Paths) Report() string    { return filepath.Join(paths.Data(), "report.json") }
func (paths Paths) Summary() string   { return filepath.Join(paths.Data(), "summary.json") }
func (paths Paths) Progress() string  { return filepath.Join(paths.Data(), "progress.json") }
func (paths Paths) Log() string       { return filepath.Join(paths.Data(), "log.txt") }
func (paths Paths) Lock() string      { return filepath.Join(paths.Data(), "lock") }
func (paths Paths) Selection() string { return filepath.Join(paths.Data(), "selected.txt") }
func (paths Paths) Page() string      { return filepath.Join(paths.Data(), "page.txt") }
func (paths Paths) Backups() string   { return filepath.Join(paths.Data(), "backups") }
func (paths Paths) Trash() string     { return filepath.Join(paths.Data(), "trash") }

// relativePath is the report form of a library file: slash-separated, below the storage root.
func relativePath(absolute string) (string, error) {
	relative, err := filepath.Rel(storage.Root(), absolute)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(relative), nil
}

// folderOf returns the allowlisted folder a relative path starts with.
func folderOf(relative string) (storage.Folder, bool) {
	first, _, _ := strings.Cut(relative, "/")
	for _, folder := range scannedFolders {
		if filepath.Base(storage.Path(folder)) == first {
			return folder, true
		}
	}
	return "", false
}

// validateRelative accepts only clean slash paths inside one scanned folder, never hidden parts.
func validateRelative(relative string) error {
	cleaned := path.Clean(relative)
	if cleaned != relative || path.IsAbs(relative) || strings.HasPrefix(relative, "../") || relative == "." {
		return i18n.NewError("msg.err_bad_path", "path", relative)
	}
	if _, inside := folderOf(relative); !inside || !strings.Contains(relative, "/") {
		return i18n.NewError("msg.err_bad_path", "path", relative)
	}
	for _, part := range strings.Split(relative, "/") {
		if storage.IsSkippedName(part) {
			return i18n.NewError("msg.err_bad_path", "path", relative)
		}
	}
	return nil
}

// resolveInside turns a validated relative path into an absolute one and checks that its
// parent really lives inside the folder (no symlinked sub-folder leads out of it).
func resolveInside(relative string) (string, error) {
	if err := validateRelative(relative); err != nil {
		return "", err
	}
	folder, _ := folderOf(relative)
	realFolder, err := filepath.EvalSymlinks(storage.Path(folder))
	if err != nil {
		return "", i18n.NewError("msg.err_read", "path", storage.Path(folder)).WithCause(err)
	}
	absolute := filepath.Join(storage.Root(), filepath.FromSlash(relative))
	realParent, err := filepath.EvalSymlinks(filepath.Dir(absolute))
	if err != nil {
		return "", i18n.NewError("msg.err_read", "path", absolute).WithCause(err)
	}
	if realParent != realFolder && !strings.HasPrefix(realParent, realFolder+string(filepath.Separator)) {
		return "", i18n.NewError("msg.err_outside", "path", relative)
	}
	return filepath.Join(realParent, filepath.Base(absolute)), nil
}

// regularFile returns the size of a regular, non-symlink file.
func regularFile(absolute string) (int64, error) {
	info, err := os.Lstat(absolute)
	if err != nil {
		return 0, err
	}
	if !info.Mode().IsRegular() {
		return 0, errors.New("not a regular file")
	}
	return info.Size(), nil
}

func validateDate(date string) error {
	if !datePattern.MatchString(date) {
		return i18n.NewError("msg.err_bad_date", "date", date)
	}
	return nil
}

func today() string { return time.Now().Format(dateLayout) }

func megabytes(size int64) string { return fmt.Sprintf("%.1f MB", float64(size)/(1<<20)) }
