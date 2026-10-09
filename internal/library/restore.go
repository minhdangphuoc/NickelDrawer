package library

import (
	"errors"
	"io/fs"
	"koboapps/internal/storage"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"koboapps/internal/i18n"
)

// LatestBackupDate returns the newest data/backups/<date> folder name, "" when there is none.
func (service *Service) LatestBackupDate() string {
	return latestDate(service.Paths.Backups())
}

// HasBackups is true when at least one fix run can be undone.
func (service *Service) HasBackups() bool { return service.LatestBackupDate() != "" }

func latestDate(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	latest := ""
	for _, entry := range entries {
		if entry.IsDir() && datePattern.MatchString(entry.Name()) && entry.Name() > latest {
			latest = entry.Name()
		}
	}
	return latest
}

// Undo puts the originals of one fix run back (overwriting the fixed files) and deletes the
// restored backups. An empty date means the newest run. Returns how many files came back.
func (service *Service) Undo(date string) (restored int, err error) {
	if date == "" {
		date = service.LatestBackupDate()
	}
	if date == "" {
		return 0, i18n.NewError("msg.err_no_backups")
	}
	if err := validateDate(date); err != nil {
		return 0, err
	}
	restored, err = service.restoreTree(filepath.Join(service.Paths.Backups(), date), true)
	service.Logger.Printf("undo %s: restored %d", date, restored)
	return restored, err
}

// RestoreTrash puts the duplicates trashed on one day back; a file is never overwritten.
func (service *Service) RestoreTrash(date string) (restored int, err error) {
	if date == "" {
		date = latestDate(service.Paths.Trash())
	}
	if date == "" {
		return 0, i18n.NewError("msg.err_no_trash")
	}
	if err := validateDate(date); err != nil {
		return 0, err
	}
	restored, err = service.restoreTree(filepath.Join(service.Paths.Trash(), date), false)
	service.Logger.Printf("restore-trash %s: restored %d", date, restored)
	return restored, err
}

// restoreTree moves every file below source (a dated folder) back to the same relative path
// below the storage root. Targets are checked to stay inside a library folder.
func (service *Service) restoreTree(source string, overwrite bool) (int, error) {
	if info, err := os.Lstat(source); err != nil || !info.IsDir() {
		return 0, i18n.NewError("msg.err_no_backups")
	}
	var relatives []string
	walkErr := filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err == nil && entry.Type().IsRegular() {
			relative, relErr := filepath.Rel(source, path)
			if relErr == nil {
				relatives = append(relatives, filepath.ToSlash(relative))
			}
		}
		return err
	})
	if walkErr != nil {
		return 0, i18n.NewError("msg.err_read", "path", source).WithCause(walkErr)
	}
	sort.Strings(relatives)
	restored := 0
	var firstFailure error
	for _, relative := range relatives {
		if err := service.restoreOne(source, relative, overwrite); err != nil {
			service.Logger.Printf("restore %s: %v", relative, err)
			if firstFailure == nil {
				firstFailure = err
			}
			continue
		}
		restored++
	}
	removeEmptyDirs(source)
	return restored, firstFailure
}

func (service *Service) restoreOne(source, relative string, overwrite bool) error {
	if err := validateRelative(relative); err != nil {
		return err
	}
	if err := makeFolders(path.Dir(relative)); err != nil {
		return err
	}
	target, err := resolveInside(relative)
	if err != nil {
		return err
	}
	if _, statErr := os.Lstat(target); statErr == nil && !overwrite {
		return i18n.NewError("msg.err_exists", "path", relative)
	}
	return moveFile(filepath.Join(source, filepath.FromSlash(relative)), target)
}

// removeEmptyDirs deletes root and any empty sub-folders; non-empty ones stay.
func removeEmptyDirs(root string) {
	var dirs []string
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err == nil && entry.IsDir() {
			dirs = append(dirs, path)
		}
		return nil
	})
	for index := len(dirs) - 1; index >= 0; index-- {
		_ = os.Remove(dirs[index])
	}
}

// makeFolders creates the folders of a relative path one level at a time and refuses to pass
// through a symlink, which could lead outside the library folder.
func makeFolders(relativeDir string) error {
	current := storage.Root()
	for _, part := range strings.Split(relativeDir, "/") {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		switch {
		case errors.Is(err, os.ErrNotExist):
			if err := os.Mkdir(current, 0o755); err != nil {
				return i18n.NewError("msg.err_mkdir", "path", current).WithCause(err)
			}
		case err != nil:
			return i18n.NewError("msg.err_read", "path", current).WithCause(err)
		case !info.IsDir():
			return i18n.NewError("msg.err_outside", "path", current)
		}
	}
	return nil
}
