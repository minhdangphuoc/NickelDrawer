package main

import (
	"math"
	"strconv"
)

const (
	drawerIconSize    = 96 // app icons and host item icons (grid cells, empty states)
	drawerRowIconSize = 64 // the same icons in a drawer list row, drawn 1:1 (icon.drawerRow)
	uiIconSmall       = 64
	uiIconLarge       = 96

	// A bit lighter than Tabler's default 1.5 on its 24-unit grid, closer to Nickel's thin line icons.
	tablerStroke = 1.25

	// Nickel's More-tab row icons are 50x50 with ~2 px strokes and ink from 2 px to the edge.
	moreRowIconSize = 50
	moreRowStroke   = 2.25
	moreRowInset    = 0.0 // whole grid: oksvg ignores a viewBox origin, so cropping cut off the right and bottom
)

// Nickel's own status-bar icons (the brightness, search and rotate glyphs), measured by decoding
// their images from libnickel's resources on the five target builds, which all agree: an 80 x 80
// canvas, a black glyph about 44 px across (55 % of the canvas, centred) drawn with a 3 px
// stroke, and a pressed "_hit" look that is the glyph in white on a black disc 80 px across.
// Only these numbers are kept here; no Kobo image is copied into the repository.
const (
	statusBarCanvasSize = 80
	statusBarGlyphInk   = 44.0
	statusBarStroke     = 3.0
	// The Tabler "screenshot" outline spans 16 grid units between its stroke centre lines (5..21).
	screenshotGridSpan = 16.0
)

// statusBarGlyphSize is the pixel size the whole 24-unit Tabler grid is rendered at so that the
// ink of the screenshot glyph (16 units plus one stroke) is statusBarGlyphInk px across.
var statusBarGlyphSize = int(math.Round((statusBarGlyphInk - statusBarStroke) / screenshotGridSpan * tablerGrid))

// icon is one generated file: its path below the repo root, the Tabler outline icon it
// renders, its pixel size, its stroke weight in pixels and the grid inset (0 = whole grid).
type icon struct {
	path         string
	tabler       string
	size         int
	strokePixels float64
	inset        float64
}

// canvasSizes lists the files whose glyph is centred (by its ink) on a larger transparent square
// so they can be drawn 1:1; the icon's size is then the glyph's own render size.
var canvasSizes = map[string]int{"helpers/extras/icons/screenshot.png": statusBarCanvasSize}

// outputSize is the side of the PNG that is written.
func (entry icon) outputSize() int {
	if canvas, found := canvasSizes[entry.path]; found {
		return canvas
	}
	return entry.size
}

// uiIcon is a generic UI icon, rendered at both UI sizes into host/icons/ui/.
type uiIcon struct{ name, tabler string }

// Names follow the ui-kit header icons (import, settings, refresh, add) where one exists.
var uiIcons = []uiIcon{
	{"settings", "settings"},
	{"import", "download"},
	{"refresh", "refresh"},
	{"add", "plus"},
	{"back", "arrow-left"},
	{"close", "x"},
	{"list", "layout-list"},
	{"grid", "layout-grid"},
	{"check", "check"},
	{"chevron-left", "chevron-left"},
	{"chevron-right", "chevron-right"},
	{"picture", "photo"},
	{"picture-off", "photo-off"},
	{"apps", "apps"},
	// Fallback of the app screens' toggle when Nickel's switch_on/off images are missing.
	{"toggle-on", "toggle-right"},
	{"toggle-off", "toggle-left"},
}

var icons = buildIconTable()

func buildIconTable() []icon {
	table := []icon{
		{"apps/localsend/icon.png", "send", drawerIconSize, tablerStroke * drawerIconSize / tablerGrid, 0},
		{"apps/wallpaper/icon.png", "photo", drawerIconSize, tablerStroke * drawerIconSize / tablerGrid, 0},
		{"apps/library/icon.png", "books", drawerIconSize, tablerStroke * drawerIconSize / tablerGrid, 0},
		{"apps/todo/icon.png", "list-check", drawerIconSize, tablerStroke * drawerIconSize / tablerGrid, 0},
		{"apps/uikit/icon.png", "layout-dashboard", drawerIconSize, tablerStroke * drawerIconSize / tablerGrid, 0},
		{"apps/ghsync/icon.png", "brand-github", drawerIconSize, tablerStroke * drawerIconSize / tablerGrid, 0},
		{"apps/localsend/icon-64.png", "send", drawerRowIconSize, tablerStroke * drawerRowIconSize / tablerGrid, 0},
		{"apps/wallpaper/icon-64.png", "photo", drawerRowIconSize, tablerStroke * drawerRowIconSize / tablerGrid, 0},
		{"apps/library/icon-64.png", "books", drawerRowIconSize, tablerStroke * drawerRowIconSize / tablerGrid, 0},
		{"apps/todo/icon-64.png", "list-check", drawerRowIconSize, tablerStroke * drawerRowIconSize / tablerGrid, 0},
		{"apps/uikit/icon-64.png", "layout-dashboard", drawerRowIconSize, tablerStroke * drawerRowIconSize / tablerGrid, 0},
		{"apps/ghsync/icon-64.png", "brand-github", drawerRowIconSize, tablerStroke * drawerRowIconSize / tablerGrid, 0},
		// The Extras helper's status-bar button, drawn 1:1 beside Nickel's own status-bar icons.
		{"helpers/extras/icons/screenshot.png", "screenshot", statusBarGlyphSize, statusBarStroke, 0},
		// The helper's "Tweaks" row in Nickel's More tab, like host/icons/apps.png.
		{"helpers/extras/icons/tweaks.png", "adjustments-horizontal", moreRowIconSize, moreRowStroke, moreRowInset},
		{"host/icons/refresh.png", "refresh", drawerIconSize, tablerStroke * drawerIconSize / tablerGrid, 0},
		{"host/icons/updates.png", "cloud-download", drawerIconSize, tablerStroke * drawerIconSize / tablerGrid, 0},
		// The "Apps" row in Nickel's More tab (mod/drawer/entry_row.cc), next to Nickel's own rows.
		{"host/icons/apps.png", "apps", moreRowIconSize, moreRowStroke, moreRowInset},
	}
	for _, ui := range uiIcons {
		for _, size := range []int{uiIconSmall, uiIconLarge} {
			path := "host/icons/ui/" + ui.name + "-" + strconv.Itoa(size) + ".png"
			table = append(table, icon{path, ui.tabler, size, tablerStroke * float64(size) / tablerGrid, 0})
		}
	}
	return table
}
