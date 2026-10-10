package wallpaper

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"koboapps/internal/i18n"
	"koboapps/internal/storage"
)

// ParsePictureID accepts "0003" and "3".
func ParsePictureID(text string) (int, error) {
	id, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil || id <= 0 {
		return 0, i18n.NewError("msg.err_picture_id_invalid", "value", text)
	}
	return id, nil
}

func (library *Library) require(id int) (Entry, error) {
	entry, found := library.find(id)
	if !found {
		return Entry{}, i18n.NewError("msg.err_picture_unknown", "id", pictureID(id))
	}
	return entry, nil
}

// SetCurrent makes one picture the wallpaper and applies the sleep screen.
func (service *Service) SetCurrent(id int) (Entry, error) {
	library, _, err := service.load()
	if err != nil {
		return Entry{}, err
	}
	entry, err := library.require(id)
	if err != nil {
		return Entry{}, err
	}
	if err := service.updateConfig(func(config *Config) { config.Current = id }); err != nil {
		return Entry{}, err
	}
	_, err = service.applyCurrentSettings(library)
	return entry, err
}

// RemoveReport says what Remove did beyond forgetting the picture.
type RemoveReport struct {
	Entry            Entry
	OriginalDeleted  bool
	OriginalNotFound bool // deleteOriginal was asked but no safe matching file exists
}

// Remove forgets one picture: our converted copy, thumbnail, index entry and our file in the
// screensaver folder. Its hash is recorded so an import does not bring it back. The original in
// Pictures is deleted only when deleteOriginal is set and it is provably the same file.
func (service *Service) Remove(id int, deleteOriginal bool) (RemoveReport, error) {
	library, config, err := service.load()
	if err != nil {
		return RemoveReport{}, err
	}
	entry, err := library.require(id)
	if err != nil {
		return RemoveReport{}, err
	}
	// Tombstone first: if a later step fails, the picture still does not come back.
	if err := library.markRemoved(entry.Hash); err != nil {
		return RemoveReport{}, err
	}
	wasShown := OwnedCount(service.ScreensaverDir) > 0
	library.drop(id)
	if err := library.save(); err != nil {
		return RemoveReport{}, err
	}
	service.removeFiles(library, id)
	if config.Current == id {
		next := library.successorOf(id)
		if err := service.updateConfig(func(config *Config) { config.Current = next }); err != nil {
			return RemoveReport{}, err
		}
	}
	if err := service.refreshSleepScreen(library, wasShown); err != nil {
		return RemoveReport{}, err
	}
	report := RemoveReport{Entry: entry}
	if deleteOriginal {
		report.OriginalDeleted = service.deleteOriginal(entry)
		report.OriginalNotFound = !report.OriginalDeleted
	}
	return report, nil
}

func (library *Library) drop(id int) {
	kept := library.Entries[:0:0]
	for _, entry := range library.Entries {
		if entry.ID != id {
			kept = append(kept, entry)
		}
	}
	library.Entries = kept
}

// successorOf is the next picture after id, else the previous one, else 0.
func (library *Library) successorOf(id int) int {
	previous := 0
	for _, entry := range library.Entries {
		if entry.ID > id {
			return entry.ID
		}
		previous = entry.ID
	}
	return previous
}

// removeFiles deletes our converted copy and thumbnail; failures are logged, the index no longer lists them.
func (service *Service) removeFiles(library *Library, id int) {
	for _, path := range []string{library.Paths.Image(id), library.Paths.Thumb(id)} {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			service.Log.Printf("cannot remove %s: %v", path, err)
		}
	}
}

// refreshSleepScreen drops our file for the removed picture and keeps the sleep screen valid.
// A wallpaper the user turned off stays off.
func (service *Service) refreshSleepScreen(library *Library, wasShown bool) error {
	if len(library.Entries) == 0 {
		_, err := Off(service.ScreensaverDir, service.Log)
		return err
	}
	if !wasShown {
		return nil
	}
	_, err := service.applyCurrentSettings(library)
	return err
}

// deleteOriginal removes the entry's original from a source folder, and only a regular file that
// is a direct child (storage.ListImages resolves links) whose name is not a link and whose SHA-256
// equals the entry's hash. Anything else is left alone.
func (service *Service) deleteOriginal(entry Entry) bool {
	for _, source := range service.Sources {
		realSource, err := filepath.EvalSymlinks(source)
		if err != nil {
			continue
		}
		for _, file := range storage.ListImages(source, maxSourceBytes, service.Log.Printf) {
			if file.Name != entry.Name || file.Path != filepath.Join(realSource, file.Name) {
				continue
			}
			content, err := storage.ReadLimited(file.Path, maxSourceBytes)
			if err != nil || hashOf(content) != entry.Hash {
				continue
			}
			if err := os.Remove(file.Path); err != nil {
				service.Log.Printf("cannot delete original %s: %v", file.Path, err)
				return false
			}
			service.Log.Printf("deleted original %s", file.Name)
			return true
		}
	}
	return false
}
