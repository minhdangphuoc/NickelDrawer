package main

import (
	"image"
	"image/color"
	"image/draw"
	"io"

	xdraw "golang.org/x/image/draw"

	"koboapps/internal/copperui"
)

// The drawer's grid view (docs/spec/ui-kit.md section 5.2). The numbers are CopperUI tokens, the
// same as mod/drawer/drawer_logic.h (GRID_*). Geometry of the page: drawer_view.go.
const (
	gridColumns     = copperui.DrawerGridColumns
	gridRowHeight   = copperui.DrawerGridRow
	gridIconSide    = copperui.IconGrid
	gridIconNameGap = copperui.DrawerGridIconGap
	gridNameLines   = copperui.DrawerGridNameLines
	gridNameLineH   = copperui.TypeGridNameLine
	gridNameSize    = copperui.TypeGridNameSize
	gridNameInset   = copperui.DrawerGridNameInset
)

// gridCell is the rectangle of cell i inside the content area, as drawer_logic.cc gridCellRect.
func gridCell(index int) image.Rectangle {
	cellWidth := contentWidth / gridColumns
	inset := (contentWidth - gridColumns*cellWidth) / 2
	column, row := index%gridColumns, index/gridColumns
	x := gutter + inset + column*cellWidth
	return image.Rect(x, headerHeight+row*gridRowHeight, x+cellWidth, headerHeight+(row+1)*gridRowHeight)
}

// gridRows is how many rows fit in the page area (DrawerLogic::gridRowsThatFit).
func gridRows() int {
	return max(1, drawerPageArea/gridRowHeight)
}

// gridNameText wraps a name into at most gridNameLines lines; the last one is elided.
func gridNameText(name string) []string {
	width := contentWidth/gridColumns - 2*gridNameInset
	return wrapLimited(name, width, regular(gridNameSize, familySans), gridNameLines)
}

func gridCellItems(index int, app drawerItem) []Item {
	cell := gridCell(index)
	groupHeight := gridIconSide + gridIconNameGap + gridNameLines*gridNameLineH
	iconTop := cell.Min.Y + (gridRowHeight-groupHeight)/2
	iconLeft := cell.Min.X + (cell.Dx()-gridIconSide)/2
	var items []Item
	if icon, ok := appIconItem(app.iconPath, gridIconSide, iconLeft, iconTop); ok {
		items = append(items, icon)
	}
	if app.hasUpdate {
		items = append(items, Item{Shape: "ellipse", X: iconLeft + gridIconSide - updateDotSide, Y: iconTop, W: updateDotSide, H: updateDotSide, Fill: inkBlack, Ink: inkBlack, Label: "update"})
	}
	nameTop := iconTop + gridIconSide + gridIconNameGap
	for line, text := range gridNameText(app.name) {
		item := textItem(cell.Min.X+gridNameInset, nameTop+line*gridNameLineH, cell.Dx()-2*gridNameInset, gridNameLineH, text, gridNameSize, familySans, "center")
		item.Label = app.name
		items = append(items, item)
	}
	return items
}

// appIconItem draws an app's icon PNG over white at side x side: the files have a transparent
// background that a plain grey conversion would turn black. A file of another size is scaled.
func appIconItem(path string, side, x, y int) (Item, bool) {
	source, loaded := loadPicture(path)
	if !loaded || source.Bounds().Empty() {
		return Item{}, false
	}
	scaled := image.NewGray(image.Rect(0, 0, side, side))
	draw.Draw(scaled, scaled.Bounds(), image.NewUniform(color.Gray{Y: paperFull}), image.Point{}, draw.Src)
	xdraw.ApproxBiLinear.Scale(scaled, scaled.Bounds(), source, source.Bounds(), xdraw.Over, nil)
	return Item{Shape: "image", X: x, Y: y, W: side, H: side, Fill: noFill, Picture: scaled}, true
}

var sampleGridNames = map[string][]string{
	"en": {"LocalSend", "GitHub Sync", "Library", "Wallpaper", "Calendar sync for the reading list", "Notes", "Dictionary", "Clock"},
	"vi": {"LocalSend", "Đồng bộ GitHub", "Thư viện", "Hình nền", "Đồng bộ lịch cho danh sách đọc sách", "Ghi chú", "Từ điển", "Đồng hồ"},
}

// drawerGridPage lays out page pageIndex of the grid for apps: the ⚙ gear carries the update dot,
// the pager shows only with more than one page.
func drawerGridPage(apps []drawerItem, chrome drawerChrome, pageIndex int) Page {
	perPage := gridColumns * gridRows()
	pages := max(1, (len(apps)+perPage-1)/perPage)
	pageIndex = clamp(pageIndex, 0, pages-1)
	items := drawerHeaderItems(chrome.title, true, true)
	for index := pageIndex * perPage; index < min(len(apps), (pageIndex+1)*perPage); index++ {
		items = append(items, gridCellItems(index-pageIndex*perPage, apps[index])...)
	}
	items = append(items, drawerFooterItems(chrome.hint, pageIndex+1, pages)...)
	return Page{ScreenID: "drawer-grid", Items: finishPage(items, nil), Index: pageIndex, Count: pages}
}

func runDrawerGrid(args []string, stdout, stderr io.Writer) int {
	flags, ok := parseDrawerFlags("drawer-grid", args, stderr, 7)
	if !ok {
		return exitFailure
	}
	build := func() Page {
		return drawerGridPage(sampleDrawerItems(flags.appsDir, flags.locale, flags.count, "icon.png"), chromeFor(flags.locale), 0)
	}
	return renderDrawerPage("drawer grid", flags, build, stdout, stderr)
}
