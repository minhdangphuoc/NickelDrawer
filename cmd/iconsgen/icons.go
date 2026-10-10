package main

import "strconv"

const (
	drawerIconSize = 96 // app icons and host item icons
	uiIconSmall    = 64
	uiIconLarge    = 96

	// A bit lighter than Tabler's default 1.5 on its 24-unit grid, closer to Nickel's thin line icons.
	tablerStroke = 1.25

	// Nickel's More-tab row icons are 50x50 with ~2 px strokes and ink from 2 px to the edge.
	moreRowIconSize = 50
	moreRowStroke   = 2.25
	moreRowInset    = 0.0 // whole grid: oksvg ignores a viewBox origin, so cropping cut off the right and bottom
)

// icon is one generated file: its path below the repo root, the Tabler outline icon it
// renders, its pixel size, its stroke weight in pixels and the grid inset (0 = whole grid).
type icon struct {
	path         string
	tabler       string
	size         int
	strokePixels float64
	inset        float64
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
	{"picture", "photo"},
	{"picture-off", "photo-off"},
	{"tweaks", "adjustments-horizontal"},
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
