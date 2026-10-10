package main

import (
	"fmt"
	"io"
	"strings"
)

// The drawer's list view (mod/drawer/drawer_view.cc renderListPage, item_widgets.cc AppRow):
// 152 px rows with a 64 px icon, the name (36) and a status caption (32, muted) under it, the
// update dot and, for rows that do something, a Tabler chevron at the right edge; a section label
// (28 bold, uppercase, muted, 64 px block with the black rule) starts the page and every section.

func drawerRowItems(item drawerItem, y int) []Item {
	var items []Item
	iconTop := y + (drawerRowHeight-rowIconSize)/2
	if icon, ok := drawerRowIcon(item.iconPath, gutter, iconTop); ok {
		items = append(items, icon...)
	}
	textX := gutter + rowIconSize + rhythm
	right := gutter + contentWidth
	if item.hasAction {
		right -= chevronIconSize + rhythm
	}
	if item.hasUpdate {
		right -= updateDotSide + rhythm
	}
	textWidth := right - textX
	nameTop := y + (drawerRowHeight-rowLabelHeight)/2
	if item.status != "" {
		nameTop = y + (drawerRowHeight-rowLabelHeight-captionLineH)/2
	}
	items = append(items, textItem(textX, nameTop, textWidth, rowLabelHeight, elide(item.name, textWidth, regular(rowTextSize, familySans), false), rowTextSize, familySans, "start"))
	if item.status != "" {
		status := textItem(textX, nameTop+rowLabelHeight, textWidth, captionLineH, elide(item.status, textWidth, regular(captionSize, familySans), false), captionSize, familySans, "start")
		status.Ink = inkMuted
		items = append(items, status)
	}
	if item.hasUpdate {
		items = append(items, Item{Shape: "ellipse", X: right + rhythm - 0, Y: y + (drawerRowHeight-updateDotSide)/2, W: updateDotSide, H: updateDotSide, Fill: inkBlack, Ink: inkBlack, Label: "update"})
	}
	if item.hasAction {
		items = append(items, iconCentred("chevron-right", chevronIconSize, gutter+contentWidth-chevronIconSize, y, chevronIconSize, drawerRowHeight, inkBlack)...)
	}
	return append(items, separator(y+drawerRowHeight-separatorPx))
}

// drawerRowIcon draws the app's 64 px icon (icon-64.png beside icon.png) or the generic apps-64.
func drawerRowIcon(iconPath string, x, y int) ([]Item, bool) {
	if iconPath != "" {
		sibling := strings.TrimSuffix(iconPath, ".png") + "-64.png"
		if picture, loaded := appIconItem(sibling, rowIconSize, x, y); loaded {
			return []Item{picture}, true
		}
	}
	generic := iconAt("apps", rowIconSize, x, y, inkBlack)
	return generic, len(generic) > 0
}

func sectionItems(label string, y int) []Item {
	upper := strings.ToUpper(label)
	text := textItem(gutter, y+sectionTopInset, contentWidth, sectionLineH, elide(upper, contentWidth, bold(sectionSize, familySans), false), sectionSize, familySans, "start")
	text.Ink, text.Bold = inkMuted, true
	return []Item{text, sectionRule(y + sectionHeight - separatorPx)}
}

// drawerListPage lays out one page of the list view: rows in one section.
func drawerListPage(items []drawerItem, chrome drawerChrome, pageIndex int, extras drawerListExtras) Page {
	layout := drawerHeaderItems(chrome.title, false, true)
	y := headerHeight
	hint := chrome.hint
	if extras.hasQt6Notice { // the notice takes the hint line's room and some rows (DrawerLogic::listRowsThatFit)
		notice, taken := qt6NoticeItems(chrome.qt6Notice)
		layout = append(layout, notice...)
		y += taken
		hint = ""
	}
	rowsPerPage := listRowsThatFit(drawerPageArea + drawerHintHeight*boolToInt(extras.hasQt6Notice) - (y - headerHeight))
	pages := max(1, (len(items)+rowsPerPage-1)/rowsPerPage)
	pageIndex = clamp(pageIndex, 0, pages-1)
	layout = append(layout, sectionItems(chrome.section, y)...)
	y += sectionHeight
	for index := pageIndex * rowsPerPage; index < min(len(items), (pageIndex+1)*rowsPerPage); index++ {
		layout = append(layout, drawerRowItems(items[index], y)...)
		y += drawerRowHeight
	}
	if extras.hasProblem && pageIndex == pages-1 && y+sectionHeight+drawerRowHeight <= headerHeight+drawerPageArea {
		layout = append(layout, sectionItems(chrome.problemsSection, y)...)
		layout = append(layout, problemRowItems(chrome.problemName, chrome.problemStatus, y+sectionHeight)...)
	}
	layout = append(layout, drawerFooterItems(hint, pageIndex+1, pages)...)
	return Page{ScreenID: "drawer-list", Items: finishPage(layout, nil), Index: pageIndex, Count: pages}
}

// drawerListExtras are the optional surfaces of a list preview.
type drawerListExtras struct{ hasProblem, hasQt6Notice bool }

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

// listRowsThatFit is DrawerLogic::listRowsThatFit: rows of drawerRowHeight under one section label,
// at least 1 and at most drawerListRows.
func listRowsThatFit(available int) int {
	return clamp((available-sectionHeight)/drawerRowHeight, 1, drawerListRows)
}

func runDrawerList(args []string, stdout, stderr io.Writer) int {
	flags, ok := parseDrawerFlags("drawer-list", args, stderr, 8)
	if !ok {
		return exitFailure
	}
	items, err := drawerItemsFor(flags, "icon.png")
	if err != nil {
		fmt.Fprintln(stderr, "ndpreview:", err)
		return exitFailure
	}
	build := func() Page {
		extras := drawerListExtras{hasProblem: flags.problems, hasQt6Notice: flags.qt6Notice}
		return drawerListPage(items, chromeFor(flags.locale), 0, extras)
	}
	return renderDrawerPage("drawer list", flags, build, stdout, stderr)
}
