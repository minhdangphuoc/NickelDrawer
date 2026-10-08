// Package device identifies the Kobo model from /mnt/onboard/.kobo/version.
package device

import (
	"fmt"
	"os"
	"strings"
)

const (
	// DefaultVersionFile is where Nickel records serial, firmware and device id.
	DefaultVersionFile = "/mnt/onboard/.kobo/version"
	// FallbackName is used when the model is unknown or the file is unreadable.
	FallbackName = "Kobo"

	idSuffixLength = 3
	minFieldCount  = 6
	firmwareField  = 2
)

// modelNames maps the last three digits of the device id to a model. The ids come from
// FBInk (fbink.h, KOBO_DEVICE_ID_E; fbink_device_id.c reads the last 3 characters of the
// id field in .kobo/version the same way). Names follow FBInk's device names.
var modelNames = map[string]string{
	"310": "Kobo Touch",
	"330": "Kobo Glo",
	"340": "Kobo Mini",
	"350": "Kobo Aura HD",
	"360": "Kobo Aura",
	"370": "Kobo Aura H2O",
	"371": "Kobo Glo HD",
	"372": "Kobo Touch 2.0",
	"373": "Kobo Aura ONE",
	"374": "Kobo Aura H2O 2",
	"375": "Kobo Aura SE",
	"376": "Kobo Clara HD",
	"377": "Kobo Forma",
	"378": "Kobo Aura H2O 2 r2",
	"379": "Kobo Aura SE r2",
	"380": "Kobo Forma 32GB",
	"381": "Kobo Aura ONE LE",
	"382": "Kobo Nia",
	"383": "Kobo Sage",
	"384": "Kobo Libra H2O",
	"386": "Kobo Clara 2E",
	"387": "Kobo Elipsa",
	"388": "Kobo Libra 2",
	"389": "Kobo Elipsa 2E",
	"390": "Kobo Libra Colour",
	"391": "Kobo Clara BW",
	"393": "Kobo Clara Colour",
	"395": "Kobo Clara BW",
}

// Info is what the version file says about this device.
type Info struct {
	ID       string // full device id field
	Firmware string
	Model    string // model name, FallbackName when the id is unknown
}

// Parse reads one line of .kobo/version:
// serial,4.1.15,4.38.23828,4.1.15,4.1.15,00000000-0000-0000-0000-000000000388
func Parse(line string) (Info, error) {
	fields := strings.Split(strings.TrimSpace(line), ",")
	if len(fields) < minFieldCount {
		return Info{}, fmt.Errorf("version line has %d comma-separated fields, want at least %d", len(fields), minFieldCount)
	}
	id := fields[len(fields)-1]
	return Info{ID: id, Firmware: fields[firmwareField], Model: ModelName(id)}, nil
}

// ModelName maps a device id (or just its last digits) to a model name.
func ModelName(id string) string {
	if len(id) < idSuffixLength {
		return FallbackName
	}
	if name, found := modelNames[id[len(id)-idSuffixLength:]]; found {
		return name
	}
	return FallbackName
}

// Read parses the version file at path.
func Read(path string) (Info, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Info{}, fmt.Errorf("cannot read %s: %w", path, err)
	}
	return Parse(string(data))
}
