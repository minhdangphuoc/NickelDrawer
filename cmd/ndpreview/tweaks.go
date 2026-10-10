package main

import (
	"fmt"
	"io"

	"koboapps/internal/copperui"
)

// The "Tinh chỉnh" modal as the mod draws it today (mod/drawer/tweaks_overlay.cc,
// tweaks_panel.cc): N3Dialog's top row (title centred between two squares, ✕ at the right), a
// rule, two columns of 128 px cells with 1 px dividers (label, value under it) and a page label
// when there is more than one page. Its numbers are still "legacy" tokens until fix 3.2 moves it
// onto the dialog tokens; they live only in this block. The device draws the label in Georgia
// italic; the preview has no italic face and draws the regular serif.
const (
	tweaksBoxWidth    = copperui.LegacyTweaksBoxWidth
	tweaksBoxBorder   = copperui.LegacyTweaksBoxBorder
	tweaksCloseSide   = copperui.LegacyTweaksCloseSide
	tweaksTitleSize   = copperui.LegacyTweaksTitleSize
	tweaksLabelSize   = copperui.LegacyTweaksLabelSize
	tweaksValueSize   = copperui.LegacyTweaksValueSize
	tweaksCellHeight  = copperui.LegacyTweaksCellHeight
	tweaksCellPadding = copperui.LegacyTweaksCellPadding
	tweaksPageLabelH  = copperui.LegacyTweaksPageLabelHeight
	tweaksColumns     = 2 // TweaksLogic::COLUMN_COUNT
	tweaksRowsPerPage = 7 // TweaksLogic::ROWS_PER_PAGE
)

type tweakCell struct{ label, value string }

var sampleTweaks = map[string][]tweakCell{
	"en": {{"Sleep screen", "On"}, {"Screenshots", "Off"}, {"Auto USB connect", "Off"}, {"Text legibility", "Off"},
		{"IP address", "192.168.1.5"}, {"Invert and reboot", ""}, {"Sleep", ""}, {"Reboot", ""}, {"Rescan library", ""}},
	"vi": {{"Màn hình chờ", "Bật"}, {"Chụp màn hình", "Tắt"}, {"Tự động kết nối USB", "Tắt"}, {"Làm rõ chữ", "Tắt"},
		{"Địa chỉ IP", "192.168.1.5"}, {"Đảo màu và khởi động lại", ""}, {"Ngủ", ""}, {"Khởi động lại", ""}, {"Quét lại thư viện", ""}},
}

// tweaksPage draws the modal centred on the whole screen, over the white page.
func tweaksPage(cells []tweakCell, title string, pageIndex int) Page {
	perPage := tweaksColumns * tweaksRowsPerPage
	pages := max(1, (len(cells)+perPage-1)/perPage)
	pageIndex = clamp(pageIndex, 0, pages-1)
	shown := cells[pageIndex*perPage : min(len(cells), (pageIndex+1)*perPage)]
	firstPage := min(len(cells), perPage)
	rows := (firstPage + tweaksColumns - 1) / tweaksColumns // every page as high as the first
	gridHeight := rows * (tweaksCellHeight + hairline)
	height := 2*tweaksBoxBorder + tweaksCloseSide + hairline + gridHeight
	if pages > 1 {
		height += tweaksPageLabelH
	}
	x, y := (screenWidth-tweaksBoxWidth)/2, (screenHeight-height)/2
	inner := tweaksBoxWidth - 2*tweaksBoxBorder
	items := []Item{{Shape: "rect", X: x, Y: y, W: tweaksBoxWidth, H: height, Fill: paperFull, Border: tweaksBoxBorder, Ink: inkBlack}}
	top := y + tweaksBoxBorder
	titleWidth := inner - 2*tweaksCloseSide
	items = append(items, textItem(x+tweaksBoxBorder+tweaksCloseSide, top, titleWidth, tweaksCloseSide,
		elide(title, titleWidth, regular(tweaksTitleSize, familySans), false), tweaksTitleSize, familySans, "center"))
	items = append(items, iconCentred("close", chevronIconSize, x+tweaksBoxBorder+inner-tweaksCloseSide, top, tweaksCloseSide, tweaksCloseSide, inkBlack)...)
	top += tweaksCloseSide
	items = append(items, solidRect(x+tweaksBoxBorder, top, inner, hairline, lineRow))
	top += hairline
	leftWidth := (inner - hairline) / tweaksColumns
	for index, cell := range shown {
		column, row := index%tweaksColumns, index/tweaksColumns
		cellX := x + tweaksBoxBorder
		cellWidth := leftWidth
		if column == 1 {
			cellX += leftWidth + hairline
			cellWidth = inner - hairline - leftWidth
		}
		cellY := top + row*(tweaksCellHeight+hairline)
		textWidth := cellWidth - 2*tweaksCellPadding
		label := textItem(cellX+tweaksCellPadding, cellY+16, textWidth, 56, elide(cell.label, textWidth, regular(tweaksLabelSize, familySerif), false), tweaksLabelSize, familySerif, "start")
		value := textItem(cellX+tweaksCellPadding, cellY+16+56, textWidth, 40, elide(cell.value, textWidth, regular(tweaksValueSize, familySans), false), tweaksValueSize, familySans, "start")
		items = append(items, label, value)
		if column == 0 { // every row ends with a divider across the whole width, as the panel paints it
			items = append(items, solidRect(x+tweaksBoxBorder, cellY+tweaksCellHeight, inner, hairline, lineRow))
		}
		if column == 0 && index+1 < len(shown) {
			items = append(items, solidRect(cellX+cellWidth, cellY, hairline, tweaksCellHeight, lineRow))
		}
	}
	if pages > 1 {
		pageLabel := textItem(x+tweaksBoxBorder, top+gridHeight, inner, tweaksPageLabelH, fmt.Sprintf("%d / %d", pageIndex+1, pages), tweaksValueSize, familySans, "center")
		items = append(items, pageLabel)
	}
	return Page{ScreenID: "tweaks", Items: append(nickelChromeItems(), items...), Index: pageIndex, Count: pages}
}

func runTweaks(args []string, stdout, stderr io.Writer) int {
	flags, ok := parseDrawerFlags("tweaks", args, stderr, 9)
	if !ok {
		return exitFailure
	}
	build := func() Page {
		cells := sampleTweaks[flags.locale]
		if cells == nil {
			cells = sampleTweaks["en"]
		}
		return tweaksPage(cells[:min(len(cells), max(1, flags.count))], chromeFor(flags.locale).tweaksTitle, 0)
	}
	return renderDrawerPage("tweaks modal", flags, build, stdout, stderr)
}
