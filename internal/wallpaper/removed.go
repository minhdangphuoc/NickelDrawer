package wallpaper

import (
	"encoding/json"
	"errors"
	"os"
	"sort"

	"koboapps/internal/i18n"
)

// loadRemoved reads data/removed.json (a JSON array of SHA-256 hashes); a missing file is empty.
func (library *Library) loadRemoved() error {
	library.Removed = map[string]bool{}
	data, err := os.ReadFile(library.Paths.Removed())
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	var hashes []string
	if err == nil {
		err = json.Unmarshal(data, &hashes)
	}
	if err != nil {
		return i18n.NewError("msg.err_removed_read", "path", library.Paths.Removed()).WithCause(err)
	}
	for _, hash := range hashes {
		library.Removed[hash] = true
	}
	return nil
}

// markRemoved records hash so a later import skips the picture.
func (library *Library) markRemoved(hash string) error {
	library.Removed[hash] = true
	hashes := make([]string, 0, len(library.Removed))
	for known := range library.Removed {
		hashes = append(hashes, known)
	}
	sort.Strings(hashes)
	encoded, err := json.MarshalIndent(hashes, "", "  ")
	if err != nil {
		return i18n.NewError("msg.err_write", "path", library.Paths.Removed()).WithCause(err)
	}
	return writeFileAtomic(library.Paths.Removed(), ".removed-", append(encoded, '\n'))
}
