package wallpaper

import (
	"errors"
	"os"

	"koboapps/internal/i18n"
	"koboapps/internal/storage"
)

// FirstImportDue tells whether the app should import Pictures by itself now, and records the
// marker when it should. Due once per install: empty library, no marker, and at least one
// supported picture. It only lists and sniffs the first match, never decodes, so it is cheap
// enough for the state action. The caller makes sure no job is running.
func (service *Service) FirstImportDue() (bool, error) {
	if hasFile(service.Paths.FirstImportMarker()) {
		return false, nil
	}
	library, err := LoadLibrary(service.Paths)
	if err != nil {
		return false, err
	}
	if len(library.Entries) > 0 || !service.hasSourcePicture() {
		return false, nil
	}
	return service.recordFirstImport()
}

func (service *Service) recordFirstImport() (bool, error) {
	marker := service.Paths.FirstImportMarker()
	if err := os.MkdirAll(service.Paths.Data(), 0o755); err != nil {
		return false, i18n.NewError("msg.err_mkdir", "path", service.Paths.Data()).WithCause(err)
	}
	if err := writeFileAtomic(marker, ".first-import-", []byte("1\n")); err != nil {
		return false, err
	}
	service.Log.Printf("first import: library empty, Pictures has pictures, importing")
	return true, nil
}

func (service *Service) hasSourcePicture() bool {
	for _, source := range service.Sources {
		if _, found := storage.FirstImage(source, maxSourceBytes, service.Log.Printf); found {
			return true
		}
	}
	return false
}

func hasFile(path string) bool {
	_, err := os.Stat(path)
	return !errors.Is(err, os.ErrNotExist)
}
