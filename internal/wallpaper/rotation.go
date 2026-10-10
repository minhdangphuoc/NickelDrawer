package wallpaper

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"math/big"
	"os"

	"koboapps/internal/i18n"
)

// historyLength is how many recently shown pictures a random pick avoids (the current one and the one
// before it), so the sleep screen never shows the same picture twice in a row and never bounces back.
const historyLength = 2

// pickRandom chooses uniformly among the entries that are neither current nor in recent. With too few
// pictures it relaxes the rule: first only current is avoided (2 pictures alternate), and a single
// picture stays. Returns 0 for an empty library.
func pickRandom(entries []Entry, current int, recent []int, randomIndex func(int) int) int {
	avoided := map[int]bool{current: true}
	for _, id := range recent {
		avoided[id] = true
	}
	candidates := entriesNotIn(entries, avoided)
	if len(candidates) == 0 {
		candidates = entriesNotIn(entries, map[int]bool{current: true})
	}
	if len(candidates) == 0 {
		candidates = entries
	}
	if len(candidates) == 0 {
		return 0
	}
	return candidates[randomIndex(len(candidates))].ID
}

func entriesNotIn(entries []Entry, avoided map[int]bool) []Entry {
	var kept []Entry
	for _, entry := range entries {
		if !avoided[entry.ID] {
			kept = append(kept, entry)
		}
	}
	return kept
}

// cryptoIndex is a uniform index in [0, count) from the kernel's random source. Nickel's own
// qrand() choice is what this replaces, so no process-local seed is involved.
func cryptoIndex(count int) int {
	if count <= 1 {
		return 0
	}
	number, err := rand.Int(rand.Reader, big.NewInt(int64(count)))
	if err != nil {
		return 0 // only on a broken /dev/urandom; the history rule still avoids a repeat
	}
	return int(number.Int64())
}

// loadHistory reads data/rotation-history.json (most recent first); missing or unreadable gives none,
// which only weakens the no-repeat rule for one pick.
func loadHistory(path string) []int {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var history []int
	if json.Unmarshal(data, &history) != nil {
		return nil
	}
	return history
}

// rememberShown puts id first and keeps historyLength entries.
func rememberShown(history []int, id int) []int {
	updated := []int{id}
	for _, previous := range history {
		if previous != id && len(updated) < historyLength {
			updated = append(updated, previous)
		}
	}
	return updated
}

func saveHistory(path string, history []int) error {
	encoded, err := json.Marshal(history)
	if err != nil {
		return i18n.NewError("msg.err_write", "path", path).WithCause(err)
	}
	return writeFileAtomic(path, ".history-", append(encoded, '\n'))
}

// syncWakeFlag makes data/rotate-on-wake exist exactly while rotation is random. The NickelDrawer mod
// only checks that the file exists when the device wakes, then runs `wallpaper next` in the background.
// It writes only on a change, so repeated calls cost one stat.
func syncWakeFlag(path string, rotation Rotation) error {
	_, statErr := os.Stat(path)
	exists := statErr == nil
	switch {
	case rotation == RotationRandom && !exists:
		return writeFileAtomic(path, ".rotate-on-wake-", nil)
	case rotation != RotationRandom && exists:
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return i18n.NewError("msg.err_remove", "path", path).WithCause(err)
		}
	}
	return nil
}
