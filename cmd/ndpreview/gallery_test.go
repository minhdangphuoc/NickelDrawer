package main

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const galleryAppManifest = `{"schema":1,"id":"pics","name":"Pics","version":"1.0.0","actions":[
 {"id":"import","label":"Import","run":"bin/go.sh","output":"show"},
 {"id":"set-current","label":"Set","run":"bin/go.sh","output":"show"},
 {"id":"remove","label":"Remove","run":"bin/go.sh","output":"show"}]}`

const galleryAppUI = `{"kit":1,"start":"main",
 "vars":{"pictures":[],"pending":0,"selected":"","summary":"","mode":"fill"},
 "screens":{
  "main":{"title":"Pics","actions":[{"icon":"import","label":"Import","action":"import"},{"icon":"settings","label":"Options","push":"options"}],
   "body":[{"text":"$summary","style":"caption"},
    {"gallery":"$pictures","on_tap":{"push":"viewer"},"pending":"$pending",
     "empty":{"icon":"picture","title":"No pictures","text":"Copy some","button":"Import","action":"import"}}]},
  "viewer":{"title":"Viewer","body":[{"viewer":"$pictures","selected":"selected","current_status":"In use",
     "primary":{"label":"Set as wallpaper","label_current":"In use","action":"set-current"},
     "secondary":{"label":"Remove","action":"remove","confirm":{"text":"Sure?","ok":"Remove"}}}]},
  "options":{"title":"Options","body":[{"radio":"Mode","bind":"mode","options":[["fill","Fill","Crop the edges"],["fit","Fit"]]}]}}}`

