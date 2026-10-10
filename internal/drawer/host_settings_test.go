package drawer

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"koboapps/internal/i18n"
	"koboapps/internal/uikit"
)

// settingsConfig lays out <root>/.adds/koboapps with the real settings screen, locales and
// project.json, and a drawer Config pointing at it, so the tests cover the shipped files.
func settingsConfig(t *testing.T, version, locale string) (Config, string) {
	t.Helper()
	root := t.TempDir()
	hostDir := filepath.Join(root, ".adds", "koboapps")
	copyTree(t, filepath.Join("..", "..", "host", "screens"), filepath.Join(hostDir, "screens"))
	copyTree(t, filepath.Join("..", "..", "host", "config"), filepath.Join(hostDir, "config"))
	bundle, err := i18n.LoadBundle(filepath.Join("..", "..", "host", "locales"), locale)
	if err != nil {
		t.Fatal(err)
	}
	config := Config{AppsDir: filepath.Join(root, ".adds", "apps"), HostDir: hostDir, CacheDir: t.TempDir(),
		Locale: locale, Version: version, Bundle: bundle, StatusTimeout: 50 * time.Millisecond}
	writeFileIn(t, hostDir, "data/update.json", `{"repo":"o/r"}`)
	if err := os.MkdirAll(config.AppsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	return config, root
}

func copyTree(t *testing.T, source, target string) {
	t.Helper()
	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() {
			return walkErr
		}
		relative, _ := filepath.Rel(source, path)
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		destination := filepath.Join(target, relative)
		if mkdirErr := os.MkdirAll(filepath.Dir(destination), 0o755); mkdirErr != nil {
			return mkdirErr
		}
		return os.WriteFile(destination, data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func writeFileIn(t *testing.T, dir, relative, content string) {
	t.Helper()
	full := filepath.Join(dir, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func systemItem(t *testing.T, config Config, id string) Item {
	t.Helper()
	model, err := config.BuildModel()
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range model.Sections[1].Items {
		if item.ID == id {
			return item
		}
	}
	t.Fatalf("no system item %s", id)
	return Item{}
}

func settingsBody(t *testing.T, config Config) ScreenReply {
	t.Helper()
	reply, err := config.Screen(HostSettingsID, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	return reply
}

func kvRows(t *testing.T, components []ScreenComponent, index int) [][2]string {
	t.Helper()
	return findComponent(t, components, "kv", index).Rows.([][2]string)
}

const waitingUpdate = `{"available":true,"version":"1.1.0","current":"1.0.0","notes":"Faster scan.","checked":"2026-10-10T10:00:00Z"}`

func TestSettingsItemGetsADotOnceACheckFoundAnUpdate(t *testing.T) {
	config, _ := settingsConfig(t, "1.0.0", "en")
	item := systemItem(t, config, HostSettingsID)
	if item.Badge || item.Status != "" || !item.Screen {
		t.Fatalf("before any check: %+v", item)
	}
	writeFileIn(t, config.HostDir, "data/update/available.json", waitingUpdate)
	item = systemItem(t, config, HostSettingsID)
	if !item.Badge || item.Status != "1 update" {
		t.Fatalf("after a check: %+v", item)
	}
	model, _ := config.BuildModel()
	if !model.UpdateAvailable {
		t.Fatal("the model must say an update is waiting")
	}
	config.Version = "1.1.0" // the update was installed since the check
	if item = systemItem(t, config, HostSettingsID); item.Badge || item.Status != "" {
		t.Fatalf("a stale check must not show: %+v", item)
	}
}

func TestModelHasNoSeparateUpdatesItem(t *testing.T) {
	config, _ := settingsConfig(t, "1.0.0", "en")
	model, _ := config.BuildModel()
	for _, item := range model.Sections[1].Items {
		if item.ID == "_updates" {
			t.Fatalf("the updates item moved into the settings screen: %+v", item)
		}
	}
}

func TestUpdateRowsShowVersionLatestWithDotAndSource(t *testing.T) {
	config, _ := settingsConfig(t, "1.0.0", "en")
	writeFileIn(t, config.HostDir, "data/update/available.json", waitingUpdate)
	reply := settingsBody(t, config)
	if reply.Title != "NickelDrawer settings" || !reply.Live {
		t.Fatalf("title=%q live=%v", reply.Title, reply.Live)
	}
	rows := kvRows(t, reply.Body, 0)
	want := [][2]string{{"Version", "1.0.0"}, {"New version", "1.1.0"}, {"Source", "o/r"}}
	if len(rows) != 3 || rows[0] != want[0] || rows[1] != want[1] || rows[2] != want[2] {
		t.Fatalf("rows = %v, want %v", rows, want)
	}
	if notice := findComponent(t, reply.Body, "notice", 0); notice.Text != "Faster scan." {
		t.Fatalf("notice = %+v", notice)
	}
	apply := findComponent(t, reply.Footer, "button", 1)
	if apply.Action != "apply" || apply.Text != "Update to 1.1.0" || apply.Style != "primary" || !*apply.Enabled {
		t.Fatalf("apply button = %+v", apply)
	}
	if apply.Confirm == nil || !strings.Contains(apply.Confirm.Text, "1.1.0") {
		t.Fatalf("the confirm must name the version: %+v", apply.Confirm)
	}
	if check := findComponent(t, reply.Footer, "button", 0); check.Action != "check" || check.Style != "secondary" {
		t.Fatalf("check button = %+v", check)
	}
	if reply.Toasts["apply_started"] == "" {
		t.Fatal("the toasts the scripts print must be declared")
	}
}

func TestLatestRowIsHiddenUntilAnUpdateIsWaiting(t *testing.T) {
	config, _ := settingsConfig(t, "1.0.0", "en")
	reply := settingsBody(t, config)
	rows := kvRows(t, reply.Body, 0)
	if len(rows) != 2 || rows[0][0] != "Version" || rows[1][0] != "Source" {
		t.Fatalf("rows = %v", rows)
	}
	if len(reply.Footer) != 1 {
		t.Fatalf("with nothing to install the footer offers Check alone: %+v", reply.Footer)
	}
	if check := reply.Footer[0]; check.Action != "check" || check.Style != "primary" || !*check.Enabled {
		t.Fatalf("Check is the primary action until an update waits (A8): %+v", check)
	}
}

func TestSourceReadsOffAndButtonsWaitWhenNoRepoIsSet(t *testing.T) {
	config, _ := settingsConfig(t, "1.0.0", "en")
	writeFileIn(t, config.HostDir, "data/update.json", `{"repo":""}`)
	writeFileIn(t, config.HostDir, "data/update/available.json", waitingUpdate)
	reply := settingsBody(t, config)
	if rows := kvRows(t, reply.Body, 0); rows[len(rows)-1][1] != "Off" {
		t.Fatalf("rows = %v", rows)
	}
	if len(reply.Footer) != 1 || reply.Footer[0].Action != "check" || *reply.Footer[0].Enabled {
		t.Fatalf("Check needs a source, and a waiting update without one is not offered: %+v", reply.Footer)
	}
}

func TestButtonsWaitWhileARunIsActive(t *testing.T) {
	config, _ := settingsConfig(t, "1.0.0", "en")
	writeFileIn(t, config.HostDir, "data/update/available.json", waitingUpdate)
	status, _ := json.Marshal(map[string]any{"state": "running", "pid": os.Getpid(), "time": "2026-10-10T10:00:00Z",
		"report": map[string]any{"key": "update.running"}})
	writeFileIn(t, config.HostDir, "data/update/status.json", string(status))
	reply := settingsBody(t, config)
	check, apply := findComponent(t, reply.Footer, "button", 0), findComponent(t, reply.Footer, "button", 1)
	if *apply.Enabled || *check.Enabled {
		t.Fatalf("both buttons wait while a run is active: apply=%v check=%v", *apply.Enabled, *check.Enabled)
	}
	steps := findComponent(t, reply.Body, "steps", 0)
	if steps.Label != "Downloading and installing…" || steps.Done != "" || steps.Total != "" {
		t.Fatalf("the running state is a steps block without a count: %+v", steps)
	}
}

func TestSettingsEventsRunTheUpdateScripts(t *testing.T) {
	config, _ := settingsConfig(t, "1.0.0", "en")
	var ran []string
	config.RunScript = func(_ context.Context, script string) ScriptOutput {
		ran = append(ran, filepath.Base(script))
		return ScriptOutput{Stdout: `{"toast":"apply_started"}`}
	}
	config.HasIP = func() bool { return true }
	result, err := config.RunEvent(Event{AppID: HostSettingsID, ActionID: "apply", ScreenID: "main"})
	if err != nil {
		t.Fatal(err)
	}
	if len(ran) != 1 || ran[0] != "update-apply.sh" || result.Toast != "Updating. Keep Wi-Fi on." {
		t.Fatalf("ran=%v result=%+v", ran, result)
	}
	config.HasIP = func() bool { return false }
	if result, _ = config.RunEvent(Event{AppID: HostSettingsID, ActionID: "check", ScreenID: "main"}); !result.NeedsWifi {
		t.Fatalf("a network action without Wi-Fi must ask the mod to connect: %+v", result)
	}
}

func TestDebugToggleCreatesAndRemovesTheFlagFile(t *testing.T) {
	config, root := settingsConfig(t, "1.0.0", "en")
	flag := filepath.Join(root, ".adds", "nickeldrawer", "debug")
	if toggle := findComponent(t, settingsBody(t, config).Body, "toggle", 0); toggle.Value != false || *toggle.Enabled != true {
		t.Fatalf("debug toggle starts off and works: %+v", toggle)
	}
	on, off := "true", "false"
	if _, err := config.RunEvent(Event{AppID: HostSettingsID, ActionID: "debug", Value: &on, ScreenID: "main"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(flag); err != nil {
		t.Fatalf("the flag file must exist: %v", err)
	}
	if toggle := findComponent(t, settingsBody(t, config).Body, "toggle", 0); toggle.Value != true {
		t.Fatalf("debug toggle must read on: %+v", toggle)
	}
	if _, err := config.RunEvent(Event{AppID: HostSettingsID, ActionID: "debug", Value: &off}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(flag); !os.IsNotExist(err) {
		t.Fatalf("the flag file must be gone: %v", err)
	}
	if _, err := config.RunEvent(Event{AppID: HostSettingsID, ActionID: "debug", Value: &off}); err != nil {
		t.Fatalf("turning off twice is fine: %v", err)
	}
}

func TestToggleEventsNeedATrueOrFalseValue(t *testing.T) {
	config, _ := settingsConfig(t, "1.0.0", "en")
	maybe := "maybe"
	for _, value := range []*string{nil, &maybe} {
		_, err := config.RunEvent(Event{AppID: HostSettingsID, ActionID: "debug", Value: value})
		if drawerErr, ok := err.(*Error); !ok || drawerErr.Code != CodeUsage {
			t.Fatalf("value %v: err = %v", value, err)
		}
	}
}

func TestFirmwareRowMarksSupportedAndUnsupportedBuilds(t *testing.T) {
	config, root := settingsConfig(t, "1.0.0", "en")
	firmwareRow := func() string {
		rows := kvRows(t, settingsBody(t, config).Body, 1)
		return rows[0][1]
	}
	if got := firmwareRow(); got != "Unknown" {
		t.Fatalf("no version file: %q", got)
	}
	versionLine := func(build string) string {
		return "N000,4.1.15," + build + ",4.1.15,4.1.15,00000000-0000-0000-0000-000000000388"
	}
	writeFileIn(t, root, ".kobo/version", versionLine("4.38.23828"))
	if got := firmwareRow(); got != "4.38.23828 ✓" {
		t.Fatalf("supported build: %q", got)
	}
	writeFileIn(t, root, ".kobo/version", versionLine("4.38.23429"))
	if got := firmwareRow(); got != "4.38.23429 !" {
		t.Fatalf("unsupported build: %q", got)
	}
}

func TestAboutScreenCarriesTheProjectFromTheOneConfig(t *testing.T) {
	config, _ := settingsConfig(t, "1.2.3", "vi")
	reply, err := config.Screen(HostSettingsID, "about", nil)
	if err != nil {
		t.Fatal(err)
	}
	if reply.Title != "Giới thiệu" {
		t.Fatalf("title = %q", reply.Title)
	}
	qr := findComponent(t, reply.Body, "qr", 0)
	if qr.Text != "https://github.com/minhdangphuoc/NickelDrawer" || qr.Caption != "Quét để mở GitHub" {
		t.Fatalf("qr = %+v", qr)
	}
	texts := map[string]bool{}
	for _, component := range reply.Body {
		texts[component.Text] = true
	}
	if !texts["NickelDrawer"] || !texts["Phiên bản 1.2.3"] {
		t.Fatalf("heading and version must show: %v", texts)
	}
	rows := kvRows(t, reply.Body, 0)
	if rows[0] != [2]string{"Người bảo trì", "Minh Dang"} || rows[1] != [2]string{"GitHub", "minhdangphuoc/NickelDrawer"} || rows[3] != [2]string{"Giấy phép", "PolyForm Noncommercial 1.0.0"} ||
		rows[4][1] != "4.38.23552 – 4.38.23828" {
		t.Fatalf("project rows = %v", rows)
	}
	if credits := kvRows(t, reply.Body, 1); len(credits) != 6 {
		t.Fatalf("credits = %v", credits)
	}
}

func TestAboutIsOneLevelBelowMain(t *testing.T) {
	config, _ := settingsConfig(t, "1.0.0", "en")
	main := settingsBody(t, config)
	pushes := 0
	for _, component := range main.Body {
		if component.Type == "row" && component.Push == "about" {
			pushes++
		}
	}
	if pushes != 1 {
		t.Fatalf("main must push about once, got %d", pushes)
	}
}

func TestSettingsUIPassesTheKitValidatorAndAboutIsOneLevelDown(t *testing.T) {
	config, _ := settingsConfig(t, "1.0.0", "en")
	app, err := config.hostApp(HostSettingsID)
	if err != nil {
		t.Fatal(err)
	}
	doc, report, err := uikit.LoadDir(app.Dir, actionIDList(app))
	if err != nil || doc == nil {
		t.Fatalf("load: %v", err)
	}
	if !report.OK() || len(report.Warnings) != 0 {
		t.Fatalf("errors=%v warnings=%v", report.Errors, report.Warnings)
	}
	if _, found := doc.Screens["about"]; !found || doc.Start != "main" {
		t.Fatalf("start=%q screens=%d", doc.Start, len(doc.Screens))
	}
	if report.HasCode("ui.flow_too_deep") || report.HasCode("ui.flow_unreachable") {
		t.Fatalf("flow main -> about must be valid: %v", report.Errors)
	}
}
