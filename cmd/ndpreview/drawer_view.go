package main

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"slices"

	"koboapps/internal/copperui"
)

// The drawer page the renderers below draw (mod/drawer/drawer_view.cc). Like an app screen it is
// laid out in the page area below Nickel's status bar (viewHeight = 1426 px) and shifted by
// finishPage: header (121), page area, hint line, footer (120).
//
// The page area is what is left: viewHeight - header - hint - footer = 1139 px with the 46 px
// caption line height. The device logged "page area 1152 px" on an older build whose hint line
// was 33 px high; both give 4 grid rows and 7 list rows plus a section label.
const (
	drawerFooterHeight = copperui.DrawerFooterButton
	drawerHintHeight   = captionLineH
	drawerListRows     = copperui.DrawerListRows
	drawerRowHeight    = copperui.RowDrawer
	drawerPageArea     = viewHeight - headerHeight - drawerHintHeight - drawerFooterHeight
)

// drawerItem is one sample app of the drawer.
type drawerItem struct {
	name, status, iconPath string
	hasUpdate, hasAction   bool
}

// drawerChrome are the drawer's own texts (host/locales mod.* keys) for the previews.
type drawerChrome struct {
	title, hint, section, tweaksTitle string
	loading, emptyTitle, emptyHint    string
	findApps, problemsSection         string
	qt6Notice                         string
	problemName, problemStatus        string
	statuses                          []string
}

var drawerChromes = map[string]drawerChrome{
	"en": {title: "Apps", hint: "Long-press an app for info or remove", section: "Installed", tweaksTitle: "Tweaks",
		loading: "Loading…", emptyTitle: "No apps yet", emptyHint: "Copy an app folder to .adds/apps, then tap Find apps",
		findApps: "Find apps", problemsSection: "Problems", qt6Notice: "Kobo is offering update 5.x — NickelDrawer will stop working",
		problemName: "broken-app", problemStatus: "Cannot open the app: ui.json is not valid JSON at line 12",
		statuses: []string{"Receiving", "Last sync 14:20", "", "1 update", "3 tasks", ""}},
	"vi": {title: "Ứng dụng", hint: "Nhấn giữ một ứng dụng để xem thông tin hoặc gỡ", section: "Đã cài", tweaksTitle: "Tinh chỉnh",
		loading: "Đang tải…", emptyTitle: "Chưa có ứng dụng", emptyHint: "Chép thư mục ứng dụng vào .adds/apps rồi chạm Tìm ứng dụng",
		findApps: "Tìm ứng dụng", problemsSection: "Sự cố", qt6Notice: "Kobo đang mời cập nhật 5.x — NickelDrawer sẽ ngừng hoạt động",
		problemName: "ung-dung-loi", problemStatus: "Không mở được ứng dụng: ui.json không phải JSON hợp lệ ở dòng 12",
		statuses: []string{"Đang nhận", "Đồng bộ lần cuối 14:20", "", "1 bản cập nhật", "3 việc", ""}},
}

func chromeFor(locale string) drawerChrome {
	if chrome, known := drawerChromes[locale]; known {
		return chrome
	}
	return drawerChromes["en"]
}

// drawerHeaderItems is Nickel's header (back arrow, title) with the host's icon buttons at the
// right in DrawerLogic::headerButtonAt order: refresh, list, grid, settings. The active view's
// toggle is underlined; hasUpdate puts the black dot on the gear.
func drawerHeaderItems(title string, isGrid, hasUpdate bool) []Item {
	items := []Item{{Shape: "rect", X: gutter, Y: 0, W: headerIconWidth, H: headerRowHeight, Fill: noFill, Label: "back"}}
	items = append(items, iconCentred("back", headerIconWidth, gutter, 0, headerIconWidth, headerRowHeight, inkBlack)...)
	titleX := gutter + headerIconWidth + rhythm
	items = append(items, textItem(titleX, 0, 600, headerRowHeight, elide(title, 600, regular(headerTitleSz, familySans), false), headerTitleSz, familySans, "start"))
	x := screenWidth - gutter - 4*headerIconTarget
	for position, name := range []string{"refresh", "list", "grid", "settings"} {
		boxX := x + position*headerIconTarget
		items = append(items, iconCentred(name, copperui.IconToolbar, boxX, 0, headerIconTarget, headerRowHeight, inkBlack)...)
		isActive := (name == "grid" && isGrid) || (name == "list" && !isGrid)
		if isActive {
			iconX, iconY := boxX+(headerIconTarget-copperui.IconToolbar)/2, (headerRowHeight-copperui.IconToolbar)/2
			items = append(items, solidRect(iconX, iconY+copperui.IconToolbar+copperui.UnderlineGap, copperui.IconToolbar, copperui.UnderlineHeight, inkBlack))
		}
		if name == "settings" && hasUpdate {
			items = append(items, Item{Shape: "ellipse", X: boxX + headerIconTarget - updateDotSide - hostDotInset, Y: hostDotInset, W: updateDotSide, H: updateDotSide, Fill: inkBlack, Ink: inkBlack, Label: "update"})
		}
	}
	return append(items, Item{Shape: "rect", X: gutter, Y: headerRowHeight, W: contentWidth, H: headerDivider, Fill: headerDividerInk})
}

