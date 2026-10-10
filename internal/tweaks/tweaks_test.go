package tweaks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"koboapps/internal/i18n"
)

type fixture struct {
	config   Config
	confPath string
	root     string
}

func newFixture(t *testing.T, conf string) fixture {
	t.Helper()
	root := t.TempDir()
	confPath := filepath.Join(root, ".kobo", "Kobo", "Kobo eReader.conf")
	if conf != "" {
		writeFile(t, confPath, conf)
	}
	english := i18n.Catalog{
		"tweak.screensaver.label": "Sleep screen", "tweak.toast.on": "{label} on", "tweak.toast.off": "{label} off",
		"tweak.toast.on_reopen": "{label} on. Reopen.", "tweak.toast.off_reopen": "{label} off. Reopen.",
		"tweak.ip.none": "No Wi-Fi", "tweak.toast.reboot": "Rebooting", "tweak.toast.sleep": "Sleeping",
		"tweak.toast.rescan": "Rescanning", "tweak.reboot.confirm": "Reboot now?",
		"tweak.err.unknown": "No tweak '{id}'.", "tweak.err.write": "Cannot save: {detail}",
		"tweak.err.no_screensaver": "No folder {folder}.",
		"tweak.screenshots.label":  "Screenshots", "tweak.legibility.label": "Legibility",
		"tweak.screenshots.caption": "With the power button",
	}
	return fixture{
		root: root, confPath: confPath,
		config: Config{
			Bundle: i18n.NewBundle(english), ConfPath: confPath,
			ScreensaverDir: filepath.Join(root, ".kobo", "screensaver"),
			IPAddress:      func() string { return "192.168.1.5" },
		},
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

func (f fixture) conf(t *testing.T) string { return readFile(t, f.confPath) }

func TestListKeepsOrderAndKinds(t *testing.T) {
	listing := newFixture(t, "").config.List()
	want := []string{"screensaver:toggle", "screenshots:toggle", "auto-usb:toggle", "legibility:toggle",
		"ip:info", "invert-reboot:action", "sleep:action", "reboot:action", "library-rescan:action"}
	var got []string
	for _, item := range listing.Items {
		got = append(got, item.ID+":"+item.Kind)
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("items = %v, want %v", got, want)
	}
}

func TestListJSONShape(t *testing.T) {
	fix := newFixture(t, "[FeatureSettings]\nScreenshots=true\n")
	if err := os.MkdirAll(fix.config.ScreensaverDir, 0o755); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(fix.config.List())
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		V     int              `json:"v"`
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	byID := map[string]map[string]any{}
	for _, item := range decoded.Items {
		byID[item["id"].(string)] = item
	}
	tests := []struct {
		id        string
		value     any
		hasValue  bool
		confirm   string
		isConfirm bool
	}{
		{id: "screensaver", value: true, hasValue: true},
		{id: "screenshots", value: true, hasValue: true},
		{id: "auto-usb", value: false, hasValue: true},
		{id: "legibility", value: false, hasValue: true},
		{id: "ip", value: "192.168.1.5", hasValue: true},
		{id: "sleep"},
		{id: "reboot", confirm: "Reboot now?", isConfirm: true},
	}
	if decoded.V != 1 {
		t.Fatalf("v = %d", decoded.V)
	}
	for _, test := range tests {
		item := byID[test.id]
		value, hasValue := item["value"]
		if hasValue != test.hasValue || (hasValue && value != test.value) {
			t.Errorf("%s value = %v (present %v), want %v (present %v)", test.id, value, hasValue, test.value, test.hasValue)
		}
		confirm, isConfirm := item["confirm"]
		if isConfirm != test.isConfirm || (isConfirm && confirm != test.confirm) {
			t.Errorf("%s confirm = %v, want %q", test.id, confirm, test.confirm)
		}
	}
}

func TestIPWithoutWifiIsLocalizedText(t *testing.T) {
	fix := newFixture(t, "")
	fix.config.IPAddress = func() string { return "" }
	for _, item := range fix.config.List().Items {
		if item.ID == IDIP && item.Value != "No Wi-Fi" {
			t.Fatalf("ip value = %v", item.Value)
		}
	}
}

func TestScreensaverToggleRenamesFolderBothWays(t *testing.T) {
	fix := newFixture(t, "")
	pictureFile := filepath.Join(fix.config.ScreensaverDir, "a.png")
	writeFile(t, pictureFile, "x")
	off := mustApply(t, fix.config, IDScreensaver)
	if off.Value == nil || *off.Value || off.Toast != "Sleep screen off" {
		t.Fatalf("off result = %+v", off)
	}
	disabledPicture := filepath.Join(fix.config.disabledScreensaverDir(), "a.png")
	if _, err := os.Stat(disabledPicture); err != nil {
		t.Fatalf("picture not kept: %v", err)
	}
	on := mustApply(t, fix.config, IDScreensaver)
	if on.Value == nil || !*on.Value || on.Toast != "Sleep screen on" {
		t.Fatalf("on result = %+v", on)
	}
	if _, err := os.Stat(pictureFile); err != nil {
		t.Fatalf("picture not restored: %v", err)
	}
}

func TestScreensaverToggleWithoutFolderFails(t *testing.T) {
	fix := newFixture(t, "")
	_, err := fix.config.Apply(IDScreensaver)
	assertMessage(t, err, "No folder "+fix.config.ScreensaverDir+".")
}

func TestBooleanTogglesWriteTheirKey(t *testing.T) {
	tests := []struct {
		id, section, key string
	}{
		{IDScreenshots, "FeatureSettings", "Screenshots"},
		{IDAutoUSB, "DeveloperSettings", "AutoUsbGadget"},
	}
	for _, test := range tests {
		t.Run(test.id, func(t *testing.T) {
			fix := newFixture(t, "")
			first := mustApply(t, fix.config, test.id)
			if first.Value == nil || !*first.Value {
				t.Fatalf("first result = %+v", first)
			}
			if want := "[" + test.section + "]\n" + test.key + "=true\n"; fix.conf(t) != want {
				t.Fatalf("conf = %q, want %q", fix.conf(t), want)
			}
			second := mustApply(t, fix.config, test.id)
			if second.Value == nil || *second.Value {
				t.Fatalf("second result = %+v", second)
			}
			if !strings.Contains(fix.conf(t), test.key+"=false") {
				t.Fatalf("conf = %q", fix.conf(t))
			}
		})
	}
}

func TestToggleKeepsOtherLinesAndSections(t *testing.T) {
	original := "[ApplicationPreferences]\nCurrentLocale=vi_VN\n; keep me\n\n[FeatureSettings]\nAcceptPreviewUpgrades=true\n\n[Reading]\nfont=Georgia\n"
	fix := newFixture(t, original)
	mustApply(t, fix.config, IDScreenshots)
	want := "[ApplicationPreferences]\nCurrentLocale=vi_VN\n; keep me\n\n[FeatureSettings]\nAcceptPreviewUpgrades=true\nScreenshots=true\n\n[Reading]\nfont=Georgia\n"
	if fix.conf(t) != want {
		t.Fatalf("conf = %q\nwant   %q", fix.conf(t), want)
	}
}

func TestLegibilityAddsAndRemovesTheLineAndMentionsReopen(t *testing.T) {
	fix := newFixture(t, "[Reading]\nfont=Georgia\n")
	on := mustApply(t, fix.config, IDLegibility)
	if on.Toast != "Legibility on. Reopen." || on.Value == nil || !*on.Value {
		t.Fatalf("on result = %+v", on)
	}
	if !strings.Contains(fix.conf(t), "[Reading]\nfont=Georgia\nwebkitTextRendering=optimizeLegibility") {
		t.Fatalf("conf = %q", fix.conf(t))
	}
	off := mustApply(t, fix.config, IDLegibility)
	if off.Toast != "Legibility off. Reopen." {
		t.Fatalf("off toast = %q", off.Toast)
	}
	if want := "[Reading]\nfont=Georgia\n"; fix.conf(t) != want {
		t.Fatalf("conf = %q, want %q", fix.conf(t), want)
	}
}

func TestInvertAndRebootFlipsSettingThenAsksForReboot(t *testing.T) {
	fix := newFixture(t, "")
	result := mustApply(t, fix.config, IDInvertReboot)
	if result.Nickel != "reboot" || result.Value != nil || result.Toast != "Rebooting" {
		t.Fatalf("result = %+v", result)
	}
	if !strings.Contains(fix.conf(t), "InvertScreen=true") {
		t.Fatalf("conf = %q", fix.conf(t))
	}
	mustApply(t, fix.config, IDInvertReboot)
	if !strings.Contains(fix.conf(t), "InvertScreen=false") {
		t.Fatalf("conf = %q", fix.conf(t))
	}
}

func TestNativeActions(t *testing.T) {
	tests := []struct{ id, nickel, toast string }{
		{IDSleep, "sleep", "Sleeping"},
		{IDReboot, "reboot", "Rebooting"},
		{IDLibraryRescan, "rescan-library", "Rescanning"},
	}
	for _, test := range tests {
		t.Run(test.id, func(t *testing.T) {
			fix := newFixture(t, "")
			result := mustApply(t, fix.config, test.id)
			if result.Nickel != test.nickel || result.Toast != test.toast || !result.OK || result.Value != nil {
				t.Fatalf("result = %+v", result)
			}
			if _, err := os.Stat(fix.confPath); err == nil {
				t.Fatal("a native action must not touch the conf file")
			}
		})
	}
}

func TestResultJSONShape(t *testing.T) {
	fix := newFixture(t, "")
	toggle, _ := json.Marshal(mustApply(t, fix.config, IDScreenshots))
	if want := `{"v":1,"ok":true,"toast":"Screenshots on","value":true}`; string(toggle) != want {
		t.Fatalf("toggle = %s, want %s", toggle, want)
	}
	action, _ := json.Marshal(mustApply(t, fix.config, IDReboot))
	if want := `{"v":1,"ok":true,"toast":"Rebooting","nickel":"reboot"}`; string(action) != want {
		t.Fatalf("action = %s, want %s", action, want)
	}
	failure, _ := json.Marshal(Failure{V: 1, Error: "x"})
	if want := `{"v":1,"ok":false,"error":"x"}`; string(failure) != want {
		t.Fatalf("failure = %s, want %s", failure, want)
	}
}

func TestUnknownAndInfoIDsFail(t *testing.T) {
	fix := newFixture(t, "")
	_, err := fix.config.Apply("nope")
	assertMessage(t, err, "No tweak 'nope'.")
	_, err = fix.config.Apply(IDIP)
	assertMessage(t, err, "No tweak 'ip'.")
}

func TestUnwritableConfIsAnError(t *testing.T) {
	fix := newFixture(t, "")
	writeFile(t, filepath.Join(fix.root, "blocker"), "file")
	fix.config.ConfPath = filepath.Join(fix.root, "blocker", "Kobo eReader.conf")
	_, err := fix.config.Apply(IDScreenshots)
	if err == nil || !strings.HasPrefix(err.Error(), "Cannot save:") {
		t.Fatalf("err = %v", err)
	}
}

func TestConfWriteLeavesNoTempFiles(t *testing.T) {
	fix := newFixture(t, "[FeatureSettings]\nScreenshots=false\n")
	mustApply(t, fix.config, IDScreenshots)
	entries, err := os.ReadDir(filepath.Dir(fix.confPath))
	if err != nil || len(entries) != 1 {
		t.Fatalf("entries = %v, err = %v", entries, err)
	}
}

func mustApply(t *testing.T, config Config, id string) Result {
	t.Helper()
	result, err := config.Apply(id)
	if err != nil {
		t.Fatalf("Apply(%s): %v", id, err)
	}
	return result
}

func assertMessage(t *testing.T, err error, want string) {
	t.Helper()
	var tweakErr *Error
	if err == nil || !asTweakError(err, &tweakErr) || tweakErr.Message != want {
		t.Fatalf("err = %v, want message %q", err, want)
	}
}

func asTweakError(err error, target **Error) bool {
	tweakErr, ok := err.(*Error)
	*target = tweakErr
	return ok
}

func TestOnlyTheScreenshotsTweakCarriesACaption(t *testing.T) {
	fix := newFixture(t, "")
	for _, item := range fix.config.List().Items {
		hasCaption := item.ID == IDScreenshots
		if (item.Caption != "") != hasCaption {
			t.Errorf("%s caption = %q", item.ID, item.Caption)
		}
	}
	encoded, _ := json.Marshal(Item{ID: "sleep", Label: "Sleep", Kind: KindAction})
	if strings.Contains(string(encoded), "caption") {
		t.Fatalf("an item without caption must not print the field: %s", encoded)
	}
}
