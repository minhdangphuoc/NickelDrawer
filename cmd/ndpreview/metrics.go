package main

// Layout metrics in device pixels (Kobo Libra 2, 300 dpi). They mirror docs/spec/ui-kit.md
// section 2; change them there first.
const (
	screenWidth  = 1264
	screenHeight = 1680
	gutter       = 72
	contentWidth = screenWidth - 2*gutter
	rhythm       = 24

	headerHeight   = 168
	headerTitleSz  = 60
	headerStatusSz = 32
	separatorPx    = 2

	rowHeight     = 104 // key-value, list, toggle and radio rows
	tapRowHeight  = 128 // a tappable "row" component: above the 120 px tap target
	fieldHeight   = 120
	buttonHeight  = 120
	buttonTallHt  = 168 // a button whose label wraps to two lines
	sectionHeight = 64
	bottomMargin  = gutter
	pageBarHeight = 56

	bodySize     = 40
	bodyLineH    = 56
	captionSize  = 32
	captionLineH = 44
	headingSize  = 52
	headingLineH = 72
	rowTextSize  = 36

	bigCodeSize   = 144
	bigCodeHeight = 200
	qrSide        = 480
	noticePadding = 24
	noticeBorder  = separatorPx
	iconBoxSize   = 52

	dialogWidth = 1040
	toastBottom = 260
)

// Greys in 0 (black) .. 255 (white). Only the notice box may have a grey fill.
const (
	inkBlack  = 0
	inkMuted  = 64
	inkDim    = 140
	paperGrey = 232
	paperFull = 255
	noFill    = -1
)

// Text families map to the font stacks of the HTML page and the --font flags of render.
const (
	familySans  = "sans"
	familySerif = "serif"
	familyMono  = "mono"
)

// Average glyph advance as a fraction of the font size; good enough to wrap and elide text
// the same way in the HTML and PNG previews without measuring real glyphs.
var advanceByFamily = map[string]float64{familySans: 0.56, familySerif: 0.52, familyMono: 0.6}

// hostStrings are the texts the host (not the app) draws on a screen.
var hostStrings = map[string]map[string]string{
	"en": {"on": "On", "off": "Off", "cancel": "Cancel", "ok": "OK", "back": "Back", "stale": "Not set"},
	"vi": {"on": "Bật", "off": "Tắt", "cancel": "Hủy", "ok": "OK", "back": "Quay lại", "stale": "Chưa đặt"},
}

func hostText(locale, key string) string {
	if texts, found := hostStrings[locale]; found {
		if text, has := texts[key]; has {
			return text
		}
	}
	return hostStrings["en"][key]
}
