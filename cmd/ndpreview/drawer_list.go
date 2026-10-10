package main

import (
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
func drawerListPage(items []drawerItem, chrome drawerChrome, pageIndex int) Page {
	pages := max(1, (len(items)+drawerListRows-1)/drawerListRows)
	pageIndex = clamp(pageIndex, 0, pages-1)
	layout := drawerHeaderItems(chrome.title, false, true)
	y := headerHeight
	layout = append(layout, sectionItems(chrome.section, y)...)
	y += sectionHeight
	for index := pageIndex * drawerListRows; index < min(len(items), (pageIndex+1)*drawerListRows); index++ {
		layout = append(layout, drawerRowItems(items[index], y)...)
		y += drawerRowHeight
	}
	layout = append(layout, drawerFooterItems(chrome.hint, pageIndex+1, pages)...)
	return Page{ScreenID: "drawer-list", Items: finishPage(layout, nil), Index: pageIndex, Count: pages}
}

func runDrawerList(args []string, stdout, stderr io.Writer) int {
	flags, ok := parseDrawerFlags("drawer-list", args, stderr, 8)
	if !ok {
		return exitFailure
	}
	build := func() Page {
		return drawerListPage(sampleDrawerItems(flags.appsDir, flags.locale, flags.count, "icon.png"), chromeFor(flags.locale), 0)
	}
	return renderDrawerPage("drawer list", flags, build, stdout, stderr)
}
