package wallpaper

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"koboapps/internal/i18n"
)

// ownedTempPrefix is the temp name used while copying; hidden, so Nickel never lists it.
const ownedTempPrefix = "." + OwnedPrefix

func ownedFileName(id int) string { return OwnedPrefix + imageFileName(id) }

// isOwned says whether name is a file we created. Nothing else in the folder is ever touched.
func isOwned(name string) bool {
	return strings.HasPrefix(name, OwnedPrefix) || strings.HasPrefix(name, ownedTempPrefix)
}

// ApplyReport says what apply changed.
type ApplyReport struct {
	Shown   int // pictures now in the screensaver folder
	Written int
	Removed int
}

// Apply makes the screensaver folder hold exactly the wanted pictures among our own files:
// fixed rotation = only the current picture, random = all of them.
func Apply(library *Library, config Config, screensaverDir string, log *Logger) (ApplyReport, error) {
	wanted := wantedEntries(library, config)
	if len(wanted) == 0 {
		return ApplyReport{}, i18n.NewError("msg.err_no_pictures")
	}
	if err := os.MkdirAll(screensaverDir, 0o755); err != nil {
		return ApplyReport{}, i18n.NewError("msg.err_mkdir", "path", screensaverDir).WithCause(err)
	}
	keep := map[string]bool{}
	var report ApplyReport
	for _, entry := range wanted {
		keep[ownedFileName(entry.ID)] = true
		wrote, err := publish(library.Paths.Image(entry.ID), filepath.Join(screensaverDir, ownedFileName(entry.ID)))
		if err != nil {
			return report, err
		}
		if wrote {
			report.Written++
		}
	}
	removed, err := removeOwned(screensaverDir, keep, log)
	report.Removed = removed
	report.Shown = len(wanted)
	return report, err
}

func wantedEntries(library *Library, config Config) []Entry {
	if config.Rotation == RotationRandom {
		return library.Entries
	}
	if entry, found := library.find(config.Current); found {
		return []Entry{entry}
	}
	if len(library.Entries) > 0 {
		return library.Entries[:1]
	}
	return nil
}

// publish copies source to target unless target already holds the same bytes (saves flash writes).
func publish(source, target string) (bool, error) {
	content, err := os.ReadFile(source)
	if err != nil {
		return false, i18n.NewError("msg.err_read", "path", source).WithCause(err)
	}
	if existing, err := os.ReadFile(target); err == nil && bytes.Equal(existing, content) {
		return false, nil
	}
	return true, writeFileAtomic(target, ownedTempPrefix, content)
}

// Off removes every file we put into the screensaver folder and leaves all others alone.
func Off(screensaverDir string, log *Logger) (int, error) {
	return removeOwned(screensaverDir, nil, log)
}

// removeOwned deletes our regular files that are not in keep. Symlinks and foreign names are skipped.
func removeOwned(dir string, keep map[string]bool, log *Logger) (int, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, i18n.NewError("msg.err_list", "path", dir).WithCause(err)
	}
	removed := 0
	for _, entry := range entries {
		if !isOwned(entry.Name()) || keep[entry.Name()] || !entry.Type().IsRegular() {
			continue
		}
		if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil {
			return removed, i18n.NewError("msg.err_remove", "path", entry.Name()).WithCause(err)
		}
		log.Printf("removed %s", entry.Name())
		removed++
	}
	return removed, nil
}

// OwnedCount is how many of our pictures sit in the screensaver folder now.
func OwnedCount(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	count := 0
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), OwnedPrefix) && entry.Type().IsRegular() {
			count++
		}
	}
	return count
}
