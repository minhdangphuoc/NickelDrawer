package main

import (
	"strings"
	"testing"

	"koboapps/internal/uikit"
)

// uikitDocument loads the shipped UI Kit app with its English texts.
func uikitContext(t *testing.T, locale string) renderContext {
	t.Helper()
	doc, report, err := uikit.LoadDir("../../apps/uikit", []string{"state", "echo", "dialog", "toast", "steps", "refresh",
		"autorescan", "sampleenabled", "shownote", "refreshmode", "searchtext", "pagenumber"})
	if err != nil || !report.OK() {
		t.Fatalf("uikit app: %v %v", err, report.Errors)
	}
	return renderContext{doc: doc, locale: locale, vars: doc.Vars}
}

func TestRowDetailThatDoesNotFitMovesUnderTheLabel(t *testing.T) {
	short, long := "Balanced", "Chính, phụ, xác nhận và nhiều hơn nữa"
	if rowDetailIsSubline(short) {
		t.Errorf("%q fits the value column", short)
	}
	if !rowDetailIsSubline(long) {
		t.Errorf("%q must go under the label", long)
	}
	if rowDetailIsSubline("") {
		t.Error("no detail, no sub-line")
	}
}

func TestSamplesBlockHasOneBlockAndReservesCaptionLines(t *testing.T) {
	rc := uikitContext(t, "en")
	screen := rc.doc.Screens["button"]
	var samples uikit.Component
	for _, component := range screen.Body {
		if component.Kind == "samples" {
			samples = component
		}
	}
	blocks := rc.samplesBlocks(samples)
	// two rows of 120 px buttons, each with the 8 px gap and the caption line, 24 px apart
	want := 2*(buttonHeight+sampleCaptionGap+captionLineH) + rhythm
	if len(blocks) != 1 || blocks[0].height != want || blocks[0].gap != rhythm || blocks[0].keepWithNext {
		t.Fatalf("samples block = %+v, want one block of %d px", blocks, want)
	}
	items := blocks[0].draw(0)
	for _, item := range items {
		if item.Event != "" || item.Confirm != "" || item.Action != "" {
			t.Fatalf("a sample is inert, got %+v", item)
		}
	}
}

func TestPressedSampleInvertsTheCellExactly(t *testing.T) {
	normal := sampleCell{width: 357, height: buttonHeight, look: "normal", isButton: true, label: "Stop", enabled: true}
	pressed := normal
	pressed.look = "pressed"
	plain, inverted := normal.items(gutter, 0), pressed.items(gutter, 0)
	if inverted[0].Shape != "rect" || inverted[0].Fill != inkBlack || inverted[0].W != 357 || inverted[0].H != buttonHeight {
		t.Fatalf("the pressed cell starts with a black rectangle: %+v", inverted[0])
	}
	// plain: outline, label, (caption none); inverted: black cell, outline, label
	outline, label := plain[0], plain[1]
	invertedOutline, invertedLabel := inverted[1], inverted[2]
	if invertedOutline.Ink != paperFull-outline.Ink || invertedLabel.Ink != paperFull-label.Ink {
		t.Errorf("inks must be 255 minus the normal ink: %d/%d vs %d/%d", invertedOutline.Ink, invertedLabel.Ink, outline.Ink, label.Ink)
	}
}

func TestDisabledSampleDimsEveryInk(t *testing.T) {
	cell := sampleCell{width: 357, height: buttonHeight, look: "disabled", isButton: true, label: "Stop", primary: true, enabled: false}
	for _, item := range cell.items(gutter, 0) {
		if item.Shape == "text" && item.Ink != inkDim {
			t.Errorf("label ink %d, want dim", item.Ink)
		}
		if item.Shape == "rect" && item.Fill != noFill {
			t.Errorf("a disabled primary has no black fill: %+v", item)
		}
	}
}

func TestEveryUIKitPageRendersAndStaysBelowDepthTwo(t *testing.T) {
	for _, locale := range []string{"en", "vi"} {
		rc := uikitContext(t, locale)
		if got := len(rc.doc.Screens); got != 20 {
			t.Fatalf("%d screens, want main + 18 pages + viewer", got)
		}
		for id := range rc.doc.Screens {
			page, err := layoutScreen(rc, id, 0, Overlay{})
			if err != nil || len(page.Items) == 0 {
				t.Errorf("%s/%s: %v", locale, id, err)
			}
		}
	}
}

func TestUIKitHomeListsEighteenRowsInSixSections(t *testing.T) {
	rc := uikitContext(t, "en")
	sections, rows := 0, 0
	for _, component := range rc.doc.Screens["main"].Body {
		switch component.Kind {
		case "section":
			sections++
		case "row":
			rows++
			if component.Prop("push") == nil {
				t.Errorf("row %v must push its page", component.Main())
			}
		}
	}
	if sections != 6 || rows != 18 {
		t.Errorf("home has %d sections and %d rows, want 6 and 18", sections, rows)
	}
}

