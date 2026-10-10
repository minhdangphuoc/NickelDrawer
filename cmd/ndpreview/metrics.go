package main

import "koboapps/internal/copperui"

// Layout metrics in device pixels (Kobo Libra 2, 300 dpi). Every number is a CopperUI token
// (lib/copperui/tokens/copperui.tokens.json, docs/copperui-tokens.md), the same source as
// mod/drawer/screen_logic.h; the names here are the layout code's short aliases.
const (
	screenWidth  = copperui.ScreenWidth
	screenHeight = copperui.ScreenHeight
	gutter       = copperui.Gutter
	contentWidth = copperui.ContentWidth
	rhythm       = copperui.Rhythm

	// The header is Nickel's page header (TextHeader, as the drawer's): a 120 px row with the
	// back arrow, the title in Nickel's header sans and the status/action at the right, then a
	// 1 px divider.
	headerRowHeight  = copperui.HeaderRow
	headerDivider    = copperui.HeaderDivider
	headerHeight     = copperui.HeaderHeight
	headerIconWidth  = copperui.HeaderIcon
	headerTitleSz    = copperui.TypeHeaderTitleSize
	headerStatusSz   = copperui.TypeHeaderStatusSize
	headerSideWidth  = copperui.HeaderSideWidth
	headerDividerInk = copperui.LineHeader
	separatorPx      = copperui.StrokeRule

	rowHeight       = copperui.RowCompact // key-value, list, toggle and radio rows
	tapRowHeight    = copperui.RowTall    // a tappable "row" component: above the 120 px tap target
	fieldHeight     = copperui.FieldHeight
	fieldTextHeight = copperui.FieldTextHeight
	buttonHeight    = copperui.ButtonHeight
	buttonTallHt    = copperui.ButtonTallHeight // a button whose label wraps to two lines
	buttonLines     = copperui.ButtonMaxLines
	sectionHeight   = copperui.SectionHeight
	bottomMargin    = copperui.MarginBottom
	pageBarHeight   = copperui.PagerHeight

	bodySize     = copperui.TypeBodySize
	bodyLineH    = copperui.TypeBodyLine
	captionSize  = copperui.TypeCaptionSize
	captionLineH = copperui.TypeCaptionLine
	headingSize  = copperui.TypeHeadingSize
	headingLineH = copperui.TypeHeadingLine
	rowTextSize  = copperui.TypeRowSize
	sectionSize  = copperui.TypeSectionSize
	sectionLineH = copperui.TypeSectionLine

	bigCodeSize   = copperui.TypeBigCodeSize
	bigCodeHeight = copperui.TypeBigCodeLine
	qrSide        = copperui.QrSide
	noticePadding = copperui.NoticePadding
	noticeBorder  = copperui.StrokeRule
	keyColumn     = copperui.SpaceKeyColumn
	iconBoxSize   = copperui.LegacyDrawerRowIcon

	// Spacing inside rows.
	rowTextInset        = copperui.SpaceRowText
	rowTextInsetTall    = copperui.SpaceRowTextTall
	rowLabelWithDetail  = copperui.SpaceRowLabelWithDetail
	rowDetailTop        = copperui.SpaceRowDetailTop
	rowLabelHeight      = copperui.RowLabelHeight
	rowTapTextInset     = copperui.RowTapTextInset
	fieldCaptionGap     = copperui.SpaceFieldCaption
	rowDetailWidth      = copperui.RowDetailWidth
	rowDetailRight      = copperui.RowDetailRight
	sectionTopInset     = copperui.SpaceSection
	fieldBorder         = copperui.StrokeRule
	radioRingBorder     = copperui.StrokeSelect
	radioRingSide       = copperui.RadioRing
	radioDotSide        = copperui.RadioDot
	radioDotInset       = copperui.RadioDotInset
	buttonBorder        = copperui.StrokeButton
	dialogBorder        = copperui.StrokeButton
	qrFrameBorder       = copperui.StrokeRule
	emptySpacerHeight   = copperui.EmptySpacer
	galleryEmptySpacer  = copperui.GalleryEmptySpacer
	pictureStroke       = copperui.StrokeSelect
	badgeCheckThickness = 6

	dialogWidth = copperui.DialogWidth
	toastBottom = copperui.ToastBottom

	// Gallery and viewer (docs/spec/wallpaper-gallery.md section 2).
	galleryGutter       = copperui.GalleryGutter // the grid spans the content gutters: no side padding of its own
	galleryBorder       = copperui.StrokeTile
	galleryCurrentBdr   = copperui.StrokeSelect
	galleryBadgeSize    = copperui.IconBadge
	galleryBadgeInset   = copperui.GalleryBadgeInset
	galleryDashWidth    = copperui.StrokeRule
	galleryPagerHeight  = copperui.PagerHeight
	defaultGalleryCols  = copperui.GalleryColumns
	defaultGalleryRows  = copperui.GalleryRows
	headerIconTarget    = copperui.HeaderIconTarget // tap target of a header icon action
	headerIconGlyph     = copperui.HeaderIcon
	radioDetailRow      = copperui.RowTall // a radio option with a sub-line; a toggle with a detail line too
	viewerBarHeight     = copperui.ViewerBar
	viewerSecondaryWide = copperui.ViewerSecondary
	viewerBarGap        = copperui.Rhythm
	viewerBackTarget    = copperui.HeaderIconTarget // full view's back button: centred on the header's back arrow
	viewerBackDisc      = copperui.ViewerBackDisc   // the pressed circle of a 120 px icon target (16 px margin)
	viewerBackOutline   = copperui.ViewerBackOutline
	selectedVarName     = "selected" // receives the screen value, as in appsctl
	maxPictureBytes     = 8 << 20    // larger files are not decoded (mod and preview)
)

// Greys in 0 (black) .. 255 (white), all on the panel's 16-level grid. Only the notice box may
// have a grey fill; row dividers are lineRow, the section rule black.
const (
	inkBlack    = copperui.InkBlack
	inkMuted    = copperui.InkMuted
	inkDim      = copperui.InkDim
	lineRow     = copperui.LineRow
	lineSection = copperui.LineSection
	paperGrey   = copperui.PaperGrey
	paperFull   = copperui.PaperWhite
	noFill      = -1
)

// Text families map to the font stacks of the HTML page and the --font flags of render.
const (
	familySans  = "sans"
	familySerif = "serif"
	familyMono  = "mono"
)

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

// Toggle: Nickel's ToggleSwitch on the Libra 2 class (state word, 25 px, 94 x 56 switch image).
const (
	toggleSwitchWidth  = copperui.ToggleSwitchWidth
	toggleSwitchHeight = copperui.ToggleSwitchHeight
	toggleStateSize    = copperui.TypeToggleStateSize
	toggleStateGap     = copperui.ToggleStateGap
	toggleStateWidth   = copperui.ToggleStateWidth
)
