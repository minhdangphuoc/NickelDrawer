package wallpaper

import (
	"koboapps/internal/device"
	"koboapps/internal/i18n"
)

// Size is a screen resolution in pixels (portrait).
type Size struct{ Width, Height int }

const deviceIDSuffixLength = 3

// resolutions maps the last three digits of the device id (see internal/device) to the screen
// size. Add a model only after checking its resolution; unknown models are refused instead of guessed.
var resolutions = map[string]Size{
	"388": {Width: 1264, Height: 1680}, // Libra 2
}

// ResolutionFor returns the screen size for a device id.
func ResolutionFor(deviceID string) (Size, error) {
	if len(deviceID) >= deviceIDSuffixLength {
		if size, found := resolutions[deviceID[len(deviceID)-deviceIDSuffixLength:]]; found {
			return size, nil
		}
	}
	return Size{}, i18n.NewError("msg.err_device_unknown", "model", device.ModelName(deviceID), "id", deviceID)
}

// ResolutionFromVersionFile reads .kobo/version and returns this device's screen size.
func ResolutionFromVersionFile(path string) (Size, error) {
	info, err := device.Read(path)
	if err != nil {
		return Size{}, i18n.NewError("msg.err_device_unreadable", "path", path).WithCause(err)
	}
	return ResolutionFor(info.ID)
}
