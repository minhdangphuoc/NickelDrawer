package wallpaper

import (
	"encoding/json"
	"errors"
	"os"
	"time"
)

// Kinds of background work the progress file can describe.
const (
	ProgressImport = "import"
	ProgressRefit  = "refit"
)

// staleProgressAfter bounds how long a progress file of a dead job is believed.
const staleProgressAfter = 2 * time.Minute

// Progress is the progress file: {"done": 3, "total": 7, "name": "bien-xanh.jpg"}.
type Progress struct {
	Done  int    `json:"done"`
	Total int    `json:"total"`
	Name  string `json:"name"`
	Kind  string `json:"kind,omitempty"` // ProgressRefit for a mode re-fit; empty means import
}

// writeProgress replaces the file atomically; "" disables progress.
func writeProgress(path string, progress Progress) error {
	if path == "" {
		return nil
	}
	encoded, err := json.Marshal(progress)
	if err != nil {
		return err
	}
	return writeFileAtomic(path, ".wallpaper-progress-", encoded)
}

func clearProgress(path string, log *Logger) {
	if path == "" {
		return
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Printf("cannot remove progress file %s: %v", path, err)
	}
}

// ReadProgress returns the running job's progress; isFresh says it was written recently, which
// separates a live job from the leftovers of a crashed one.
func ReadProgress(path string) (progress Progress, found, isFresh bool) {
	data, err := os.ReadFile(path)
	if err != nil || json.Unmarshal(data, &progress) != nil {
		return Progress{}, false, false
	}
	info, err := os.Stat(path)
	return progress, true, err == nil && time.Since(info.ModTime()) < staleProgressAfter
}
