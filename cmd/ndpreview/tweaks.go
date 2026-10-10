package main

import (
	"fmt"
	"io"
	"strings"

	"koboapps/internal/copperui"
)

// The "Tinh chỉnh" modal (mod/drawer/tweaks_overlay.cc, tweaks_panel.cc, tweaks_logic.cc) on the
// dialog tokens (design-system A6): a 1040 px box with a 3 px border centred on the whole screen,
// a 120 px title row (serif dialog title between two 120 px squares, the Nickel ✕ at the right), a
// 1 px rule, two columns of 128 px cells with 1 px dividers and a pager label when there is more
// than one page. Cell text is the sans at the row size, measured, never serif italic:
//
//   - action cell (Sleep, Reboot): the label centred, one line or two centred lines, then elided;
//   - toggle cell: the label at the left (one or two lines), Nickel's switch image at the right
//     with the state word under it, both vertically centred (a 516 px cell has no room for the
//     word beside the switch as Settings rows have it);
//   - info cell (IP address): the label on one line with the value as a muted caption under it;
//   - any cell with a caption (Screenshots): the label on one line and the caption muted under it,
//     as the info cell stacks them; a toggle keeps its switch at the right, an action stays centred.
const (
	tweaksCellHeight  = copperui.TweaksCellHeight
	tweaksCellPadding = copperui.TweaksCellPadding
	tweaksColumns     = copperui.TweaksColumns
	tweaksRowsPerPage = copperui.TweaksRowsPerPage
	tweaksLabelLines  = 2 // BUTTON_MAX_LINES: a label takes at most two lines
	tweaksStateGap    = 4 // switch image to its state word
	tweaksStateHeight = 40
)

type tweakKind int

const (
	tweakAction tweakKind = iota
	tweakToggle
	tweakInfo
)

type tweakCell struct {
	label, value string
	kind         tweakKind
	isOn         bool
	caption      string // optional second line under the label (docs/spec/tweaks.md)
}

var sampleTweaks = map[string][]tweakCell{
	"en": {
		{label: "Sleep screen", kind: tweakToggle, isOn: true},
		{label: "Screenshots", kind: tweakToggle, caption: "With the power button"},
		{label: "Screenshot button", kind: tweakToggle, isOn: true, caption: "In the status bar"},
		{label: "Auto USB connect", kind: tweakToggle},
		{label: "Text legibility", kind: tweakToggle},
		{label: "IP address", value: "192.168.1.5", kind: tweakInfo},
		{label: "Invert and reboot", kind: tweakAction},
		{label: "Sleep", kind: tweakAction},
		{label: "Reboot", kind: tweakAction},
		{label: "Rescan library", kind: tweakAction},
	},
	"vi": {
		{label: "Màn hình chờ", kind: tweakToggle, isOn: true},
		{label: "Chụp màn hình", kind: tweakToggle, caption: "Bằng nút nguồn"},
		{label: "Nút chụp màn hình", kind: tweakToggle, isOn: true, caption: "Trên thanh trạng thái"},
		{label: "Tự động kết nối USB", kind: tweakToggle},
		{label: "Làm rõ chữ", kind: tweakToggle},
		{label: "Địa chỉ IP", value: "192.168.1.5", kind: tweakInfo},
		{label: "Đảo màu và khởi động lại", kind: tweakAction},
		{label: "Ngủ", kind: tweakAction},
		{label: "Khởi động lại", kind: tweakAction},
		{label: "Quét lại thư viện", kind: tweakAction},
	},
}

// tweaksCellItems draws one cell whose top-left corner is (cellX, cellY) and whose width is cellWidth.
func tweaksCellItems(cell tweakCell, cellX, cellY, cellWidth int, locale string) []Item {
	inner := cellWidth - 2*tweaksCellPadding
	left := cellX + tweaksCellPadding
	switch {
	case cell.kind == tweakToggle:
		return tweakToggleItems(cell, left, cellY, inner, locale)
	case cell.kind == tweakInfo:
		return tweakInfoItems(cell, left, cellY, inner)
	case cell.caption != "":
		return tweakCaptionedItems(cell.label, cell.caption, left, cellY, inner, "center")
	}
	return []Item{tweakLabelItem(cell.label, left, cellY, inner, "center")}
}

// tweakCaptionedItems stacks a one-line label and its muted one-line caption, the pair centred
// vertically (the info cell's block).
func tweakCaptionedItems(label, caption string, left, cellY, width int, align string) []Item {
	top := cellY + (tweaksCellHeight-rowLabelHeight-captionLineH)/2
	labelItem := textItem(left, top, width, rowLabelHeight, elide(label, width, regular(rowTextSize, familySans), false), rowTextSize, familySans, align)
	captionItem := textItem(left, top+rowLabelHeight, width, captionLineH, elide(caption, width, regular(captionSize, familySans), false), captionSize, familySans, align)
	captionItem.Ink = inkMuted
	return []Item{labelItem, captionItem}
}

// tweakLabelItem is the label as one block of at most two lines, vertically centred in the cell.
func tweakLabelItem(label string, left, cellY, width int, align string) Item {
	return tweakLabelLines(wrapLimited(label, width, regular(rowTextSize, familySans), tweaksLabelLines), left, cellY, width, align)
}

