package wallpaper

import (
	"os"
	"strings"
)

const powerOptionsSection = "[PowerOptions]"

// Nickel shows the screensaver folder only when both settings are on (docs/spec/wallpaper.md).
// They are the Kobo eReader.conf keys behind "Show current read" and "Show book cover full screen".
var sleepScreenKeys = []string{"ShowBookCover", "FullscreenCoverStretch"}

// SleepScreenReady reads the two Nickel settings that let the folder show. Missing keys, a missing
// file or an unreadable file count as off: the answer is unknown, so the user still gets the hint.
// The file is only read; Nickel rewrites it, so we never write it.
func SleepScreenReady(confPath string) (ready bool, missing []string) {
	enabled := readPowerOptions(confPath)
	for _, key := range sleepScreenKeys {
		if !enabled[key] {
			missing = append(missing, key)
		}
	}
	return len(missing) == 0, missing
}

// readPowerOptions returns the watched keys that are set to true in [PowerOptions].
func readPowerOptions(confPath string) map[string]bool {
	enabled := map[string]bool{}
	data, err := os.ReadFile(confPath)
	if err != nil {
		return enabled
	}
	inSection := false
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") {
			inSection = line == powerOptionsSection
			continue
		}
		name, value, found := strings.Cut(line, "=")
		if inSection && found {
			enabled[strings.TrimSpace(name)] = strings.EqualFold(strings.Trim(strings.TrimSpace(value), `"`), "true")
		}
	}
	return enabled
}