// writeGalleryFixture creates the app and `count` pictures (first one current) with real PNG
// files in a state folder; it returns the app dir, the state path and the state's folder.
func writeGalleryFixture(t *testing.T, count, pending int) (appDir, statePath string) {
	t.Helper()
	appDir = filepath.Join(t.TempDir(), "pics")
	writeFile(t, filepath.Join(appDir, "app.json"), galleryAppManifest)
	writeFile(t, filepath.Join(appDir, "ui.json"), galleryAppUI)
	writeFile(t, filepath.Join(appDir, "bin", "go.sh"), "#!/bin/sh\n")
	if err := os.Chmod(filepath.Join(appDir, "bin", "go.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
	stateDir := t.TempDir()
	var pictures []map[string]any
	for index := 0; index < count; index++ {
		id := string(rune('a'+index/10)) + string(rune('0'+index%10))
		writeSolidPNG(t, filepath.Join(stateDir, "thumbs", id+".png"), 60, 80, 0)
		writeSolidPNG(t, filepath.Join(stateDir, "images", id+".png"), 90, 120, 0)
		picture := map[string]any{"id": id, "thumb": "thumbs/" + id + ".png", "image": "images/" + id + ".png", "label": id}
		if index == 0 {
			picture["badge"] = "current"
		}
		pictures = append(pictures, picture)
	}
	state, err := json.Marshal(map[string]any{"pictures": pictures, "pending": pending, "summary": ""})
	if err != nil {
		t.Fatal(err)
	}
	statePath = filepath.Join(stateDir, "state.json")
	writeFile(t, statePath, string(state))
	return appDir, statePath
}

func writeSolidPNG(t *testing.T, path string, width, height int, level uint8) {
	t.Helper()
	canvas := image.NewGray(image.Rect(0, 0, width, height))
	for index := range canvas.Pix {
		canvas.Pix[index] = level
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, canvas); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, encoded.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func renderFixture(t *testing.T, appDir, statePath string, options renderOptions) (Page, *image.Gray) {
	t.Helper()
	loaded, err := loadApp(appDir, "en")
	if err != nil {
		t.Fatal(err)
	}
	overrides, err := readStateFile(statePath)
	if err != nil {
		t.Fatal(err)
	}
	vars := map[string]any{}
	for name, value := range loaded.doc.Vars {
		vars[name] = value
	}
	for name, value := range overrides {
		vars[name] = value
	}
	setSelected(vars, options.Value)
	rc := renderContext{doc: loaded.doc, locale: "en", vars: vars, value: options.Value, baseDir: stateDir(statePath), chrome: options.Chrome}
	page, err := layoutScreen(rc, firstNonEmpty(options.Screen, "main"), options.Page, Overlay{})
	if err != nil {
		t.Fatal(err)
	}
	canvas, err := renderPNG(page, &fontSet{}, greyLevelsOff)
	if err != nil {
		t.Fatal(err)
	}
	return page, canvas
}

func countItems(page Page, match func(Item) bool) int {
	count := 0
	for _, item := range page.Items {
		if match(item) {
			count++
		}
	}
	return count
}

func isTileFrame(item Item) bool {
	return item.Shape == "rect" && item.W == 262 && item.H == 349
}

func TestGalleryDrawsTwelveTilesPerPageAndAPager(t *testing.T) {
	appDir, statePath := writeGalleryFixture(t, 14, 0)
	first, canvas := renderFixture(t, appDir, statePath, renderOptions{})
	if first.Count != 2 || countItems(first, isTileFrame) != 12 {
		t.Fatalf("page %d/%d with %d tiles, want 12 tiles on page 1/2", first.Index+1, first.Count, countItems(first, isTileFrame))
	}
	second, _ := renderFixture(t, appDir, statePath, renderOptions{Page: 1})
	if second.Index != 1 || countItems(second, isTileFrame) != 2 {
		t.Fatalf("page 2 has %d tiles, want 2", countItems(second, isTileFrame))
	}
	pager := countItems(first, func(item Item) bool { return len(item.Lines) == 1 && item.Lines[0] == "1 / 2" })
	arrows := countItems(first, func(item Item) bool { return item.Shape == "icon" && strings.HasPrefix(item.Label, "chevron-") })
	if pager != 1 || arrows != 2 {
		t.Errorf("want one pager text with two chevron icons, got %d and %d", pager, arrows)
	}
	// The thumbnail is black on white: its pixel is dark, the 1 px border of the next tile is not wide.
	if got := canvas.GrayAt(gutter+131, insetStatusBar+213-68+rhythm+174).Y; got != 0 {
		t.Errorf("thumbnail centre pixel = %d, want 0 (black picture)", got)
	}
}

func TestGalleryCurrentTileHasThickBorderAndBadge(t *testing.T) {
	appDir, statePath := writeGalleryFixture(t, 3, 0)
	page, _ := renderFixture(t, appDir, statePath, renderOptions{})
	thick := countItems(page, func(item Item) bool { return isTileFrame(item) && item.Border == galleryCurrentBdr })
	thin := countItems(page, func(item Item) bool { return isTileFrame(item) && item.Border == galleryBorder })
	badges := countItems(page, func(item Item) bool {
		return item.Shape == "ellipse" && item.W == galleryBadgeSize && item.Fill == inkBlack
	})
	if thick != 1 || thin != 2 || badges != 1 {
		t.Errorf("thick %d thin %d badges %d, want 1, 2, 1", thick, thin, badges)
	}
}

func TestGalleryDrawsPlaceholdersAndMissingGlyph(t *testing.T) {
	appDir, statePath := writeGalleryFixture(t, 2, 2)
	if err := os.Remove(filepath.Join(filepath.Dir(statePath), "thumbs", "a1.png")); err != nil {
		t.Fatal(err)
	}
	page, _ := renderFixture(t, appDir, statePath, renderOptions{})
	dashed := countItems(page, func(item Item) bool {
		return item.Dashed && isTileFrame(item) && item.Border == galleryDashWidth && item.Ink == inkDim
	})
	pictures := countItems(page, func(item Item) bool { return item.Shape == "image" })
	if dashed != 2 || pictures != 1 {
		t.Errorf("dashed placeholders %d (want 2), thumbnails %d (want 1: the other is missing)", dashed, pictures)
	}
}

func TestGalleryEmptyStateAndHiddenSummary(t *testing.T) {
	appDir, statePath := writeGalleryFixture(t, 0, 0)
	page, _ := renderFixture(t, appDir, statePath, renderOptions{})
	texts := map[string]bool{}
	for _, item := range page.Items {
		texts[strings.Join(item.Lines, "|")] = true
	}
	for _, want := range []string{"No pictures", "Copy some", "Import"} {
		if !texts[want] {
			t.Errorf("empty state lacks %q (texts %v)", want, texts)
		}
	}
	if countItems(page, isTileFrame) != 0 {
		t.Error("no tiles expected")
	}
	// The empty summary text takes no room: the first block after the header is the glyph.
	primary := countItems(page, func(item Item) bool { return item.Shape == "rect" && item.Fill == inkBlack && item.Event != "" })
	if primary != 1 {
		t.Errorf("want one primary button, got %d", primary)
	}
}

func TestGalleryReducesRowsWhenTheGridDoesNotFit(t *testing.T) {
	geometry := newGridGeometry(4, 3, 20, 700)
	if geometry.tileW != 262 || geometry.tileH != 349 || geometry.rows != 1 {
		t.Errorf("geometry %+v, want 262x349 tiles in 1 row", geometry)
	}
	if full := newGridGeometry(4, 3, 20, 1395); full.rows != 3 || full.pages != 2 {
		t.Errorf("geometry %+v, want 3 rows and 2 pages", full)
	}
}

func TestViewerChromeShownAndHidden(t *testing.T) {
	appDir, statePath := writeGalleryFixture(t, 3, 0)
	value := "a1"
	shown, _ := renderFixture(t, appDir, statePath, renderOptions{Screen: "viewer", Value: &value})
	texts := map[string]Item{}
	for _, item := range shown.Items {
		texts[strings.Join(item.Lines, "|")] = item
	}
	for _, want := range []string{"2 / 3", "Set as wallpaper", "Remove"} {
		if _, found := texts[want]; !found {
			t.Errorf("chrome shown lacks %q", want)
		}
	}
	if _, found := texts["In use"]; found {
		t.Error("a1 is not current: no status and no disabled primary")
	}
	picture := firstItem(shown, func(item Item) bool { return item.Shape == "image" })
	if picture.Y < insetStatusBar+headerHeight || picture.Y+picture.H > insetStatusBar+viewHeight-bottomMargin-viewerBarHeight {
		t.Errorf("picture %+v must lie between header and bottom bar", picture)
	}
	confirm := firstItem(shown, func(item Item) bool { return item.Label == "Remove" && item.Shape == "rect" })
	if confirm.W != viewerSecondaryWide || confirm.Confirm == "" {
		t.Errorf("secondary: %+v", confirm)
	}

	hidden, _ := renderFixture(t, appDir, statePath, renderOptions{Screen: "viewer", Value: &value, Chrome: chromeHidden})
	hiddenItems := withoutChrome(hidden)
	if len(hiddenItems) != 4 || hiddenItems[0].Shape != "image" {
		t.Fatalf("chrome hidden must draw the picture, the disc, the back target and the back icon, got %d items", len(hiddenItems))
	}
	disc, arrow := hiddenItems[1], hiddenItems[2]
	if disc.Shape != "ellipse" || disc.W != viewerBackDisc || disc.Fill != paperFull || disc.Border != viewerBackOutline {
		t.Errorf("back disc %+v: want a white %d px disc with a %d px outline", disc, viewerBackDisc, viewerBackOutline)
	}
	if arrow.X+arrow.W/2 != gutter+headerIconWidth/2 || arrow.W != viewerBackTarget {
		t.Errorf("back target %+v: want a %d px target centred on the header's arrow", arrow, viewerBackTarget)
	}
	if icon := hiddenItems[3]; icon.Shape != "icon" || icon.Label != "back" || icon.W != headerIconWidth {
		t.Errorf("back icon %+v: want the Tabler back arrow at %d px inside the disc", icon, headerIconWidth)
	}
	if hiddenItems[0].H != viewHeight && hiddenItems[0].W != screenWidth {
		t.Errorf("picture %dx%d must fill the area in one direction", hiddenItems[0].W, hiddenItems[0].H)
	}
}

// withoutChrome drops the status-bar and tab-bar strips finishPage puts first.
func withoutChrome(page Page) []Item {
	return page.Items[len(nickelChromeItems()):]
}

func firstItem(page Page, match func(Item) bool) Item {
	for _, item := range page.Items {
		if match(item) {
			return item
		}
	}
	return Item{}
}

func TestViewerOnCurrentPictureShowsStatusAndDisabledPrimary(t *testing.T) {
	appDir, statePath := writeGalleryFixture(t, 3, 0)
	value := "a0"
	page, _ := renderFixture(t, appDir, statePath, renderOptions{Screen: "viewer", Value: &value})
	if countItems(page, func(item Item) bool { return len(item.Lines) == 1 && item.Lines[0] == "In use" }) != 2 {
		t.Error("want the header status and the disabled primary both reading In use")
	}
	primary := firstItem(page, func(item Item) bool { return item.Shape == "rect" && item.Label == "In use" })
	if primary.Event != "" || primary.Ink != inkDim {
		t.Errorf("the primary must be disabled: %+v", primary)
	}
}

func TestHeaderIconActionsRadioDetailAndPictureGlyph(t *testing.T) {
	appDir, statePath := writeGalleryFixture(t, 1, 0)
	main, _ := renderFixture(t, appDir, statePath, renderOptions{})
	targets := countItems(main, func(item Item) bool {
		return item.Shape == "rect" && item.W == headerIconTarget && item.H == headerIconTarget && item.Y == insetStatusBar && item.Event != ""
	})
	if targets != 2 {
		t.Errorf("want 2 icon tap targets of 120x120, got %d", targets)
	}
	options, _ := renderFixture(t, appDir, statePath, renderOptions{Screen: "options"})
	rows := map[int]int{}
	for _, item := range options.Items {
		if item.Shape == "rect" && item.Event != "" && strings.Contains(item.Event, `"input"`) {
			rows[item.H]++
		}
	}
	if rows[radioDetailRow] != 1 || rows[rowHeight] != 1 {
		t.Errorf("radio rows by height: %v, want one 128 and one 104", rows)
	}
	if countItems(options, func(item Item) bool { return len(item.Lines) == 1 && item.Lines[0] == "Crop the edges" }) != 1 {
		t.Error("the sub-line is missing")
	}
}

func TestOversizedOrBrokenPicturesBecomeAGlyph(t *testing.T) {
	dir := t.TempDir()
	broken := filepath.Join(dir, "broken.png")
	if err := os.WriteFile(broken, []byte("not a png"), 0o644); err != nil {
		t.Fatal(err)
	}
	big := filepath.Join(dir, "big.png")
	if err := os.WriteFile(big, make([]byte, maxPictureBytes+1), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{broken, big, filepath.Join(dir, "missing.png"), ""} {
		if _, loaded := loadPicture(path); loaded {
			t.Errorf("%q must not load", path)
		}
	}
	if _, loaded := pictureItem(broken, image.Rect(0, 0, 10, 10)); loaded {
		t.Error("an undecodable picture has no item")
	}
	_ = color.Gray{}
}

func TestServePageEmbedsPicturesAndKeepsValue(t *testing.T) {
	appDir, statePath := writeGalleryFixture(t, 3, 0)
	instance := newTestServer(t, serverConfig{AppDir: appDir, Locale: "en", StatePath: statePath})
	body := get(t, instance, "/?screen=viewer&value=a1&chrome=hidden").Body.String()
	if !strings.Contains(body, "data:image/png;base64,") || strings.Contains(body, "Set as wallpaper") {
		t.Error("hidden chrome page must embed the picture and omit the bar")
	}
	grid := get(t, instance, "/").Body.String()
	if strings.Count(grid, "data:image/png;base64,") < 3 || !strings.Contains(grid, "&#34;event&#34;:&#34;screen&#34;") {
		t.Error("main page must embed 3 thumbnails and carry the tile tap event")
	}
}

func TestGalleryGridSpansTheContentGutters(t *testing.T) {
	for columns := 1; columns <= 6; columns++ {
		geometry := newGridGeometry(columns, 3, 1, 2000)
		right := geometry.left + columns*geometry.tileW + (columns-1)*galleryGutter
		leftGap, rightGap := geometry.left-gutter, screenWidth-gutter-right
		if leftGap < 0 || rightGap < leftGap || rightGap-leftGap > 1 {
			t.Errorf("%d columns: gaps %d and %d, want a grid inside the gutters with the leftover pixel split", columns, leftGap, rightGap)
		}
	}
	if geometry := newGridGeometry(4, 3, 1, 2000); geometry.left != gutter || geometry.tileW != 262 {
		t.Errorf("4 columns %+v, want flush with the gutters", geometry)
	}
}
