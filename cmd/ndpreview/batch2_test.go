package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"koboapps/internal/uikit"
)

// layoutBody lays out a one-screen app whose ui.json has the given body and footer components.
func layoutBody(t *testing.T, vars, body, footer string) Page {
	t.Helper()
	appDir := newAppDir(t, "probe", "echo hi\n")
	writeFile(t, filepath.Join(appDir, "ui.json"), `{"kit":1,"start":"main","vars":`+vars+`,
	"screens":{"main":{"title":"Probe","body":[`+body+`],"footer":[`+footer+`]}}}`)
	loaded, err := loadApp(appDir, "en")
	if err != nil {
		t.Fatal(err)
	}
	rc := renderContext{doc: loaded.doc, locale: "en", vars: loaded.doc.Vars}
	page, err := layoutScreen(rc, "main", 0, Overlay{})
	if err != nil {
		t.Fatal(err)
	}
	return page
}

func textItems(page Page, text string) []Item {
	var found []Item
	for _, item := range page.Items {
		if item.Shape == "text" && strings.Join(item.Lines, "\n") == text {
			found = append(found, item)
		}
	}
	return found
}

func onlyText(t *testing.T, page Page, text string) Item {
	t.Helper()
	found := textItems(page, text)
	if len(found) != 1 {
		t.Fatalf("want one text item %q, got %d", text, len(found))
	}
	return found[0]
}

// ---- 2.7 keep-with-next: the shared vectors ----

