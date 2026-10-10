package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
