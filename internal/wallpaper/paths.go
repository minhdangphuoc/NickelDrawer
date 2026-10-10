// Package wallpaper turns pictures from the user's Pictures folder into e-ink sleep-screen images
// that Nickel shows from /mnt/onboard/.kobo/screensaver (see docs/spec/wallpaper.md).
package wallpaper

import (
	"path/filepath"

	"koboapps/internal/storage"
)

const (
	// OwnedPrefix marks every file we put into ScreensaverFolder; nothing else there is touched.
	OwnedPrefix = "nd-wallpaper-"

	maxSourceBytes = 20 << 20
	maxPixels      = 48_000_000
)

// ScreensaverFolder is the only system-side place we write; Nickel picks sleep images here
// (/mnt/onboard/.kobo/screensaver on the device).
func ScreensaverFolder() string { return filepath.Join(storage.Root(), ".kobo", "screensaver") }

// DefaultSources is the hardcoded allowlist, the direct children of Pictures. It is deliberately not configurable.
func DefaultSources() []string {
	return []string{storage.Path(storage.Pictures)}
}

// Paths locates the app's own state.
type Paths struct{ AppDir string }

func (paths Paths) Data() string        { return filepath.Join(paths.AppDir, "data") }
func (paths Paths) Images() string      { return filepath.Join(paths.Data(), "images") }
func (paths Paths) Index() string       { return filepath.Join(paths.Images(), "index.json") }
func (paths Paths) Config() string      { return filepath.Join(paths.Data(), "config.json") }
func (paths Paths) Log() string         { return filepath.Join(paths.Data(), "log.txt") }
func (paths Paths) Image(id int) string { return filepath.Join(paths.Images(), imageFileName(id)) }

// Thumbs holds one small preview per picture for the gallery.
func (paths Paths) Thumbs() string      { return filepath.Join(paths.Data(), "thumbs") }
func (paths Paths) Thumb(id int) string { return filepath.Join(paths.Thumbs(), imageFileName(id)) }

// Removed lists the hashes of pictures the user removed, so an import does not bring them back.
func (paths Paths) Removed() string { return filepath.Join(paths.Data(), "removed.json") }