func TestPackPagesMatchesTheSharedVectors(t *testing.T) {
	data, err := os.ReadFile("../../lib/copperui/tests/pack-vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var file struct {
		Cases []struct {
			Name      string   `json:"name"`
			Available int      `json:"available"`
			Blocks    [][3]int `json:"blocks"`
			Pages     []int    `json:"pages"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		t.Fatal(err)
	}
	if len(file.Cases) == 0 {
		t.Fatal("no pack vectors")
	}
	for _, vector := range file.Cases {
		blocks := make([]block, len(vector.Blocks))
		for index, spec := range vector.Blocks {
			blocks[index] = block{height: spec[0], gap: spec[1], keepWithNext: spec[2] != 0}
		}
		got := packPageIndexes(blocks, vector.Available)
		if fmt.Sprint(got) != fmt.Sprint(vector.Pages) && !(len(got) == 0 && len(vector.Pages) == 0) {
			t.Errorf("%s: pages %v, want %v", vector.Name, got, vector.Pages)
		}
		wantCount := 1
		for _, page := range vector.Pages {
			wantCount = max(wantCount, page+1)
		}
		if count := len(packPages(blocks, vector.Available)); count != wantCount {
			t.Errorf("%s: %d page(s), want %d", vector.Name, count, wantCount)
		}
	}
}

func TestGroupsKeepWithNext(t *testing.T) {
	rc := renderContext{locale: "en", vars: map[string]any{"mode": "a"}}
	check := func(name string, blocks []block, keeps []bool) {
		t.Helper()
		if len(blocks) != len(keeps) {
			t.Fatalf("%s: %d blocks, want %d", name, len(blocks), len(keeps))
		}
		for index, keep := range keeps {
			if blocks[index].keepWithNext != keep {
				t.Errorf("%s: block %d keepWithNext = %v, want %v", name, index, blocks[index].keepWithNext, keep)
			}
		}
	}
	check("section", []block{rc.sectionBlock("Group")}, []bool{true})
	check("qr with address", rc.qrBlocks("https://x.org", "Scan"), []bool{true, false, false})
	check("qr without address", rc.qrBlocks("", ""), []bool{false})
}

func TestSectionLabelIsNeverAloneAtTheBottomOfAPage(t *testing.T) {
	// 10 rows of 104 px leave room for a section label (64) at the bottom of page 1, but not for
	// the label and its first row (page body 1129 px with the pager).
	var rows []string
	for i := 0; i < 10; i++ {
		rows = append(rows, fmt.Sprintf(`["Row %d","x"]`, i))
	}
	body := `{"list":"$files"},{"section":"Next group"},{"kv":[["A","1"],["B","2"],["C","3"],["D","4"],["E","5"]]}`
	page := layoutBody(t, `{"files":[`+strings.Join(rows, ",")+`]}`, body, ``)
	if page.Count < 2 {
		t.Fatalf("want a second page, got %d", page.Count)
	}
	if len(textItems(page, "NEXT GROUP")) != 0 {
		t.Error("the section label must move to page 2 with its row, not stay alone at the bottom of page 1")
	}
}

// ---- 2.12 page area ----

func TestScreenIsLaidOutInThePageAreaBelowTheStatusBar(t *testing.T) {
	page := layoutBody(t, `{}`, `{"text":"Hello"}`, `{"button":"Go","action":"go","style":"primary"}`)
	button := firstItem(page, func(item Item) bool { return item.Shape == "rect" && item.Label == "Go" })
	if button.Y+button.H != insetStatusBar+viewHeight-bottomMargin {
		t.Errorf("footer button ends at %d, want %d (view 1554 less the 72 px margin)", button.Y+button.H, insetStatusBar+viewHeight-bottomMargin)
	}
	if viewHeight != screenHeight-tabBarHeight-insetStatusBar {
		t.Errorf("page area %d, want %d", viewHeight, screenHeight-tabBarHeight-insetStatusBar)
	}
	chrome := countItems(page, func(item Item) bool {
		return item.Shape == "text" && (item.Lines[0] == "Nickel status bar" || item.Lines[0] == "Nickel tab bar")
	})
	if chrome != 2 {
		t.Errorf("want the status-bar and tab-bar strips, got %d captions", chrome)
	}
	header := firstItem(page, func(item Item) bool { return item.Shape == "text" && item.Lines[0] == "Probe" })
	if header.Y != insetStatusBar {
		t.Errorf("the header row starts at %d, want the inset %d", header.Y, insetStatusBar)
	}
}

func TestClickTargetsKeepTheirShift(t *testing.T) {
	page := layoutBody(t, `{}`, `{"row":"Open","action":"go"}`, ``)
	row := firstItem(page, func(item Item) bool { return item.Event != "" })
	if row.Y != insetStatusBar+headerHeight+rhythm {
		t.Errorf("row target at %d, want %d", row.Y, insetStatusBar+headerHeight+rhythm)
	}
}

// ---- 2.2 / 2.10 / 2.11 type ----

func TestHeadingIsSerifBold52Over74(t *testing.T) {
	page := layoutBody(t, `{}`, `{"text":"Title","style":"heading"},{"empty":"Nothing here","text":"Add one"}`, ``)
	for _, text := range []string{"Title", "Nothing here"} {
		heading := onlyText(t, page, text)
		if heading.Family != familySerif || heading.Size != 52 || heading.LineH != 74 || !heading.Bold {
			t.Errorf("%q: %+v, want serif bold 52/74", text, heading)
		}
	}
}

func TestBigCodeIsRegularWeight(t *testing.T) {
	page := layoutBody(t, `{"pin":"4821"}`, `{"bigcode":"$pin"}`, ``)
	if code := onlyText(t, page, "4821"); code.Bold || code.Family != familyMono || code.Size != 144 {
		t.Errorf("big code %+v, want regular mono 144", code)
	}
}

func TestRadioTitleIsACaptionNotASectionLabel(t *testing.T) {
	body := `{"radio":"Refresh mode","bind":"mode","options":[["a","Fast"],["b","Balanced"],["c","Quality"]]}`
	page := layoutBody(t, `{"mode":"b"}`, body, ``)
	title := onlyText(t, page, "Refresh mode")
	if title.Size != captionSize || title.Ink != inkMuted || title.Align != "start" {
		t.Errorf("title %+v, want a muted 32 px caption", title)
	}
	if len(textItems(page, "REFRESH MODE")) != 0 {
		t.Error("the title must not be drawn as an uppercase section label")
	}
	for _, item := range page.Items {
		if item.Shape == "rect" && item.Fill == lineSection && item.H == separatorPx {
			t.Error("a radio has no black section rule")
		}
	}
	rc := renderContext{doc: mustLoadSingle(t, body), locale: "en", vars: map[string]any{"mode": "b"}}
	blocks := rc.blocksFor(rc.doc.Screens["main"].Body)
	if len(blocks) != 4 || !blocks[0].keepWithNext || !blocks[1].keepWithNext || !blocks[2].keepWithNext || blocks[3].keepWithNext {
		t.Errorf("title and options must keep together, the last option ends the group: %+v", blocks)
	}
	if blocks[0].gap != fieldCaptionGap {
		t.Errorf("title gap %d, want %d", blocks[0].gap, fieldCaptionGap)
	}
}

func mustLoadSingle(t *testing.T, body string) *uikit.Document {
	t.Helper()
	appDir := newAppDir(t, "probe", "echo hi\n")
	writeFile(t, filepath.Join(appDir, "ui.json"), `{"kit":1,"start":"main","vars":{"mode":"b"},"screens":{"main":{"title":"Probe","body":[`+body+`]}}}`)
	loaded, err := loadApp(appDir, "en")
	if err != nil {
		t.Fatal(err)
	}
	return loaded.doc
}

// ---- 2.4 disabled looks ----

func TestDisabledToggleRadioAndFieldAreDimmedWithoutEvents(t *testing.T) {
	body := `{"toggle":"Auto","bind":"auto","enabled":false,"detail":"Needs Wi-Fi"},
	{"radio":"Mode","bind":"mode","enabled":false,"options":[["a","Fast"],["b","Balanced","slow"]]},
	{"field":"Name","bind":"name","enabled":false}`
	page := layoutBody(t, `{"auto":true,"mode":"b","name":"Kobo"}`, body, ``)
	for _, text := range []string{"Auto", "ON", "Fast", "Balanced", "Kobo"} {
		if item := onlyText(t, page, text); item.Ink != inkDim {
			t.Errorf("%q ink %d, want dim %d", text, item.Ink, inkDim)
		}
	}
	if detail := onlyText(t, page, "Needs Wi-Fi"); detail.Ink != inkDim {
		t.Errorf("toggle detail ink %d, want dim", detail.Ink)
	}
	track := firstItem(page, func(item Item) bool {
		return item.Shape == "rect" && item.W == toggleSwitchWidth && item.H == toggleSwitchHeight
	})
	if track.Ink != inkDim || track.Fill != inkDim {
		t.Errorf("switch track %+v, want drawn at dim ink", track)
	}
	for _, item := range page.Items {
		if item.Event != "" {
			t.Errorf("disabled control has an event: %+v", item)
		}
	}
	field := firstItem(page, func(item Item) bool { return item.Shape == "rect" && item.W == contentWidth && item.H == fieldHeight })
	if field.Ink != inkDim || field.Border != fieldBorder {
		t.Errorf("field box %+v, want a dim 2 px outline", field)
	}
	ring := firstItem(page, func(item Item) bool { return item.Shape == "ellipse" && item.W == radioRingSide })
	if ring.Ink != inkDim {
		t.Errorf("radio ring ink %d, want dim", ring.Ink)
	}
}

func TestEnabledControlsStayBlackAndFieldEnabledFollowsItsFlag(t *testing.T) {
	body := `{"toggle":"Auto","bind":"auto"},{"field":"Name","bind":"name","enabled":"$auto"},{"field":"Other","bind":"other"}`
	page := layoutBody(t, `{"auto":true,"name":"Kobo","other":"x"}`, body, ``)
	if item := onlyText(t, page, "Auto"); item.Ink != inkBlack {
		t.Errorf("enabled toggle label ink %d", item.Ink)
	}
	if item := onlyText(t, page, "Kobo"); item.Ink != inkBlack {
		t.Errorf("a field enabled by $auto must be black, got %d", item.Ink)
	}
	off := layoutBody(t, `{"auto":false,"name":"Kobo","other":"x"}`, body, ``)
	if item := onlyText(t, off, "Kobo"); item.Ink != inkDim {
		t.Errorf("a field disabled by $auto must be dim, got %d", item.Ink)
	}
	if item := onlyText(t, off, "x"); item.Ink != inkBlack {
		t.Errorf("a field without enabled stays enabled, got ink %d", item.Ink)
	}
}

// ---- 2.5 list row ----

func rowItems(t *testing.T, primary, secondary string) []Item {
	t.Helper()
	return listRowBlock(primary, secondary, "e", "tap", false).draw(0)
}

func TestListRowPrimaryYieldsToTheSecondaryCaption(t *testing.T) {
	items := rowItems(t, "Dune.epub", "Book")
	var primary, secondary Item
	for _, item := range items {
		if item.Shape == "text" && item.Size == rowTextSize {
			primary = item
		}
		if item.Shape == "text" && item.Size == captionSize {
			secondary = item
		}
	}
	if secondary.Ink != inkMuted || secondary.Align != "end" || secondary.X+secondary.W != gutter+contentWidth {
		t.Errorf("secondary %+v, want a muted caption at the right edge", secondary)
	}
	if primary.W != contentWidth-secondary.W-rhythm || primary.Bold || primary.Ink != inkBlack {
		t.Errorf("primary %+v, want regular black and %d px wide", primary, contentWidth-secondary.W-rhythm)
	}
	if len(items) < 3 || items[len(items)-1].Event != "e" || items[len(items)-1].H != rowHeight {
		t.Error("the row must keep its 104 px tap target")
	}
}

func TestListRowWithoutSecondaryUsesTheWholeWidthAndMiddleElidesFileNames(t *testing.T) {
	long := strings.Repeat("a", 90) + ".epub"
	items := rowItems(t, long, "")
	primary := items[0]
	if primary.W != contentWidth {
		t.Errorf("primary width %d, want the whole row %d", primary.W, contentWidth)
	}
	shown := primary.Lines[0]
	if !strings.Contains(shown, ellipsis) || !strings.HasSuffix(shown, ".epub") {
		t.Errorf("file name %q must be elided in the middle and keep its extension", shown)
	}
}

func TestListRowSecondaryIsCapped(t *testing.T) {
	items := rowItems(t, "x", strings.Repeat("very long secondary ", 10))
	for _, item := range items {
		if item.Shape == "text" && item.Size == captionSize && item.W > listSecondaryMax {
			t.Errorf("secondary %d px wide, cap %d", item.W, listSecondaryMax)
		}
	}
}

func TestListRowCheckMarkIsTheTablerCheckIcon(t *testing.T) {
	items := rowItems(t, "Return books", checkMark)
	var icon Item
	for _, item := range items {
		if item.Shape == "icon" {
			icon = item
		}
		if item.Shape == "text" && item.Size == captionSize {
			t.Error("the check mark must not be drawn as a glyph")
		}
	}
	if icon.Label != "check" || icon.W != chevronIconSize || icon.X+icon.W != gutter+contentWidth {
		t.Errorf("icon %+v, want check-64 at the right edge", icon)
	}
	if items[0].W != contentWidth-chevronIconSize-rhythm {
		t.Errorf("primary width %d, want %d", items[0].W, contentWidth-chevronIconSize-rhythm)
	}
}

func TestListRowsAreNotKeyValueRows(t *testing.T) {
	page := layoutBody(t, `{"files":[["A long primary text that needs more than the key column of 440 px width to show","x"]]}`, `{"list":"$files"}`, ``)
	for _, item := range page.Items {
		if item.Shape == "text" && item.Size == rowTextSize && item.W == keyColumn {
			t.Error("a list row must not use the 440 px key column")
		}
	}
}

// ---- 2.6 pager ----

func TestPagerHasChevronIconsAndIsHiddenOnASinglePage(t *testing.T) {
	short := layoutBody(t, `{}`, `{"text":"Hello"}`, ``)
	if short.Count != 1 || countItems(short, func(item Item) bool { return item.Shape == "icon" && strings.HasPrefix(item.Label, "chevron") }) != 0 {
		t.Error("one page must show no pager")
	}
	var rows []string
	for i := 0; i < 20; i++ {
		rows = append(rows, fmt.Sprintf(`["Row %d","x"]`, i))
	}
	long := layoutBody(t, `{"files":[`+strings.Join(rows, ",")+`]}`, `{"list":"$files"}`, ``)
	if long.Count < 2 {
		t.Fatal("want several pages")
	}
	label := onlyText(t, long, fmt.Sprintf("1 / %d", long.Count))
	left := firstItem(long, func(item Item) bool { return item.Label == "chevron-left" })
	right := firstItem(long, func(item Item) bool { return item.Label == "chevron-right" })
	if label.W != pagerLabelWidth || label.Ink != inkMuted || label.Align != "center" {
		t.Errorf("label %+v", label)
	}
	if left.W != chevronIconSize || left.X+left.W+pagerArrowGap != label.X || right.X != label.X+label.W+pagerArrowGap {
		t.Errorf("arrows %+v %+v must flank the label %+v at %d px gaps", left, right, label, pagerArrowGap)
	}
	if left.Y != label.Y+(pageBarHeight-chevronIconSize)/2 {
		t.Errorf("arrow y %d, want centred on the pager line", left.Y)
	}
}

// ---- 2.8 dialogs and menus ----

func TestDialogUsesTheDialogTokensWithCancelLeft(t *testing.T) {
	rc := renderContext{locale: "en"}
	items := rc.dialogItems(DialogSpec{Title: "Remove", Text: "Remove this picture?", Ok: "Remove", Cancel: "Cancel", OkEvent: "x"})
	box := items[0]
	if box.W != dialogWidth || box.Border != 3 || dialogWidth != 1040 {
		t.Errorf("box %+v, want 1040 wide with a 3 px border", box)
	}
	var titleItem, textItem, cancel, confirm Item
	for _, item := range items {
		switch {
		case item.Shape == "text" && item.Family == familySerif:
			titleItem = item
		case item.Shape == "text" && item.Size == rowTextSize && !item.Bold:
			textItem = item
		case item.Shape == "rect" && item.Label == "Cancel":
			cancel = item
		case item.Shape == "rect" && item.Label == "Remove" && item.Event != "":
			confirm = item
		}
	}
	if titleItem.Size != 42 || titleItem.LineH != 60 || !titleItem.Bold {
		t.Errorf("title %+v, want serif bold 42/60", titleItem)
	}
	if textItem.Size != 36 || textItem.X != box.X+dialogPadding {
		t.Errorf("text %+v, want 36 px at the 48 px padding", textItem)
	}
	if cancel.X >= confirm.X || cancel.H != 120 || confirm.H != 120 || cancel.Border != 3 {
		t.Errorf("cancel %+v confirm %+v: want cancel left, 120 px tall, 3 px border", cancel, confirm)
	}
}

func TestMenuUsesNickelsMetrics(t *testing.T) {
	items := menuItems("Dune.epub", []string{"Info", "Remove…"}, 632, 800)
	box := items[0]
	if box.W != 760 || box.Border != 3 || box.H != 2*3+120*3+2 {
		t.Errorf("box %+v, want 760 wide, 3 px border, a title row and two 120 px rows with 1 px dividers", box)
	}
	dividers := 0
	for _, item := range items {
		if item.Shape == "rect" && item.H == 1 && item.Fill == lineRow {
			dividers++
		}
		if item.Shape == "text" && item.Size != 34 {
			t.Errorf("menu text %+v, want 34 px (A11)", item)
		}
	}
	if dividers != 2 {
		t.Errorf("%d dividers, want 2", dividers)
	}
	if title := items[1]; !title.Bold {
		t.Errorf("the title row is bold: %+v", title)
	}
}

// ---- 2.9 icons ----

func TestMissingIconFolderDrawsNothingAndWarnsOnce(t *testing.T) {
	var warnings bytes.Buffer
	previous := iconWarnings
	iconWarnings = &warnings
	t.Cleanup(func() { iconWarnings = previous; setIconDir("") })
	setIconDir(t.TempDir()) // exists but holds no PNGs
	if items := iconAt("settings", 64, 0, 0, inkBlack); items != nil {
		t.Errorf("a missing icon must draw nothing, got %+v", items)
	}
	iconAt("settings", 64, 0, 0, inkBlack)
	if strings.Count(warnings.String(), "settings-64.png") != 1 {
		t.Errorf("want one warning naming the file, got %q", warnings.String())
	}
}

func TestIconsLoadFromTheRepoFolder(t *testing.T) {
	setIconDir("")
	for _, name := range []string{"back", "settings", "refresh", "import", "add", "check", "chevron-left", "chevron-right", "list", "grid", "apps"} {
		items := iconAt(name, 64, 10, 20, inkBlack)
		if len(items) != 1 || items[0].Shape != "icon" || items[0].W != 64 || items[0].X != 10 {
			t.Errorf("icon %s: %+v", name, items)
		}
	}
	if items := whiteIconAt("check", 64, 0, 0); len(items) != 1 || items[0].Icon.Pix[0] != 255 {
		t.Error("the white check must have white strokes")
	}
}

func TestIconCompositesOverWhatIsBelow(t *testing.T) {
	setIconDir("")
	page := Page{Items: append([]Item{{Shape: "rect", X: 0, Y: 0, W: 200, H: 200, Fill: inkBlack}}, whiteIconAt("check", 64, 20, 20)...)}
	canvas, err := renderPNG(page, &fontSet{}, greyLevelsOff)
	if err != nil {
		t.Fatal(err)
	}
	light := 0
	for y := 20; y < 84; y++ {
		for x := 20; x < 84; x++ {
			if canvas.GrayAt(x, y).Y > 200 {
				light++
			}
		}
	}
	if light < 40 {
		t.Errorf("only %d light pixels: the white check must show on the black disc", light)
	}
}

// ---- state fixtures ----

func TestEnglishFixtureIsPickedForTheEnglishLocale(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "preview-state.json")
	english := filepath.Join(dir, "preview-state.en.json")
	writeFile(t, base, `{}`)
	if got := localizedStatePath(base, "en"); got != base {
		t.Errorf("without a sibling the file is unchanged, got %s", got)
	}
	writeFile(t, english, `{}`)
	if got := localizedStatePath(base, "en"); got != english {
		t.Errorf("got %s, want %s", got, english)
	}
	if got := localizedStatePath(base, "vi"); got != base {
		t.Errorf("vi keeps the given file, got %s", got)
	}
	if got := localizedStatePath("", "en"); got != "" {
		t.Errorf("no state stays none, got %q", got)
	}
}

func TestShippedAppsHaveEnglishFixturesNextToEveryState(t *testing.T) {
	states, _ := filepath.Glob("../../apps/*/preview-state*.json")
	for _, state := range states {
		if strings.HasSuffix(state, ".en.json") {
			continue
		}
		if _, err := os.Stat(strings.TrimSuffix(state, ".json") + ".en.json"); err != nil {
			t.Errorf("%s has no English fixture", state)
		}
	}
}

// ---- 2.2 fonts ----

func TestVietnameseCoverageCheckFindsGapsInAFontWithoutThem(t *testing.T) {
	fonts, err := loadFonts("", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !fonts.hasText() {
		t.Skip("no system font")
	}
	for family, font := range fonts.fonts {
		hasAll := coversVietnamese(font)
		warned := false
		for _, warning := range fonts.warnings {
			warned = warned || strings.Contains(warning, family)
		}
		if family != familyMono && hasAll == warned {
			t.Errorf("%s: coverage %v but warned %v; a warning must appear exactly when letters are missing", family, hasAll, warned)
		}
	}
}

// ---- drawer renderers ----

func TestDrawerListPageGeometry(t *testing.T) {
	chrome := chromeFor("en")
	items := sampleDrawerItems("../../apps", "en", 8, "icon.png")
	page := drawerListPage(items, chrome, 0, drawerListExtras{})
	if page.Count != 2 {
		t.Fatalf("8 apps make %d pages, want 2 (7 rows a page)", page.Count)
	}
	rowDividers := countItems(page, func(item Item) bool { return item.Shape == "rect" && item.Fill == lineRow && item.H == separatorPx })
	if rowDividers != drawerListRows {
		t.Errorf("%d row dividers, want %d", rowDividers, drawerListRows)
	}
	rule := firstItem(page, func(item Item) bool { return item.Shape == "rect" && item.Fill == lineSection && item.H == separatorPx })
	if rule.Y != insetStatusBar+headerHeight+sectionHeight-separatorPx {
		t.Errorf("section rule at %d", rule.Y)
	}
	if chevrons := countItems(page, func(item Item) bool { return item.Label == "chevron-right" }); chevrons != drawerListRows+1 {
		t.Errorf("%d chevrons, want one per row plus the pager's", chevrons)
	}
	icons := countItems(page, func(item Item) bool { return item.Shape == "image" && item.W == rowIconSize && item.H == rowIconSize })
	if icons != drawerListRows {
		t.Errorf("%d row icons of 64 px, want %d (icon-64.png drawn 1:1)", icons, drawerListRows)
	}
	if countItems(page, func(item Item) bool {
		return item.Label == "update" && item.Shape == "ellipse" && item.W == updateDotSide
	}) < 2 {
		t.Error("want the update dot on the gear and on the app row")
	}
	last := page.Items[len(page.Items)-1]
	if last.Y+last.H > insetStatusBar+viewHeight {
		t.Errorf("content reaches %d, past the view bottom %d", last.Y+last.H, insetStatusBar+viewHeight)
	}
}

func TestDrawerPagerShowsOnlyWithMoreThanOnePage(t *testing.T) {
	one := drawerListPage(sampleDrawerItems("../../apps", "en", 3, "icon.png"), chromeFor("en"), 0, drawerListExtras{})
	if one.Count != 1 || countItems(one, func(item Item) bool { return item.Label == "chevron-left" }) != 0 {
		t.Error("a single page shows no pager in the drawer")
	}
	grid := drawerGridPage(sampleDrawerItems("../../apps", "en", 4*gridRows()+1, "icon.png"), chromeFor("en"), 0)
	if grid.Count != 2 || countItems(grid, func(item Item) bool { return item.Label == "chevron-left" }) != 1 {
		t.Errorf("a full grid plus one app has 2 pages and a pager, got %d pages", grid.Count)
	}
}

func TestDrawerHeaderHasRefreshListGridAndSettingsWithTheDot(t *testing.T) {
	items := drawerHeaderItems("Apps", true, true)
	var order []string
	for _, item := range items {
		if item.Shape == "icon" && item.Label != "back" {
			order = append(order, item.Label)
		}
	}
	if strings.Join(order, ",") != "refresh,list,grid,settings" {
		t.Errorf("header order %v", order)
	}
	dot := firstItem(Page{Items: items}, func(item Item) bool { return item.Label == "update" })
	settings := firstItem(Page{Items: items}, func(item Item) bool { return item.Label == "settings" })
	if dot.W != updateDotSide || dot.X != settings.X-(headerIconTarget-chevronIconSize)/2+headerIconTarget-updateDotSide-hostDotInset {
		t.Errorf("dot %+v not at the gear's inset", dot)
	}
	underline := countItems(Page{Items: items}, func(item Item) bool { return item.Shape == "rect" && item.H == 4 })
	if underline != 1 {
		t.Errorf("want the active grid toggle underlined once, got %d", underline)
	}
}

func TestDrawerGridRowsFitThePageArea(t *testing.T) {
	if drawerPageArea != viewHeight-headerHeight-drawerHintHeight-drawerFooterHeight || gridRows()*gridRowHeight > drawerPageArea {
		t.Errorf("page area %d, rows %d", drawerPageArea, gridRows())
	}
	page := drawerGridPage(sampleDrawerItems("../../apps", "en", 7, "icon.png"), chromeFor("en"), 0)
	if page.Count != 1 {
		t.Errorf("7 apps fit one page, got %d", page.Count)
	}
}

func TestTweaksModalIsCentredAndRendersBothLocales(t *testing.T) {
	for _, locale := range []string{"en", "vi"} {
		cells := sampleTweaks(locale)
		page := tweaksPage(cells, extrasCatalog(locale)["tweaks.title"], locale, 0)
		box := page.Items[len(nickelChromeItems())]
		if box.W != dialogWidth || box.Border != dialogBorder || box.X != (screenWidth-dialogWidth)/2 || box.Y != (screenHeight-box.H)/2 {
			t.Errorf("%s: box %+v is not the centred dialog box", locale, box)
		}
		if got := countItems(page, func(item Item) bool { return item.Shape == "text" && item.Family == familySerif }); got != 1 {
			t.Errorf("%s: %d serif texts, want only the dialog title", locale, got)
		}
		if page.Count != 1 {
			t.Errorf("%s: %d pages", locale, page.Count)
		}
	}
	if len(sampleTweaks("en")) != len(sampleTweaks("vi")) {
		t.Error("both locales need the same sample tweaks")
	}
}

func TestTweaksCaptionStacksUnderTheLabelAndKeepsTheSwitch(t *testing.T) {
	for _, locale := range []string{"en", "vi"} {
		cell := sampleTweaks(locale)[1]
		if cell.caption == "" || cell.kind != tweakToggle {
			t.Fatalf("%s: sample 1 should be the captioned screenshot button toggle: %+v", locale, cell)
		}
		items := tweaksCellItems(cell, 0, 0, 516, locale)
		var label, caption Item
		var hasSwitch bool
		for _, item := range items {
			switch {
			case item.Shape == "text" && item.Ink == inkMuted:
				caption = item
			case item.Shape == "text" && strings.Join(item.Lines, "\n") == cell.label:
				label = item
			case item.Shape != "text":
				hasSwitch = true
			}
		}
		if label.W == 0 || len(caption.Lines) != 1 || !strings.HasPrefix(caption.Lines[0], cell.caption[:8]) || !hasSwitch {
			t.Fatalf("%s: label %+v, caption %+v, switch %v", locale, label, caption, hasSwitch)
		}
		if caption.Y != label.Y+label.H || label.X+label.W > 516-tweaksCellPadding-toggleSwitchWidth {
			t.Errorf("%s: caption at y=%d under the label (y=%d h=%d), label must stop before the switch", locale, caption.Y, label.Y, label.H)
		}
	}
}

func TestDrawerCommandsWritePNGs(t *testing.T) {
	dir := t.TempDir()
	for _, command := range []struct {
		name string
		run  func([]string, *bytes.Buffer, *bytes.Buffer) int
	}{
		{"drawer-list", func(a []string, o, e *bytes.Buffer) int { return runDrawerList(a, o, e) }},
		{"drawer-grid", func(a []string, o, e *bytes.Buffer) int { return runDrawerGrid(a, o, e) }},
		{"extras-tweaks", func(a []string, o, e *bytes.Buffer) int { return runExtrasTweaks(a, o, e) }},
	} {
		out := filepath.Join(dir, command.name+".png")
		var stdout, stderr bytes.Buffer
		if code := command.run([]string{"--out", out, "--apps", "../../apps", "--locale", "vi"}, &stdout, &stderr); code != exitOK {
			t.Errorf("%s exited %d: %s", command.name, code, stderr.String())
		}
		if info, err := os.Stat(out); err != nil || info.Size() == 0 {
			t.Errorf("%s wrote no PNG", command.name)
		}
	}
}
