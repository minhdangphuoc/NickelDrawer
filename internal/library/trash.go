package library

import (
	"os"
	"path/filepath"
	"time"

	"koboapps/internal/i18n"
)

// RemoveDuplicate moves one file of a duplicate group to data/trash/<date>/<relative path>.
// It refuses unless confirmed, unless the file is in a group of the last report, and unless
// another member of the group still exists, so the last copy is never trashed.
// Trash is restorable (RestoreTrash) and purged after 30 days. The caller holds the lock.
func (service *Service) RemoveDuplicate(relative string, confirmed bool) error {
	if !confirmed {
		return i18n.NewError("msg.err_need_confirm")
	}
	report, err := service.requireReport()
	if err != nil {
		return err
	}
	groupIndex, err := service.removableGroup(report, relative)
	if err != nil {
		return err
	}
	absolute, err := resolveInside(relative)
	if err != nil {
		return err
	}
	if _, err := regularFile(absolute); err != nil {
		return i18n.NewError("msg.err_read", "path", relative).WithCause(err)
	}
	target := filepath.Join(service.Paths.Trash(), today(), filepath.FromSlash(relative))
	if _, err := os.Lstat(target); err == nil {
		return i18n.NewError("msg.err_exists", "path", target)
	}
	if err := moveFile(absolute, target); err != nil {
		return err
	}
	service.Logger.Printf("trash %s", relative)
	service.dropFromReport(&report, groupIndex, relative)
	return SaveReport(service.Paths, report)
}

// removableGroup finds the group holding relative and checks another member is still on disk.
func (service *Service) removableGroup(report Report, relative string) (int, error) {
	if err := validateRelative(relative); err != nil {
		return 0, err
	}
	for index, group := range report.Duplicates {
		if !contains(group.Paths, relative) {
			continue
		}
		for _, other := range group.Paths {
			if other == relative {
				continue
			}
			if _, err := regularFile(absoluteOf(other)); err == nil {
				return index, nil
			}
		}
		return 0, i18n.NewError("msg.err_last_copy", "path", relative)
	}
	return 0, i18n.NewError("msg.err_not_duplicate", "path", relative)
}

func (service *Service) dropFromReport(report *Report, groupIndex int, relative string) {
	group := &report.Duplicates[groupIndex]
	group.Paths = without(group.Paths, relative)
	if len(group.Paths) < 2 {
		report.Duplicates = append(report.Duplicates[:groupIndex], report.Duplicates[groupIndex+1:]...)
	}
	report.Books = withoutBook(report.Books, relative)
	report.refreshCounts()
	if selected, _ := os.ReadFile(service.Paths.Selection()); string(selected) == relative {
		_ = os.Remove(service.Paths.Selection())
	}
}

// PurgeTrash deletes trash day-folders older than 30 days, only below data/trash.
func (service *Service) PurgeTrash() {
	entries, err := os.ReadDir(service.Paths.Trash())
	if err != nil {
		return
	}
	cutoff := service.Now().Add(-trashRetention)
	for _, entry := range entries {
		day, parseErr := time.ParseInLocation(dateLayout, entry.Name(), time.Local)
		if !entry.IsDir() || parseErr != nil || !day.Before(cutoff) {
			continue
		}
		if err := os.RemoveAll(filepath.Join(service.Paths.Trash(), entry.Name())); err != nil {
			service.Logger.Printf("cannot purge trash %s: %v", entry.Name(), err)
			continue
		}
		service.Logger.Printf("purged trash of %s", entry.Name())
	}
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func without(values []string, removed string) []string {
	kept := make([]string, 0, len(values))
	for _, value := range values {
		if value != removed {
			kept = append(kept, value)
		}
	}
	return kept
}

func withoutBook(books []Book, relative string) []Book {
	kept := make([]Book, 0, len(books))
	for _, book := range books {
		if book.Path != relative {
			kept = append(kept, book)
		}
	}
	return kept
}

// Select remembers the duplicate the user tapped, so a later confirmed button can trash it.
func (service *Service) Select(relative string) error {
	report, err := service.requireReport()
	if err != nil {
		return err
	}
	if _, err := service.removableGroup(report, relative); err != nil {
		return err
	}
	return writeFileAtomic(service.Paths.Selection(), []byte(relative))
}

// Selected returns the remembered duplicate, "" when none.
func (service *Service) Selected() string {
	content, err := os.ReadFile(service.Paths.Selection())
	if err != nil {
		return ""
	}
	return string(content)
}
