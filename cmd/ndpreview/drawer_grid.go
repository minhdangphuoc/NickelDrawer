package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"io"
	"path/filepath"

	xdraw "golang.org/x/image/draw"

	"koboapps/internal/copperui"
)

// The drawer's grid view (docs/spec/ui-kit.md section 5.2). The numbers are CopperUI tokens, the
// same as mod/drawer/drawer_logic.h (GRID_*).
const (
	gridColumns      = copperui.DrawerGridColumns
	gridRowHeight    = copperui.DrawerGridRow
	gridIconSide     = copperui.IconGrid
	gridIconNameGap  = copperui.DrawerGridIconGap
	gridNameLines    = copperui.DrawerGridNameLines
	gridNameLineH    = copperui.TypeGridNameLine
	gridNameSize     = copperui.TypeGridNameSize
	gridNameInset    = copperui.DrawerGridNameInset
	gridHintHeight   = 48
	gridFooterHeight = copperui.DrawerFooterButton
	gridToggleSide   = copperui.HeaderIconTarget
)

// gridCell is the rectangle of cell i inside the content area, as drawer_logic.cc gridCellRect.
func gridCell(index int) image.Rectangle {
	cellWidth := contentWidth / gridColumns
	inset := (contentWidth - gridColumns*cellWidth) / 2
	column, row := index%gridColumns, index/gridColumns
	x := gutter + inset + column*cellWidth
	return image.Rect(x, headerHeight+row*gridRowHeight, x+cellWidth, headerHeight+(row+1)*gridRowHeight)
}

// gridRows is how many rows fit between the header and the hint and footer lines.
func gridRows() int {
	return max(1, (screenHeight-headerHeight-gridHintHeight-gridFooterHeight)/gridRowHeight)
}

type gridApp struct {
	name     string
	iconPath string
}

// gridNameLines wraps a name into at most gridNameLines lines; the last one is elided.
func gridNameText(name string) []string {
	width := contentWidth/gridColumns - 2*gridNameInset
	return wrapLimited(name, width, regular(gridNameSize, familySans), gridNameLines)
}

func gridCellItems(index int, app gridApp) []Item {
	cell := gridCell(index)
	groupHeight := gridIconSide + gridIconNameGap + gridNameLines*gridNameLineH
	iconTop := cell.Min.Y + (gridRowHeight-groupHeight)/2
	iconLeft := cell.Min.X + (cell.Dx()-gridIconSide)/2
	var items []Item
	if icon, ok := iconItem(app.iconPath, iconLeft, iconTop); ok {
		items = append(items, icon)
	}
	nameTop := iconTop + gridIconSide + gridIconNameGap
	for line, text := range gridNameText(app.name) {
		item := textItem(cell.Min.X+gridNameInset, nameTop+line*gridNameLineH, cell.Dx()-2*gridNameInset, gridNameLineH, text, gridNameSize, familySans, "center")
		item.Label = app.name
		items = append(items, item)
	}
	return items
}

// iconItem draws an app's icon.png over white: the files have a transparent background that a
// plain grey conversion would turn black.
func iconItem(path string, x, y int) (Item, bool) {
	source, loaded := loadPicture(path)
	if !loaded || source.Bounds().Empty() {
		return Item{}, false
	}
	scaled := image.NewGray(image.Rect(0, 0, gridIconSide, gridIconSide))
	draw.Draw(scaled, scaled.Bounds(), image.NewUniform(color.Gray{Y: paperFull}), image.Point{}, draw.Src)
	xdraw.ApproxBiLinear.Scale(scaled, scaled.Bounds(), source, source.Bounds(), xdraw.Over, nil)
	return Item{Shape: "image", X: x, Y: y, W: gridIconSide, H: gridIconSide, Fill: noFill, Picture: scaled}, true
}

func listToggleShapes(x, y int) []Item {
	var items []Item
	for row := 0; row < 3; row++ {
		items = append(items, solidRect(x+20, y+14+row*18, 44, 5, inkBlack))
	}
	return items
}

func gridToggleShapes(x, y int) []Item {
	var items []Item
	for column := 0; column < 2; column++ {
		for row := 0; row < 2; row++ {
			items = append(items, Item{Shape: "rect", X: x + 8 + column*30, Y: y + 8 + row*30, W: 24, H: 24, Fill: noFill, Border: 4, Ink: inkBlack})
		}
	}
	return items
}