func tweakLabelLines(lines []string, left, cellY, width int, align string) Item {
	height := len(lines) * rowLabelHeight
	return Item{Shape: "text", X: left, Y: cellY + (tweaksCellHeight-height)/2, W: width, H: height, Lines: lines, Size: rowTextSize,
		LineH: rowLabelHeight, Family: familySans, Align: align, Ink: inkBlack, Fill: noFill}
}

func tweakToggleItems(cell tweakCell, left, cellY, inner int, locale string) []Item {
	switchX := left + inner - toggleSwitchWidth
	blockHeight := toggleSwitchHeight + tweaksStateGap + tweaksStateHeight
	switchY := cellY + (tweaksCellHeight-blockHeight)/2
	state := hostText(locale, "off")
	if cell.isOn {
		state = hostText(locale, "on")
	}
	word := textItem(switchX, switchY+toggleSwitchHeight+tweaksStateGap, toggleSwitchWidth, tweaksStateHeight, strings.ToUpper(state), toggleStateSize, familySans, "center")
	labelWidth := inner - toggleSwitchWidth - rhythm
	var texts []Item
	if cell.caption != "" {
		texts = tweakCaptionedItems(cell.label, cell.caption, left, cellY, labelWidth, "start")
	} else {
		texts = []Item{tweakLabelItem(cell.label, left, cellY, labelWidth, "start")}
	}
	items := append(texts, word)
	return append(items, switchItems(switchX, switchY, cell.isOn, inkBlack)...)
}

func tweakInfoItems(cell tweakCell, left, cellY, inner int) []Item {
	return tweakCaptionedItems(cell.label, cell.value, left, cellY, inner, "start")
}

// tweaksPage draws the modal centred on the whole screen, over the white page.
func tweaksPage(cells []tweakCell, title, locale string, pageIndex int) Page {
	perPage := tweaksColumns * tweaksRowsPerPage
	pages := max(1, (len(cells)+perPage-1)/perPage)
	pageIndex = clamp(pageIndex, 0, pages-1)
	shown := cells[pageIndex*perPage : min(len(cells), (pageIndex+1)*perPage)]
	firstPage := min(len(cells), perPage)
	rows := (firstPage + tweaksColumns - 1) / tweaksColumns // every page as high as the first
	gridHeight := rows * (tweaksCellHeight + hairline)
	height := 2*dialogBorder + headerIconTarget + hairline + gridHeight
	if pages > 1 {
		height += pageBarHeight
	}
	x, y := (screenWidth-dialogWidth)/2, (screenHeight-height)/2
	inner := dialogWidth - 2*dialogBorder
	items := []Item{{Shape: "rect", X: x, Y: y, W: dialogWidth, H: height, Fill: paperFull, Border: dialogBorder, Ink: inkBlack}}
	top := y + dialogBorder
	titleWidth := inner - 2*headerIconTarget
	titleItem := textItem(x+dialogBorder+headerIconTarget, top+(headerIconTarget-dialogTitleLineH)/2, titleWidth, dialogTitleLineH,
		elide(title, titleWidth, bold(dialogTitleSize, familySerif), false), dialogTitleSize, familySerif, "center")
	titleItem.Bold = true
	items = append(items, titleItem)
	items = append(items, iconCentred("close", chevronIconSize, x+dialogBorder+inner-headerIconTarget, top, headerIconTarget, headerIconTarget, inkBlack)...)
	top += headerIconTarget
	items = append(items, solidRect(x+dialogBorder, top, inner, hairline, lineRow))
	top += hairline
	leftWidth := (inner - hairline) / tweaksColumns
	for index, cell := range shown {
		column, row := index%tweaksColumns, index/tweaksColumns
		cellX, cellWidth := x+dialogBorder, leftWidth
		if column == 1 {
			cellX += leftWidth + hairline
			cellWidth = inner - hairline - leftWidth
		}
		cellY := top + row*(tweaksCellHeight+hairline)
		items = append(items, tweaksCellItems(cell, cellX, cellY, cellWidth, locale)...)
		if column == 0 { // every row ends with a divider across the whole width, as the panel paints it
			items = append(items, solidRect(x+dialogBorder, cellY+tweaksCellHeight, inner, hairline, lineRow))
		}
		if column == 0 && index+1 < len(shown) {
			items = append(items, solidRect(cellX+cellWidth, cellY, hairline, tweaksCellHeight, lineRow))
		}
	}
	if pages > 1 {
		pageLabel := textItem(x+dialogBorder, top+gridHeight, inner, pageBarHeight, fmt.Sprintf("%d / %d", pageIndex+1, pages), captionSize, familySans, "center")
		pageLabel.Ink = inkMuted
		items = append(items, pageLabel)
	}
	return Page{ScreenID: "tweaks", Items: append(nickelChromeItems(), items...), Index: pageIndex, Count: pages}
}

func runTweaks(args []string, stdout, stderr io.Writer) int {
	flags, ok := parseDrawerFlags("tweaks", args, stderr, len(sampleTweaks["en"]))
	if !ok {
		return exitFailure
	}
	build := func() Page {
		cells := sampleTweaks[flags.locale]
		if cells == nil {
			cells = sampleTweaks["en"]
		}
		return tweaksPage(cells[:min(len(cells), max(1, flags.count))], chromeFor(flags.locale).tweaksTitle, flags.locale, 0)
	}
	return renderDrawerPage("tweaks modal", flags, build, stdout, stderr)
}