func TestRowRunsAreJoinedAndSectionsKeepTheirGroupTogether(t *testing.T) {
	row := func() block { return block{height: tapRowHeight, gap: rhythm, isTapRow: true} }
	section := func() block { return block{height: sectionHeight, keepWithNext: true, isSection: true} }
	blocks := keepSectionsTogether(joinRowRuns([]block{section(), row(), row(), row(), section(), row()}), 4*tapRowHeight)
	gaps := []int{blocks[0].gap, blocks[1].gap, blocks[2].gap, blocks[3].gap, blocks[4].gap, blocks[5].gap}
	if want := []int{0, 0, 0, 0, 0, rhythm}; !equalInts(gaps, want) {
		t.Errorf("gaps = %v, want %v: rows touch each other and the next section", gaps, want)
	}
	kept := []bool{blocks[0].keepWithNext, blocks[1].keepWithNext, blocks[2].keepWithNext, blocks[3].keepWithNext, blocks[4].keepWithNext, blocks[5].keepWithNext}
	for index, want := range []bool{true, true, true, false, true, false} {
		if kept[index] != want {
			t.Errorf("keepWithNext = %v, want %v (the last block of a group ends it)", kept, []bool{true, true, true, false, true, false})
			break
		}
	}
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for index := range a {
		if a[index] != b[index] {
			return false
		}
	}
	return true
}

func TestUIKitHomeBreaksOnGroupBoundaries(t *testing.T) {
	rc := uikitContext(t, "en")
	blocks := rc.blocksFor(rc.doc.Screens["main"].Body)
	bodyTop, bodyBottom := bodyBounds(0)
	available := bodyBottom - bodyTop - pageBarHeight - rhythm
	pages := packPages(keepSectionsTogether(blocks, available), available)
	if len(pages) != 3 {
		t.Fatalf("home has %d pages, want 3 (uikit-plan section 2)", len(pages))
	}
	for index, page := range pages {
		if !page[0].isSection {
			t.Errorf("page %d starts in the middle of a group", index+1)
		}
	}
}

func TestEmptyDrawerShowsIconHeadingCaptionAndOnePrimary(t *testing.T) {
	for _, locale := range []string{"en", "vi"} {
		chrome := chromeFor(locale)
		page := drawerEmptyPage(chrome, false)
		texts := map[string]Item{}
		icons, primaries := 0, 0
		for _, item := range page.Items {
			if len(item.Lines) == 1 {
				texts[item.Lines[0]] = item
			}
			if item.Shape == "icon" && item.Label == "apps" {
				icons++
				if item.W != 96 {
					t.Errorf("%s: the empty-state icon is 96 px, got %d", locale, item.W)
				}
			}
			if item.Shape == "rect" && item.Fill == inkBlack && item.Border == buttonBorder && item.W == contentWidth {
				primaries++
			}
		}
		if icons != 1 || primaries != 1 {
			t.Errorf("%s: %d icons and %d primaries, want one of each", locale, icons, primaries)
		}
		heading, found := texts[chrome.emptyTitle]
		if !found || heading.Family != familySerif || !heading.Bold || heading.Size != headingSize {
			t.Errorf("%s: heading %+v", locale, heading)
		}
		if _, found := texts[chrome.findApps]; !found {
			t.Errorf("%s: the primary %q is missing", locale, chrome.findApps)
		}
	}
}

func TestLoadingDrawerIsTheHeadingAlone(t *testing.T) {
	page := drawerEmptyPage(chromeFor("en"), true)
	for _, item := range page.Items {
		if item.Shape == "icon" && item.Label == "apps" {
			t.Error("Loading… has no icon")
		}
		if item.Shape == "rect" && item.Fill == inkBlack && item.W == contentWidth && item.Border == buttonBorder {
			t.Error("Loading… has no button")
		}
	}
}

func TestProblemRowIsOneBoundedLinePerText(t *testing.T) {
	long := "Không mở được ứng dụng: " + strings.Repeat("rất dài ", 40)
	items := problemRowItems("ung-dung-loi", long, 0)
	for _, item := range items {
		if item.Shape == "text" && len(item.Lines) != 1 {
			t.Errorf("a problem row text is one line, got %v", item.Lines)
		}
		if item.Shape == "text" && textMeter.width(item.Lines[0], textFace{size: item.Size, family: item.Family, isBold: item.Bold}) > item.W+1 {
			t.Errorf("%q is wider than its slot %d", item.Lines[0], item.W)
		}
	}
	if !items[0].Bold || items[1].Ink != inkMuted {
		t.Errorf("name must be bold and the status a muted caption: %+v %+v", items[0], items[1])
	}
}

func TestQt6NoticeTakesRowsFromTheList(t *testing.T) {
	chrome := chromeFor("en")
	items := sampleDrawerItems("../../apps", "en", 8, "icon.png")
	plain := drawerListPage(items, chrome, 0, drawerListExtras{})
	warned := drawerListPage(items, chrome, 0, drawerListExtras{hasQt6Notice: true})
	if plain.Count != 2 || warned.Count < plain.Count {
		t.Errorf("pages: plain %d, warned %d", plain.Count, warned.Count)
	}
	if listRowsThatFit(drawerPageArea) != drawerListRows || listRowsThatFit(drawerPageArea-190) >= drawerListRows {
		t.Error("the notice must cost list rows, the plain page keeps its 7")
	}
}
