package main

import (
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// Icons are never drawn here (docs/DECISIONS.md): they are the repo's Tabler PNGs,
// host/icons/ui/<name>-<64|96>.png, composited 1:1 like the device does when Nickel has no image
// of its own. The device prefers Nickel's resources for the header icons, back arrow, check badge
// and switch (re/notes/system-icons.md); the preview cannot ship Kobo images, so it shows the
// Tabler fallback for all of them. A missing folder or file is reported once on iconWarnings and
// the icon is left out.

const iconFolder = "host/icons/ui"

// iconWarnings receives the one-line problems of icon loading (stderr in the CLI).
var iconWarnings io.Writer = os.Stderr

var iconState struct {
	mutex   sync.Mutex
	dir     string
	warned  map[string]bool
	cache   map[string]*image.NRGBA
	checked bool
}

// setIconDir points the preview at a folder of <name>-<size>.png files ("" searches the repo).
func setIconDir(dir string) {
	iconState.mutex.Lock()
	defer iconState.mutex.Unlock()
	iconState.dir, iconState.checked = dir, false
	iconState.warned, iconState.cache = map[string]bool{}, map[string]*image.NRGBA{}
}

func init() { setIconDir("") }

// findIconDir walks up from start looking for host/icons/ui.
func findIconDir(start string) string {
	for dir := start; dir != "" && dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, filepath.FromSlash(iconFolder))
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	return ""
}

// resolveIconDir returns the icon folder, searching from the working directory and then from the
// executable on first use.
func resolveIconDir() string {
	if iconState.dir != "" || iconState.checked {
		return iconState.dir
	}
	iconState.checked = true
	if workDir, err := os.Getwd(); err == nil {
		iconState.dir = findIconDir(workDir)
	}
	if executable, err := os.Executable(); err == nil && iconState.dir == "" {
		iconState.dir = findIconDir(filepath.Dir(executable))
	}
	if iconState.dir == "" {
		fmt.Fprintf(iconWarnings, "ndpreview: warning: no %s folder found above the working directory or the program; icons are left out (run from the repository or pass --icons DIR)\n", iconFolder)
	}
	return iconState.dir
}

func warnIconOnce(key, message string) {
	if !iconState.warned[key] {
		iconState.warned[key] = true
		fmt.Fprintln(iconWarnings, "ndpreview: warning:", message)
	}
}

// loadIcon reads <name>-<size>.png; white is the same strokes in white for use on black.
func loadIcon(name string, size int, white bool) (*image.NRGBA, bool) {
	iconState.mutex.Lock()
	defer iconState.mutex.Unlock()
	dir := resolveIconDir()
	if dir == "" {
		return nil, false
	}
	key := fmt.Sprintf("%s-%d-%t", name, size, white)
	if cached, has := iconState.cache[key]; has {
		return cached, cached != nil
	}
	path := filepath.Join(dir, fmt.Sprintf("%s-%d.png", name, size))
	loaded := readIcon(path)
	if loaded == nil {
		warnIconOnce(path, fmt.Sprintf("icon %s is missing or unreadable (run go run ./cmd/iconsgen -repo .)", path))
	} else if white {
		loaded = whiteIcon(loaded)
	}
	iconState.cache[key] = loaded
	return loaded, loaded != nil
}

func readIcon(path string) *image.NRGBA {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()
	decoded, err := png.Decode(file)
	if err != nil {
		return nil
	}
	result := image.NewNRGBA(decoded.Bounds())
	draw.Draw(result, result.Bounds(), decoded, decoded.Bounds().Min, draw.Src)
	return result
}

func whiteIcon(source *image.NRGBA) *image.NRGBA {
	result := image.NewNRGBA(source.Bounds())
	for index := 0; index+3 < len(source.Pix); index += 4 {
		result.Pix[index], result.Pix[index+1], result.Pix[index+2], result.Pix[index+3] = 255, 255, 255, source.Pix[index+3]
	}
	return result
}

// iconAt is the icon drawn 1:1 with its top-left corner at (x, y); ink 0 is black strokes, a
// lighter ink fades them (the device paints black icons with opacity for dim ones). No items
// when the icon cannot be loaded.
func iconAt(name string, size, x, y, ink int) []Item {
	icon, loaded := loadIcon(name, size, false)
	if !loaded {
		return nil
	}
	return []Item{{Shape: "icon", X: x, Y: y, W: size, H: size, Fill: noFill, Ink: ink, Icon: icon, Label: name}}
}

// whiteIconAt is iconAt in white strokes, for black discs.
func whiteIconAt(name string, size, x, y int) []Item {
	icon, loaded := loadIcon(name, size, true)
	if !loaded {
		return nil
	}
	return []Item{{Shape: "icon", X: x, Y: y, W: size, H: size, Fill: noFill, Ink: inkBlack, Icon: icon, Label: name}}
}

// iconCentred draws the icon centred in a box.
func iconCentred(name string, size, boxX, boxY, boxW, boxH, ink int) []Item {
	return iconAt(name, size, boxX+(boxW-size)/2, boxY+(boxH-size)/2, ink)
}

// headerIconNames maps a ui.json header icon to its Tabler file.
var headerIconNames = map[string]string{"import": "import", "settings": "settings", "refresh": "refresh", "add": "add"}

// headerIconItems draws the icon (64 px) centred in a headerIconTarget square at (x, y) and the
// tap target over it.
func headerIconItems(icon string, x, y int, event, label string) []Item {
	items := iconCentred(headerIconNames[icon], headerIconGlyph, x, y, headerIconTarget, headerIconTarget, inkBlack)
	return append(items, Item{Shape: "rect", X: x, Y: y, W: headerIconTarget, H: headerIconTarget, Fill: noFill, Event: event, Label: label})
}

// pictureGlyphItems is the missing-picture icon (Tabler photo, 96 px) centred on a point.
func pictureGlyphItems(centerX, centerY, ink int) []Item {
	return iconAt("picture", gridIconBox, centerX-gridIconBox/2, centerY-gridIconBox/2, ink)
}
