package drawer

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"koboapps/internal/release"
)

// writeCatalogCache stores the catalog the last check verified, as the screen reads it.
func writeCatalogCache(t *testing.T, config Config, apps ...release.CatalogApp) {
	t.Helper()
	data, err := release.MarshalCatalog(&release.Catalog{V: 1, Sequence: 1, Issued: "2026-10-10T10:00:00Z", Apps: apps})
	if err != nil {
		t.Fatal(err)
	}
	writeFileIn(t, config.HostDir, "data/update/apps-catalog.json", string(data))
}

func catalogApp(id, version, name, vi string, hidden bool) release.CatalogApp {
	app := release.CatalogApp{ID: id, Version: version, Tag: release.AppTag(id, version), Zip: release.AppZipName(id, version),
		SHA256: strings.Repeat("a", 64), Size: 100, Name: name, Hidden: hidden}
	if vi != "" {
		app.Names = map[string]string{"vi": vi}
	}
	return app
}

func installAppFolder(t *testing.T, config Config, id, version string) {
	t.Helper()
	writeFileIn(t, config.AppsDir, id+"/app.json",
		`{"schema":1,"id":"`+id+`","name":"`+id+`","version":"`+version+`","actions":[{"id":"go","label":"Go","run":"bin/go.sh","output":"show"}]}`)
	writeFileIn(t, config.AppsDir, id+"/bin/go.sh", "#!/bin/sh\n")
	if err := os.Chmod(filepath.Join(config.AppsDir, id, "bin/go.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
}

// appsFixture: todo 1.0.0 installed with 1.0.1 offered, paper installed and current, wallpaper
// not installed, uikit (hidden) not installed.
func appsFixture(t *testing.T, locale string) Config {
	t.Helper()
	config, _ := settingsConfig(t, "1.0.0", locale)
	installAppFolder(t, config, "todo", "1.0.0")
	installAppFolder(t, config, "paper", "2.0.0")
	writeCatalogCache(t, config,
		catalogApp("todo", "1.0.1", "To-do", "Việc cần làm", false),
		catalogApp("paper", "2.0.0", "Paper", "", false),
		catalogApp("wallpaper", "0.4.0", "Wallpaper", "Hình nền", false),
		catalogApp("uikit", "1.0.0", "UI kit", "", true))
	return config
}

func appsScreen(t *testing.T, config Config) ScreenReply {
	t.Helper()
	reply, err := config.Screen(HostSettingsID, "apps", nil)
	if err != nil {
		t.Fatal(err)
	}
	return reply
}

func appsRowCaption(t *testing.T, config Config) string {
	t.Helper()
	for _, component := range settingsBody(t, config).Body {
		if component.Type == "row" && component.Push == "apps" {
			return component.Text + "|" + component.Detail
		}
	}
	t.Fatal("no Apps row on the settings screen")
	return ""
}

func TestSettingsAppsRowCaptionInBothLanguages(t *testing.T) {
	config, _ := settingsConfig(t, "1.0.0", "en")
	if got := appsRowCaption(t, config); got != "Apps|Not checked yet" {
		t.Errorf("no catalog: %q", got)
	}
	for locale, want := range map[string]string{"en": "Apps|1 update · 1 not installed", "vi": "Ứng dụng|1 bản mới · 1 chưa cài"} {
		if got := appsRowCaption(t, appsFixture(t, locale)); got != want {
			t.Errorf("%s: %q, want %q", locale, got, want)
		}
	}
	current, _ := settingsConfig(t, "1.0.0", "vi")
	installAppFolder(t, current, "paper", "2.0.0")
	writeCatalogCache(t, current, catalogApp("paper", "2.0.0", "Paper", "", false))
	if got := appsRowCaption(t, current); got != "Ứng dụng|Đã mới nhất" {
		t.Errorf("up to date: %q", got)
	}
}

func TestHiddenCatalogAppsAreOfferedOnlyWhenShownInSettings(t *testing.T) {
	config := appsFixture(t, "en")
	if view := config.appsView(); len(view.available) != 1 || view.available[0].ID != "wallpaper" {
		t.Fatalf("available = %+v", view.available)
	}
	if err := config.saveHostSettings(hostSettings{ShownHiddenApps: []string{"uikit"}}); err != nil {
		t.Fatal(err)
	}
	view := config.appsView()
	if len(view.available) != 2 {
		t.Fatalf("with uikit switched on: %+v", view.available)
	}
	if !config.offersApp(installAppActionStem, "uikit") {
		t.Error("a hidden app that is switched on must be installable")
	}
}

func TestAppsScreenListsSectionsRowsAndConfirmQuestions(t *testing.T) {
	config := appsFixture(t, "vi")
	reply := appsScreen(t, config)
	if reply.Status != "1 bản mới" || reply.Title != "Cập nhật ứng dụng" {
		t.Errorf("title=%q status=%q", reply.Title, reply.Status)
	}
	var kinds []string
	for _, component := range reply.Body {
		switch component.Type {
		case "section", "row", "kv":
			kinds = append(kinds, component.Type)
		}
	}
	if strings.Join(kinds, ",") != "section,row,section,kv,section,row" {
		t.Fatalf("body = %v", kinds)
	}
	update := findComponent(t, reply.Body, "row", 0)
	if update.Text != "Việc cần làm" || update.Detail != "1.0.0 → 1.0.1" || update.Action != "app_update_todo" {
		t.Errorf("update row = %+v", update)
	}
	confirm := update.Confirm
	if confirm == nil || confirm.Title != "Cập nhật Việc cần làm?" || confirm.OK != "Cập nhật" || confirm.Cancel != "Hủy" ||
		!strings.Contains(confirm.Text, "Lên bản 1.0.1. Dữ liệu và cài đặt của ứng dụng được giữ.") {
		t.Errorf("confirm = %+v", confirm)
	}
	install := findComponent(t, reply.Body, "row", 1)
	if install.Text != "Hình nền" || install.Detail != "0.4.0" || install.Action != "app_install_wallpaper" ||
		install.Confirm == nil || install.Confirm.Title != "Cài Hình nền?" {
		t.Errorf("install row = %+v", install)
	}
	installed := kvRows(t, reply.Body, 0)
	if len(installed) != 1 || installed[0] != [2]string{"Paper", "2.0.0"} {
		t.Errorf("installed = %v", installed)
	}
	for _, component := range reply.Body {
		if component.Type == "row" && component.Action == "apps-list" {
			t.Error("the marker row reached the reply")
		}
	}
}

func TestAppsScreenFooterPrimaryFollowsWhetherUpdatesExist(t *testing.T) {
	primaries := func(config Config) []string {
		var labels []string
		for _, button := range appsScreen(t, config).Footer {
			if button.Style == "primary" {
				labels = append(labels, button.Text+"/"+button.Action)
			}
		}
		return labels
	}
	with := primaries(appsFixture(t, "en"))
	if len(with) != 1 || with[0] != "Update all (1)/apps-update-all" {
		t.Errorf("with updates: %v", with)
	}
	config, _ := settingsConfig(t, "1.0.0", "en")
	if without := primaries(config); len(without) != 1 || without[0] != "Check/apps-check" {
		t.Errorf("without updates: %v", without)
	}
	reply := appsScreen(t, appsFixture(t, "en"))
	// CopperUI: a secondary button sits above the primary.
	if len(reply.Footer) != 2 || reply.Footer[0].Text != "Check" || reply.Footer[0].Style != "secondary" || reply.Footer[1].Style != "primary" {
		t.Errorf("footer = %+v", reply.Footer)
	}
}

func TestOnlyAppUpdatesLightTheGearAndItsStatus(t *testing.T) {
	config := appsFixture(t, "en")
	model, err := config.BuildModel()
	if err != nil || !model.UpdateAvailable {
		t.Fatalf("model: %+v err=%v", model, err)
	}
	if item := systemItem(t, config, HostSettingsID); !item.Badge || item.Status != "1 update" {
		t.Errorf("gear = %+v", item)
	}
	writeFileIn(t, config.HostDir, "data/update/available.json", waitingUpdate)
	if item := systemItem(t, config, HostSettingsID); !item.Badge || item.Status != "2 updates" {
		t.Errorf("gear with drawer and app updates = %+v", item)
	}
	installAppFolder(t, config, "todo", "1.0.1") // the update was installed since the check
	os.Remove(filepath.Join(config.HostDir, "data/update/available.json"))
	if item := systemItem(t, config, HostSettingsID); item.Badge {
		t.Errorf("an installed update must not light the gear: %+v", item)
	}
}

func TestRunningAppJobShowsTheStepsBlockAndDisablesTheButtons(t *testing.T) {
	config := appsFixture(t, "vi")
	writeFileIn(t, config.HostDir, "data/update/apps-job.json",
		`{"state":"running","job":"update","pid":`+itoa(os.Getpid())+`,"done":1,"total":2,"app":"wallpaper","phase":"download","time":"2026-10-10T10:00:00Z","report":{"key":"appupdate.working"}}`)
	reply := appsScreen(t, config)
	steps := findComponent(t, reply.Body, "steps", 0)
	if steps.Label != "Đang cập nhật" || steps.Done != "1" || steps.Total != "2" || steps.Detail != "Hình nền · đang tải và kiểm chữ ký" {
		t.Errorf("steps = %+v", steps)
	}
	for _, button := range reply.Footer {
		if button.Enabled == nil || *button.Enabled {
			t.Errorf("button %q must be disabled while a job runs", button.Text)
		}
	}
}

func TestFinishedAppJobLeavesAShortLocalizedNotice(t *testing.T) {
	cases := map[string]struct{ job, vi, en string }{
		"failed": {`{"state":"failed","job":"update","app":"todo","report":{"key":"appupdate.error.no_space","params":{"app":"todo"}}}`,
			"Không đủ dung lượng cho Việc cần làm", "Not enough space for To-do"},
		"updated": {`{"state":"done","job":"update","app":"todo","report":{"key":"appupdate.updated","params":{"app":"todo","version":"1.0.1"}}}`,
			"Đã cập nhật Việc cần làm lên bản 1.0.1", "Updated To-do to 1.0.1"},
		"checked": {`{"state":"done","job":"check","report":{"key":"appupdate.checked","params":{"updates":"1"}}}`, "", ""},
	}
	for name, test := range cases {
		for locale, want := range map[string]string{"vi": test.vi, "en": test.en} {
			config := appsFixture(t, locale)
			writeFileIn(t, config.HostDir, "data/update/apps-job.json", test.job)
			notice := ""
			for _, component := range appsScreen(t, config).Body {
				if component.Type == "notice" {
					notice = component.Text
				}
			}
			if notice != want {
				t.Errorf("%s/%s: notice = %q, want %q", name, locale, notice, want)
			}
		}
	}
}

func TestAppRowEventStartsTheDetachedUpdateWithTheAppId(t *testing.T) {
	config := appsFixture(t, "en")
	config.HasIP = func() bool { return true }
	marker := filepath.Join(t.TempDir(), "value")
	writeFileIn(t, config.HostDir, "bin/update-app.sh",
		"#!/bin/sh\nprintf %s \"$NICKELDRAWER_VALUE\" > '"+marker+"'\necho '{\"toast\":\"update_started\"}'\n")
	if err := os.Chmod(filepath.Join(config.HostDir, "bin/update-app.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
	result, err := config.RunEvent(Event{AppID: HostSettingsID, ActionID: "app_update_todo", ScreenID: "apps"})
	if err != nil || result.Toast != "Updating. Keep Wi-Fi on." || result.Text != "" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if got, _ := os.ReadFile(marker); string(got) != "todo" {
		t.Errorf("the script got app id %q", got)
	}
	config.HasIP = func() bool { return false }
	if result, _ = config.RunEvent(Event{AppID: HostSettingsID, ActionID: "app_update_todo", ScreenID: "apps"}); !result.NeedsWifi {
		t.Errorf("without Wi-Fi the mod must be asked to connect: %+v", result)
	}
}

func TestAppRowEventRefusesAppsTheScreenDoesNotOffer(t *testing.T) {
	config := appsFixture(t, "en")
	config.HasIP = func() bool { return true }
	for _, action := range []string{
		"app_update_ghost",     // not in the catalog
		"app_update_paper",     // already current
		"app_install_todo",     // installed, so no install row
		"app_update_wallpaper", // not installed, so no update row
		"app_install_uikit",    // hidden and not switched on
		"app_update_../x",      // an id that is not an id
	} {
		_, err := config.RunEvent(Event{AppID: HostSettingsID, ActionID: action, ScreenID: "apps"})
		if err == nil || !strings.Contains(err.Error(), CodeUnknownApp) {
			t.Errorf("%s: err = %v", action, err)
		}
	}
}

func TestRescanIsRequestedOnceAfterAnAppJob(t *testing.T) {
	config := appsFixture(t, "en")
	writeFileIn(t, config.HostDir, "data/update/apps-rescan", "")
	if requests := appsScreen(t, config).Requests; len(requests) != 1 || requests[0].Name != "rescan" || requests[0].Mode != "quick" {
		t.Fatalf("requests = %+v", requests)
	}
	if requests := appsScreen(t, config).Requests; len(requests) != 0 {
		t.Errorf("the request was handed over twice: %+v", requests)
	}
}

func TestOpeningTheDrawerRecoversAnInterruptedAppUpdate(t *testing.T) {
	config, _ := settingsConfig(t, "1.0.0", "en")
	installAppFolder(t, config, "todo", "1.0.0")
	// A power cut after the old folder moved aside and before the new one was renamed in.
	if err := os.Rename(filepath.Join(config.AppsDir, "todo"), filepath.Join(config.AppsDir, ".todo.old")); err != nil {
		t.Fatal(err)
	}
	writeFileIn(t, config.AppsDir, ".todo.new/data/list.json", "[1,2,3]")
	model, err := config.BuildModel()
	if err != nil {
		t.Fatal(err)
	}
	if len(model.Errors) != 0 || len(model.Sections[0].Items) != 1 || model.Sections[0].Items[0].ID != "todo" {
		t.Fatalf("model = %+v", model)
	}
	if data, _ := os.ReadFile(filepath.Join(config.AppsDir, "todo/data/list.json")); string(data) != "[1,2,3]" {
		t.Errorf("data = %q", data)
	}
}

func itoa(n int) string { return strconv.Itoa(n) }
