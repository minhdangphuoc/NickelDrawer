package drawer

import (
	"os"
	"path/filepath"
	"testing"
)

func sendToggle(t *testing.T, config Config, action, value string) error {
	t.Helper()
	_, err := config.RunEvent(Event{AppID: HostSettingsID, ActionID: action, Value: &value, ScreenID: "main"})
	return err
}

func installedIDs(t *testing.T, config Config) []string {
	t.Helper()
	model, err := config.BuildModel()
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, item := range model.Sections[0].Items {
		ids = append(ids, item.ID)
	}
	return ids
}

func TestHiddenAppAppearsInTheModelOnlyWhileTheOverrideNamesIt(t *testing.T) {
	config, _ := settingsConfig(t, "1.0.0", "en")
	writeStatusApp(t, config.AppsDir, "shown")
	writeHiddenApp(t, config.AppsDir)
	if ids := installedIDs(t, config); len(ids) != 1 || ids[0] != "shown" {
		t.Fatalf("hidden by default: %v", ids)
	}
	if err := sendToggle(t, config, "show_hidden_postponed", "true"); err != nil {
		t.Fatal(err)
	}
	if ids := installedIDs(t, config); len(ids) != 2 {
		t.Fatalf("override must show the app: %v", ids)
	}
	if err := sendToggle(t, config, "show_hidden_postponed", "false"); err != nil {
		t.Fatal(err)
	}
	if ids := installedIDs(t, config); len(ids) != 1 {
		t.Fatalf("turning it off hides it again: %v", ids)
	}
}

func TestOverrideLivesInHostDataSoReinstallingTheAppKeepsIt(t *testing.T) {
	config, _ := settingsConfig(t, "1.0.0", "en")
	writeHiddenApp(t, config.AppsDir)
	if err := sendToggle(t, config, "show_hidden_postponed", "true"); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(filepath.Join(config.HostDir, "data", "settings.json"))
	if err != nil || string(saved) != "{\n  \"shown_hidden_apps\": [\n    \"postponed\"\n  ]\n}\n" {
		t.Fatalf("settings.json = %q, %v", saved, err)
	}
	if err := os.RemoveAll(filepath.Join(config.AppsDir, "postponed")); err != nil {
		t.Fatal(err)
	}
	writeHiddenApp(t, config.AppsDir) // the reinstall brings back hidden:true
	if ids := installedIDs(t, config); len(ids) != 1 || ids[0] != "postponed" {
		t.Fatalf("after reinstall: %v", ids)
	}
	matches, _ := filepath.Glob(filepath.Join(config.HostDir, "data", ".tmp-*"))
	if len(matches) != 0 {
		t.Fatalf("the atomic write must leave no temp file: %v", matches)
	}
}

func TestSettingsScreenOffersOneToggleAndHintPerHiddenApp(t *testing.T) {
	config, _ := settingsConfig(t, "1.0.0", "vi")
	writeHiddenApp(t, config.AppsDir)
	reply := settingsBody(t, config)
	toggle := findComponent(t, reply.Body, "toggle", 0)
	if toggle.Text != "postponed" || toggle.Value != false || toggle.Action != "show_hidden_postponed" || !*toggle.Enabled {
		t.Fatalf("toggle = %+v", toggle)
	}
	hints := captionsAfter(reply.Body, "postponed")
	if hints != "Đang ẩn · dành cho nhà phát triển" {
		t.Fatalf("hint = %q", hints)
	}
	if err := sendToggle(t, config, "show_hidden_postponed", "true"); err != nil {
		t.Fatal(err)
	}
	reply = settingsBody(t, config)
	if toggle = findComponent(t, reply.Body, "toggle", 0); toggle.Value != true {
		t.Fatalf("toggle must read on: %+v", toggle)
	}
	if hints = captionsAfter(reply.Body, "postponed"); hints != "Đang hiện trong Ngăn" {
		t.Fatalf("hint = %q", hints)
	}
}

// captionsAfter returns the text right after the toggle labelled name.
func captionsAfter(components []ScreenComponent, name string) string {
	for i, component := range components {
		if component.Type == "toggle" && component.Text == name && i+1 < len(components) {
			return components[i+1].Text
		}
	}
	return ""
}

func TestSettingsScreenWithoutHiddenAppsSaysSoAndHasOnlyTheDebugToggle(t *testing.T) {
	config, _ := settingsConfig(t, "1.0.0", "en")
	reply := settingsBody(t, config)
	toggles := 0
	var sawNone bool
	for _, component := range reply.Body {
		if component.Type == "toggle" {
			toggles++
		}
		sawNone = sawNone || component.Text == "No hidden apps installed."
	}
	if toggles != 1 || !sawNone {
		t.Fatalf("toggles=%d sawNone=%v", toggles, sawNone)
	}
}

func TestOnlyInstalledHiddenAppsCanBeSwitchedOn(t *testing.T) {
	config, _ := settingsConfig(t, "1.0.0", "en")
	writeStatusApp(t, config.AppsDir, "visible")
	for _, action := range []string{"show_hidden_ghost", "show_hidden_visible", "show_hidden_../x"} {
		err := sendToggle(t, config, action, "true")
		if drawerErr, ok := err.(*Error); !ok || drawerErr.Code != CodeUnknownApp {
			t.Errorf("%s: err = %v", action, err)
		}
	}
	if _, err := os.Stat(filepath.Join(config.HostDir, "data", "settings.json")); !os.IsNotExist(err) {
		t.Fatalf("a refused toggle must not write: %v", err)
	}
}

func TestDamagedSettingsFileMeansDefaults(t *testing.T) {
	config, _ := settingsConfig(t, "1.0.0", "en")
	writeHiddenApp(t, config.AppsDir)
	writeFileIn(t, config.HostDir, "data/settings.json", "{not json")
	if ids := installedIDs(t, config); len(ids) != 0 {
		t.Fatalf("damaged file: %v", ids)
	}
	if err := sendToggle(t, config, "show_hidden_postponed", "true"); err != nil {
		t.Fatal(err)
	}
	if ids := installedIDs(t, config); len(ids) != 1 {
		t.Fatalf("the next save repairs it: %v", ids)
	}
}

func TestShownHiddenAppsAreSortedAndUnique(t *testing.T) {
	settings := hostSettings{}.withHiddenApp("b", true).withHiddenApp("a", true).withHiddenApp("b", true)
	if len(settings.ShownHiddenApps) != 2 || settings.ShownHiddenApps[0] != "a" {
		t.Fatalf("settings = %+v", settings)
	}
	if settings = settings.withHiddenApp("a", false); len(settings.ShownHiddenApps) != 1 || settings.showsHiddenApp("a") {
		t.Fatalf("settings = %+v", settings)
	}
}
