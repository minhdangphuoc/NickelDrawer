package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"koboapps/internal/uikit"
)

// settingsPreviewDir is the host settings screen as a previewable app folder: its ui.json and
// locales plus the manifest of the actions the host builds in code.
func settingsPreviewDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "settings")
	for _, name := range []string{"ui.json", "locales/en.json", "locales/vi.json"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "host", "screens", "settings", filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(dir, filepath.FromSlash(name)), string(data))
	}
	script := filepath.Join(dir, "bin", "x.sh")
	writeFile(t, script, "")
	if err := os.Chmod(script, 0o755); err != nil {
		t.Fatal(err)
	}
	actions := ""
	for _, id := range []string{"check", "apply", "debug", "apps-check", "apps-update-all", "apps-list"} {
		actions += `{"id":"` + id + `","label":"x","run":"bin/x.sh","output":"show"},`
	}
	writeFile(t, filepath.Join(dir, "app.json"), `{"schema":1,"id":"settings","name":"Settings","version":"1.0.0","actions":[`+strings.TrimSuffix(actions, ",")+`]}`)
	return dir
}

func TestAppUpdatesPreviewShowsTheRowsOfTheState(t *testing.T) {
	for _, state := range []string{"settings-apps-state-en.json", "settings-apps-state-en-working.json", "settings-apps-state-en-uptodate.json"} {
		loaded, err := loadApp(settingsPreviewDir(t), "en")
		if err != nil {
			t.Fatal(err)
		}
		vars, err := readStateFile(filepath.Join("..", "..", "tools", "preview", state))
		if err != nil {
			t.Fatal(err)
		}
		merged := map[string]any{}
		for name, value := range loaded.doc.Vars {
			merged[name] = value
		}
		for name, value := range vars {
			merged[name] = value
		}
		page, err := layoutScreen(renderContext{doc: loaded.doc, locale: "en", vars: merged}, "apps", 0, Overlay{})
		if err != nil {
			t.Fatal(err)
		}
		var texts []string
		for _, item := range page.Items {
			texts = append(texts, strings.Join(item.Lines, " "))
		}
		all := strings.Join(texts, "\n")
		for _, want := range []string{"Wallpaper", "To-do", "INSTALLED"} {
			if !strings.Contains(strings.ToUpper(all), strings.ToUpper(want)) {
				t.Errorf("%s: preview lacks %q:\n%s", state, want, all)
			}
		}
		if strings.Contains(all, "apps-list") || strings.Contains(all, "$apps_rows") {
			t.Errorf("%s: the marker row or a binding leaked:\n%s", state, all)
		}
	}
}

// appsPage lays out the App updates screen of the settings host with a state file.
func appsPage(t *testing.T, locale, state string) Page {
	t.Helper()
	loaded, err := loadApp(settingsPreviewDir(t), locale)
	if err != nil {
		t.Fatal(err)
	}
	vars, err := readStateFile(filepath.Join("..", "..", "tools", "preview", state))
	if err != nil {
		t.Fatal(err)
	}
	merged := map[string]any{}
	for name, value := range loaded.doc.Vars {
		merged[name] = value
	}
	for name, value := range vars {
		merged[name] = value
	}
	page, err := layoutScreen(renderContext{doc: loaded.doc, locale: locale, vars: merged}, "apps", 0, Overlay{})
	if err != nil {
		t.Fatal(err)
	}
	return page
}

func textItemWith(page Page, text string) (Item, bool) {
	for _, item := range page.Items {
		if item.Shape == "text" && strings.Join(item.Lines, " ") == text {
			return item, true
		}
	}
	return Item{}, false
}

// Device bug: with one update ("1.0.1 → 1.0.2") beside installed apps, every row of the screen
// must keep one line: label left, version or detail right, on the label's own line.
func TestAppUpdatesWithOneUpdateKeepsEveryRowOnOneLine(t *testing.T) {
	for _, locale := range []string{"vi", "en"} {
		state := "settings-apps-state-" + locale + "-one.json"
		vars, err := readStateFile(filepath.Join("..", "..", "tools", "preview", state))
		if err != nil {
			t.Fatal(err)
		}
		page := appsPage(t, locale, state)
		rows, _ := vars[uikit.AppRowsVar].([]any)
		checked := 0
		for _, raw := range rows {
			row, _ := raw.(map[string]any)
			label, value := row["text"], row["detail"]
			if row["kind"] == "kv" {
				label, value = row["key"], row["value"]
			}
			if row["kind"] == "section" {
				continue
			}
			labelItem, hasLabel := textItemWith(page, label.(string))
			valueItem, hasValue := textItemWith(page, value.(string))
			if !hasLabel || !hasValue {
				t.Errorf("%s: row %q / %q not drawn whole", locale, label, value)
				continue
			}
			if valueItem.Y != labelItem.Y || valueItem.Align != "end" || len(valueItem.Lines) != 1 {
				t.Errorf("%s: %q value %q at y=%d align %s, label at y=%d: must share the label's line at the right",
					locale, label, value, valueItem.Y, valueItem.Align, labelItem.Y)
			}
			checked++
		}
		if checked != 4 {
			t.Errorf("%s: checked %d rows, want the update row and 3 installed apps", locale, checked)
		}
		if !hasIcon(page, "chevron-right") {
			t.Errorf("%s: the update row lost its chevron", locale)
		}
	}
}

func hasIcon(page Page, name string) bool {
	for _, item := range page.Items {
		if item.Shape == "icon" && strings.Contains(item.Label, name) {
			return true
		}
	}
	return false
}

// glyphlessArrowMeter is a font without "→" (the preview, like the device, then shows "->"),
// with sans advances at 36 px: dots and spaces narrow, the rest about 20 px.
type glyphlessArrowMeter struct{}

func (glyphlessArrowMeter) hasGlyph(r rune, _ textFace) bool { return r != '→' }

func (glyphlessArrowMeter) width(text string, _ textFace) int {
	width := 0
	for _, r := range text {
		width += 20
		if r == '.' || r == ' ' {
			width -= 10
		}
	}
	return width
}

func TestArrowBecomesASCIIWhenTheFontLacksIt(t *testing.T) {
	saved := textMeter
	defer func() { textMeter = saved }()
	face := regular(rowTextSize, familySans)
	textMeter = glyphlessArrowMeter{}
	if got := elide("1.0.1 → 1.0.2", rowDetailWidth, face, false); got != "1.0.1 -> 1.0.2" {
		t.Errorf("without the glyph: %q", got)
	}
	if rowDetailIsSubline("1.0.1 → 1.0.2") {
		t.Error("1.0.1 -> 1.0.2 must fit the detail column")
	}
	textMeter = arrowMeter{glyphlessArrowMeter{}}
	if got := elide("1.0.1 → 1.0.2", rowDetailWidth, face, false); got != "1.0.1 → 1.0.2" {
		t.Errorf("with the glyph: %q", got)
	}
}

// arrowMeter is the same font with the arrow.
type arrowMeter struct{ glyphlessArrowMeter }

func (arrowMeter) hasGlyph(rune, textFace) bool { return true }
