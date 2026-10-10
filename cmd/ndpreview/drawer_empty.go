package main

import (
	"io"

	"koboapps/internal/copperui"
)

// The drawer's own surfaces of docs/design-system.md A7 (mod/drawer/drawer_surfaces.cc,
// lib/copperui/include/copperui/empty_state.h), as the previews draw them:
//
//   - the empty state: centred in the page area, the stack starts EMPTY_SPACER below it: an
//     optional 96 px Tabler icon, the serif heading, a muted caption and, only when the host has
//     something to do (the model is loaded and offers the refresh item), one primary button;
//     "Loading…" is the heading alone;
//   - the Qt6 warning: a notice (2 px border, paper.notice fill, 24 px padding, body text) under
//     the header that takes the hint line's room and some of the page area;
//   - a problem row: no icon, the name bold on one line and the status as one muted caption
//     line, both elided; the full message is in the Info dialog.
const (
	drawerEmptyIconSide = copperui.IconGrid
	drawerEmptyGap      = rhythm
)

// drawerEmptyPage lays out the empty state; loading shows the heading alone.
func drawerEmptyPage(chrome drawerChrome, isLoading bool) Page {
	items := drawerHeaderItems(chrome.title, false, false)
	top := headerHeight + emptySpacerHeight
	if isLoading {
		items = append(items, paragraphBlock(chrome.loading, headingSize, headingLineH, familySerif, true, "center", inkBlack).draw(top)...)
		return Page{ScreenID: "drawer-empty", Items: finishPage(items, nil), Count: 1}
	}
	items = append(items, iconAt("apps", drawerEmptyIconSide, (screenWidth-drawerEmptyIconSide)/2, top, inkDim)...)
	top += drawerEmptyIconSide + drawerEmptyGap
	heading := paragraphBlock(chrome.emptyTitle, headingSize, headingLineH, familySerif, true, "center", inkBlack)
	items = append(items, heading.draw(top)...)
	top += heading.height + drawerEmptyGap
	caption := paragraphBlock(chrome.emptyHint, captionSize, captionLineH, familySans, false, "center", inkMuted)
	items = append(items, caption.draw(top)...)
	top += caption.height + drawerEmptyGap
	items = append(items, primaryButton(chrome.findApps, top)...)
	return Page{ScreenID: "drawer-empty", Items: finishPage(items, nil), Count: 1}
}

// primaryButton is the standard primary: content width, black fill, one centred label line.
func primaryButton(label string, top int) []Item {
	lines := wrapLimited(label, contentWidth-2*rhythm, bold(bodySize, familySans), buttonLines)
	height := buttonHeight
	if len(lines) > 1 {
		height = buttonTallHt
	}
	return buttonItems(buttonLayout{x: gutter, y: top, width: contentWidth, height: height}, lines, true, true, "", "", label)
}

// qt6NoticeItems draws the warning notice at the top of the page area and returns its height with
// the 24 px below it.
func qt6NoticeItems(text string) ([]Item, int) {
	inner := contentWidth - 2*noticePadding - 2*noticeBorder
	lines := wrapText(text, inner, regular(bodySize, familySans))
	height := 2*noticePadding + len(lines)*bodyLineH
	items := []Item{{Shape: "rect", X: gutter, Y: headerHeight, W: contentWidth, H: height, Fill: paperGrey, Border: noticeBorder},
		{Shape: "text", X: gutter + noticePadding, Y: headerHeight + noticePadding, W: inner, H: len(lines) * bodyLineH, Lines: lines,
			Size: bodySize, LineH: bodyLineH, Family: familySans, Ink: inkBlack, Fill: noFill}}
	return items, height + rhythm
}

// problemRowItems draws a problem row at y: bold name, caption status, one line each.
func problemRowItems(name, status string, y int) []Item {
	nameTop := y + (drawerRowHeight-rowLabelHeight-captionLineH)/2
	title := textItem(gutter, nameTop, contentWidth, rowLabelHeight, elide(name, contentWidth, bold(rowTextSize, familySans), false), rowTextSize, familySans, "start")
	title.Bold = true
	caption := textItem(gutter, nameTop+rowLabelHeight, contentWidth, captionLineH, elide(status, contentWidth, regular(captionSize, familySans), false), captionSize, familySans, "start")
	caption.Ink = inkMuted
	return []Item{title, caption, separator(y + drawerRowHeight - separatorPx)}
}

func runDrawerEmpty(args []string, stdout, stderr io.Writer) int {
	flags, ok := parseDrawerFlags("drawer-empty", args, stderr, 0)
	if !ok {
		return exitFailure
	}
	build := func() Page { return drawerEmptyPage(chromeFor(flags.locale), flags.loading) }
	return renderDrawerPage("drawer empty state", flags, build, stdout, stderr)
}