// drawerGridHeader: back arrow, title, then refresh, list and grid icons at the right (the
// grid toggle underlined as the active view), the divider under the row.
func drawerGridHeader(title string) []Item {
	items := []Item{
		textItem(gutter, 0, headerIconWidth, headerRowHeight, "←", headerTitleSz+8, familySans, "center"),
		textItem(gutter+headerIconWidth+rhythm, 0, 600, headerRowHeight, title, headerTitleSz, familySans, "start"),
	}
	x := screenWidth - gutter - 3*gridToggleSide
	glyph := func(x int) (int, int) {
		return x + (gridToggleSide-headerIconGlyph)/2, (headerRowHeight - headerIconGlyph) / 2
	}
	gx, gy := glyph(x)
	items = append(items, iconShapes("refresh", gx, gy)...)
	gx, gy = glyph(x + gridToggleSide)
	items = append(items, listToggleShapes(gx, gy)...)
	gx, gy = glyph(x + 2*gridToggleSide)
	items = append(items, gridToggleShapes(gx, gy)...)
	items = append(items, solidRect(gx, gy+headerIconGlyph+6, headerIconGlyph, 4, inkBlack))
	return append(items, Item{Shape: "rect", X: gutter, Y: headerRowHeight, W: contentWidth, H: headerDivider, Fill: headerDividerInk})
}

// mutedCaption is the drawer's hint and pager look: caption size, muted grey.
func mutedCaption(item Item) Item {
	item.Ink = inkMuted
	return item
}

// drawerGridPage lays out the first page of the grid for apps.
func drawerGridPage(apps []gridApp, title, hint string) Page {
	items := drawerGridHeader(title)
	perPage := gridColumns * gridRows()
	for index := 0; index < len(apps) && index < perPage; index++ {
		items = append(items, gridCellItems(index, apps[index])...)
	}
	hintTop := screenHeight - gridFooterHeight - gridHintHeight
	items = append(items, mutedCaption(textItem(gutter, hintTop, contentWidth, gridHintHeight, hint, captionSize, familySans, "center")))
	pages := max(1, (len(apps)+perPage-1)/perPage)
	pager := fmt.Sprintf("‹   1 / %d   ›", pages)
	items = append(items, mutedCaption(textItem(gutter, screenHeight-gridFooterHeight, contentWidth, gridFooterHeight, pager, captionSize, familySans, "center")))
	return Page{ScreenID: "drawer-grid", Items: items, Index: 0, Count: pages}
}

var sampleGridNames = map[string][]string{
	"en": {"LocalSend", "GitHub Sync", "Library", "Wallpaper", "Calendar sync for the reading list", "Notes", "Dictionary", "Clock"},
	"vi": {"LocalSend", "Đồng bộ GitHub", "Thư viện", "Hình nền", "Đồng bộ lịch cho danh sách đọc sách", "Ghi chú", "Từ điển", "Đồng hồ"},
}

var gridChrome = map[string][2]string{
	"en": {"Apps", "Long-press an app for info or remove"},
	"vi": {"Ứng dụng", "Nhấn giữ một ứng dụng để xem thông tin hoặc gỡ"},
}

// sampleGridApps names the sample apps and gives each an icon.png of the apps folder, in turn.
func sampleGridApps(appsDir, locale string, count int) []gridApp {
	names := sampleGridNames[locale]
	if names == nil {
		names = sampleGridNames["en"]
	}
	icons, _ := filepath.Glob(filepath.Join(appsDir, "*", "icon.png"))
	var apps []gridApp
	for index := 0; index < count; index++ {
		app := gridApp{name: names[index%len(names)]}
		if len(icons) > 0 {
			app.iconPath = icons[index%len(icons)]
		}
		apps = append(apps, app)
	}
	return apps
}

func runDrawerGrid(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("drawer-grid", flag.ContinueOnError)
	flags.SetOutput(stderr)
	outPath := flags.String("out", "", "PNG file to write (required)")
	appsDir := flags.String("apps", "apps", "folder whose apps' icon.png fill the sample cells")
	locale := flags.String("locale", "en", "language of the preview (en or vi)")
	count := flags.Int("count", 7, "number of sample apps")
	fontPath := flags.String("font", "", "TrueType or OpenType file for text (default: a system font if found)")
	if _, err := parseInterspersed(flags, args); err != nil {
		return exitFailure
	}
	if *outPath == "" {
		fmt.Fprintln(stderr, "ndpreview drawer-grid needs --out FILE")
		return exitFailure
	}
	chrome, known := gridChrome[*locale]
	if !known {
		chrome = gridChrome["en"]
	}
	fonts, err := loadFonts(*fontPath, "", "")
	if err != nil {
		fmt.Fprintln(stderr, "ndpreview:", err)
		return exitFailure
	}
	useFonts(fonts, stderr)
	page := drawerGridPage(sampleGridApps(*appsDir, *locale, *count), chrome[0], chrome[1])
	canvas, err := renderPNG(page, fonts, greyLevelsOff)
	if err == nil {
		err = saveCanvas(*outPath, canvas)
	}
	if err != nil {
		fmt.Fprintln(stderr, "ndpreview:", err)
		return exitFailure
	}
	fmt.Fprintf(stdout, "wrote %s (%dx%d, drawer grid, %d apps)\n", *outPath, screenWidth, screenHeight, *count)
	return exitOK
}