func solidRect(x, y, w, h, level int) Item {
	return Item{Shape: "rect", X: x, Y: y, W: w, H: h, Fill: level}
}

// drawerFooterItems is the hint line (caption, muted, centred) and, only when there is more than
// one page, the pager in the 120 px footer.
func drawerFooterItems(hint string, page, pages int) []Item {
	hintTop := viewHeight - drawerFooterHeight - drawerHintHeight
	item := textItem(gutter, hintTop, contentWidth, drawerHintHeight, elide(hint, contentWidth, regular(captionSize, familySans), false), captionSize, familySans, "center")
	item.Ink = inkMuted
	items := []Item{item}
	if pages > 1 {
		items = append(items, pageBar(viewHeight-drawerFooterHeight+(drawerFooterHeight-pageBarHeight)/2, page, pages)...)
	}
	return items
}

// sampleDrawerItems names the sample apps and gives each an icon (apps/<app>/icon-64.png in the
// list, icon.png in the grid) of the apps folder, in turn.
func sampleDrawerItems(appsDir, locale string, count int, fileName string) []drawerItem {
	names := sampleGridNames[locale]
	if names == nil {
		names = sampleGridNames["en"]
	}
	chrome := chromeFor(locale)
	icons := sampleIcons(appsDir, fileName)
	var items []drawerItem
	for index := 0; index < count; index++ {
		item := drawerItem{name: names[index%len(names)], hasAction: true, status: chrome.statuses[index%len(chrome.statuses)]}
		item.hasUpdate = item.status != "" && index%len(chrome.statuses) == 3
		if len(icons) > 0 {
			item.iconPath = icons[index%len(icons)]
		}
		items = append(items, item)
	}
	return items
}

// drawerFlags are the options every drawer-* command shares.
type drawerFlags struct {
	out, appsDir, locale, font, icons string
	count                             int
	loading, problems, qt6Notice      bool
	installed                         bool
}

func parseDrawerFlags(name string, args []string, stderr io.Writer, defaultCount int) (drawerFlags, bool) {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(stderr)
	var parsed drawerFlags
	flags.StringVar(&parsed.out, "out", "", "PNG file to write (required)")
	flags.StringVar(&parsed.appsDir, "apps", "apps", "folder whose apps' icons fill the sample items")
	flags.StringVar(&parsed.locale, "locale", "en", "language of the preview (en or vi)")
	flags.IntVar(&parsed.count, "count", defaultCount, "number of sample items")
	flags.BoolVar(&parsed.loading, "loading", false, "drawer-empty: the Loading state (heading alone) instead of the empty state")
	flags.BoolVar(&parsed.problems, "problems", false, "drawer-list: add a Problems section with one problem row")
	flags.BoolVar(&parsed.installed, "installed", false, "drawer-list, drawer-grid: the visible apps of --apps with their own names, not the samples")
	flags.BoolVar(&parsed.qt6Notice, "qt6", false, "drawer-list: show the Qt6 upgrade warning notice under the header")
	flags.StringVar(&parsed.font, "font", "", "TrueType or OpenType file for text (default: a system font if found)")
	flags.StringVar(&parsed.icons, "icons", "", "folder with the Tabler PNGs (default: host/icons/ui of the repository)")
	if _, err := parseInterspersed(flags, args); err != nil {
		return parsed, false
	}
	if parsed.out == "" {
		fmt.Fprintf(stderr, "ndpreview %s needs --out FILE\n", name)
		return parsed, false
	}
	return parsed, true
}

// renderDrawerPage draws a page built by build with the flags' fonts and icons and writes the PNG.
func renderDrawerPage(name string, flags drawerFlags, build func() Page, stdout, stderr io.Writer) int {
	setIconDir(flags.icons)
	fonts, err := loadFonts(flags.font, "", "")
	if err != nil {
		fmt.Fprintln(stderr, "ndpreview:", err)
		return exitFailure
	}
	useFonts(fonts, stderr)
	canvas, err := renderPNG(build(), fonts, greyLevelsOff)
	if err == nil {
		err = saveCanvas(flags.out, canvas)
	}
	if err != nil {
		fmt.Fprintln(stderr, "ndpreview:", err)
		return exitFailure
	}
	fmt.Fprintf(stdout, "wrote %s (%dx%d, %s)\n", flags.out, screenWidth, screenHeight, name)
	return exitOK
}

// sampleIconOrder puts the bundled apps in the order of the sample names (LocalSend, GitHub Sync,
// Library, Wallpaper, ...); other app folders follow alphabetically.
var sampleIconOrder = []string{"localsend", "ghsync", "library", "wallpaper", "todo", "uikit"}

func sampleIcons(appsDir, fileName string) []string {
	found, _ := filepath.Glob(filepath.Join(appsDir, "*", fileName))
	var ordered []string
	for _, id := range sampleIconOrder {
		for _, path := range found {
			if filepath.Base(filepath.Dir(path)) == id {
				ordered = append(ordered, path)
			}
		}
	}
	for _, path := range found {
		if !slices.Contains(ordered, path) {
			ordered = append(ordered, path)
		}
	}
	return ordered
}
