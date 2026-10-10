package update

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"koboapps/internal/release"
)

// Availability is what the last `update check` found, kept in data/update/available.json so
// the drawer can show "1 update" without touching the network. It is a cache: the signed
// manifest is checked again by `update apply`.
type Availability struct {
	IsAvailable bool   `json:"available"`
	Version     string `json:"version,omitempty"`
	Current     string `json:"current"`
	Notes       string `json:"notes,omitempty"`
	HasMod      bool   `json:"has_mod,omitempty"` // the release also carries the Nickel mod
	Checked     string `json:"checked"`
}

func (layout Layout) availableFile() string { return filepath.Join(layout.workDir(), "available.json") }

// ReadAvailability returns the stored result of the last check; ok is false when none exists
// or it cannot be read (a damaged cache is the same as no check).
func ReadAvailability(layout Layout) (found Availability, ok bool) {
	data, err := os.ReadFile(layout.availableFile())
	if err != nil || json.Unmarshal(data, &found) != nil {
		return Availability{}, false
	}
	return found, true
}

func writeAvailability(layout Layout, found Availability) error {
	found.Checked = time.Now().UTC().Format(time.RFC3339)
	data, err := json.Marshal(found)
	if err != nil {
		return err
	}
	return WriteFileAtomically(layout.availableFile(), append(data, '\n'), 0o644)
}

func availabilityOf(manifest *release.Manifest, current string, isAvailable bool) Availability {
	found := Availability{Current: current}
	if !isAvailable {
		return found
	}
	_, hasMod := manifest.Find(release.ModPath)
	found.IsAvailable, found.Version, found.Notes, found.HasMod = true, manifest.Version, manifest.Notes, hasMod
	return found
}

// IsNewerThan reports whether the stored result still describes an update for the running
// version (an update applied since the check makes it stale).
func (found Availability) IsNewerThan(current string) bool {
	if !found.IsAvailable {
		return false
	}
	order, err := release.CompareVersions(found.Version, current)
	return err == nil && order > 0
}
