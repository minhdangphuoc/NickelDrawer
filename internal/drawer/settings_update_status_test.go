package drawer

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"koboapps/internal/i18n"
)

const checkTime = "2026-10-10T10:02:00Z"

// localClock is the check time as the default locale pattern writes it (dd/MM/yyyy HH:mm).
func localClock() string {
	moment, _ := time.Parse(time.RFC3339, checkTime)
	return i18n.FormatPattern(i18n.DefaultDateTimePattern, moment.Local())
}

func writeLastCheck(t *testing.T, config Config, record map[string]any) {
	t.Helper()
	record["time"] = checkTime
	data, _ := json.Marshal(record)
	writeFileIn(t, config.HostDir, "data/update/last-check.json", string(data))
}

func writeInstallStatus(t *testing.T, config Config, state, key, at string, params map[string]string) {
	t.Helper()
	data, _ := json.Marshal(map[string]any{"state": state, "time": at, "report": map[string]any{"key": key, "params": params}})
	writeFileIn(t, config.HostDir, "data/update/status.json", string(data))
}

// statusNotice is the text of the first notice of the screen (the status line comes before
// "What's new"), or "" when the screen shows none.
func statusNotice(t *testing.T, config Config) string {
	t.Helper()
	for _, component := range settingsBody(t, config).Body {
		if component.Type == "notice" && component.Title == "" {
			return component.Text
		}
	}
	return ""
}

// statusSteps is the label of the steps block, "" while no update job runs.
func statusSteps(t *testing.T, config Config) string {
	t.Helper()
	for _, component := range settingsBody(t, config).Body {
		if component.Type == "steps" {
			return component.Label
		}
	}
	return ""
}

func TestCheckOutcomeShowsInTheUpdatesSectionInBothLanguages(t *testing.T) {
	cases := []struct {
		name   string
		record map[string]any
		want   map[string]string
	}{
		{"up to date", map[string]any{"result": "up_to_date", "version": "1.0.0"}, map[string]string{
			"en": "Checked " + localClock() + " · Up to date (1.0.0)",
			"vi": "Đã kiểm tra " + localClock() + " · Đang dùng bản mới nhất (1.0.0)"}},
		{"available", map[string]any{"result": "available", "version": "1.1.0"}, map[string]string{
			"en": "Checked " + localClock() + " · New version 1.1.0 available",
			"vi": "Đã kiểm tra " + localClock() + " · Có bản mới 1.1.0"}},
		{"no wifi", map[string]any{"result": "failed", "error_key": "no_network"}, map[string]string{
			"en": "Could not check " + localClock() + ": no Wi-Fi",
			"vi": "Không kiểm tra được " + localClock() + ": chưa có Wi-Fi"}},
		{"unreachable", map[string]any{"result": "failed", "error_key": "unreachable"}, map[string]string{
			"en": "Could not check " + localClock() + ": cannot reach GitHub",
			"vi": "Không kiểm tra được " + localClock() + ": không kết nối được GitHub"}},
		{"rate limited", map[string]any{"result": "failed", "error_key": "rate_limited"}, map[string]string{
			"en": "Could not check " + localClock() + ": GitHub is limiting requests, try later",
			"vi": "Không kiểm tra được " + localClock() + ": GitHub đang giới hạn, thử lại sau"}},
		{"bad signature", map[string]any{"result": "failed", "error_key": "bad_signature"}, map[string]string{
			"en": "Could not check " + localClock() + ": bad signature, nothing changed",
			"vi": "Không kiểm tra được " + localClock() + ": chữ ký sai, không có gì thay đổi"}},
		{"no release", map[string]any{"result": "failed", "error_key": "no_release_on_channel"}, map[string]string{
			"en": "Could not check " + localClock() + ": no release on this channel",
			"vi": "Không kiểm tra được " + localClock() + ": kênh này chưa có bản phát hành"}},
		{"other", map[string]any{"result": "failed", "error_key": "other", "report": map[string]any{"key": "update.bad_channel",
			"params": map[string]string{"channel": "x", "file": "update.json"}}}, map[string]string{
			"en": "Could not check " + localClock() + ": Bad channel 'x' in update.json. Use stable or beta.",
			"vi": "Không kiểm tra được " + localClock() + ": Kênh 'x' trong update.json sai. Dùng stable hoặc beta."}},
	}
	for _, tc := range cases {
		for locale, want := range tc.want {
			config, _ := settingsConfig(t, "1.0.0", locale)
			writeLastCheck(t, config, tc.record)
			if got := statusNotice(t, config); got != want {
				t.Errorf("%s/%s:\n got %q\nwant %q", tc.name, locale, got, want)
			}
		}
	}
}

func TestNoCheckYetShowsNoStatusLine(t *testing.T) {
	config, _ := settingsConfig(t, "1.0.0", "en")
	if got := statusNotice(t, config); got != "" {
		t.Fatalf("notice = %q", got)
	}
}

func TestAvailableOutcomeForAnInstalledVersionIsHidden(t *testing.T) {
	config, _ := settingsConfig(t, "1.1.0", "en")
	writeLastCheck(t, config, map[string]any{"result": "available", "version": "1.1.0"})
	if got := statusNotice(t, config); got != "" {
		t.Fatalf("a stale outcome must not show: %q", got)
	}
}

func TestRunningCheckShowsCheckingStaysLiveAndDisablesCheck(t *testing.T) {
	for locale, want := range map[string]string{"en": "Checking for updates…", "vi": "Đang kiểm tra cập nhật…"} {
		config, _ := settingsConfig(t, "1.0.0", locale)
		writeLastCheck(t, config, map[string]any{"result": "up_to_date", "version": "1.0.0"})
		running, _ := json.Marshal(map[string]any{"state": "running", "job": "check", "pid": os.Getpid(), "time": checkTime,
			"report": map[string]any{"key": "update.checking"}})
		writeFileIn(t, config.HostDir, "data/update/status.json", string(running))
		reply := settingsBody(t, config)
		if got := statusSteps(t, config); got != want {
			t.Errorf("%s: steps = %q, want %q", locale, got, want)
		}
		if got := statusNotice(t, config); got != "" {
			t.Errorf("%s: a running check is progress, not a notice: %q", locale, got)
		}
		if !reply.Live {
			t.Errorf("%s: the screen must stay live so the outcome appears on its own", locale)
		}
		if check := findComponent(t, reply.Footer, "button", 0); *check.Enabled {
			t.Errorf("%s: Check must wait while a check runs", locale)
		}
	}
}

func TestInstallProgressAndResultsShowInBothLanguages(t *testing.T) {
	cases := []struct {
		name, state, key string
		params           map[string]string
		want             map[string]string
	}{
		{"running", "running", "update.running", nil, map[string]string{
			"en": "Downloading and installing…", "vi": "Đang tải và cài…"}},
		{"installed, restart", "done", "update.applied_reboot", map[string]string{"version": "1.1.0", "previous": "1.0.0"}, map[string]string{
			"en": "Installed 1.1.0 — restart the Kobo to finish", "vi": "Đã cài 1.1.0 — khởi động lại để hoàn tất"}},
		{"failed", "failed", "update.failed", map[string]string{"cause": "disk full"}, map[string]string{
			"en": "Update failed: disk full", "vi": "Cập nhật lỗi: disk full"}},
	}
	for _, tc := range cases {
		for locale, want := range tc.want {
			config, _ := settingsConfig(t, "1.0.0", locale)
			status := map[string]any{"state": tc.state, "time": checkTime, "report": map[string]any{"key": tc.key, "params": tc.params}}
			if tc.state == "running" {
				status["pid"] = os.Getpid()
			}
			data, _ := json.Marshal(status)
			writeFileIn(t, config.HostDir, "data/update/status.json", string(data))
			got := statusNotice(t, config)
			if tc.state == "running" {
				got = statusSteps(t, config) // progress is a steps block, not a notice
				if statusNotice(t, config) != "" {
					t.Errorf("%s/%s: progress must not be a notice", tc.name, locale)
				}
			}
			if got != want {
				t.Errorf("%s/%s: got %q, want %q", tc.name, locale, got, want)
			}
		}
	}
}

func TestNewestOfInstallResultAndCheckOutcomeWins(t *testing.T) {
	config, _ := settingsConfig(t, "1.0.0", "en")
	writeLastCheck(t, config, map[string]any{"result": "up_to_date", "version": "1.0.0"})
	writeInstallStatus(t, config, "failed", "update.failed", "2026-10-10T09:00:00Z", map[string]string{"cause": "old"})
	if got := statusNotice(t, config); got == "Update failed: old" || got == "" {
		t.Fatalf("a later check outcome replaces an older install failure: %q", got)
	}
	writeInstallStatus(t, config, "done", "update.applied_reboot", "2026-10-10T11:00:00Z", map[string]string{"version": "1.1.0"})
	if got := statusNotice(t, config); got != "Installed 1.1.0 — restart the Kobo to finish" {
		t.Fatalf("a later install result replaces the check outcome: %q", got)
	}
}

func TestStatusOfAFinishedCheckIsNotShownTwice(t *testing.T) {
	config, _ := settingsConfig(t, "1.0.0", "en")
	writeLastCheck(t, config, map[string]any{"result": "failed", "error_key": "no_network"})
	data, _ := json.Marshal(map[string]any{"state": "failed", "job": "check", "time": "2026-10-10T11:00:00Z",
		"report": map[string]any{"key": "update.no_network", "params": map[string]string{"cause": "x"}}})
	writeFileIn(t, config.HostDir, "data/update/status.json", string(data))
	if got, want := statusNotice(t, config), "Could not check "+localClock()+": no Wi-Fi"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestCheckAndApplyDeclareWifiSoTheDrawerTurnsItOnFirst(t *testing.T) {
	config, _ := settingsConfig(t, "1.0.0", "en")
	app, err := config.hostApp(HostSettingsID)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{updateActionCheck, updateActionApply} {
		action, found := findAction(app, id)
		if !found || !action.Wifi {
			t.Errorf("action %s must declare wifi: %+v", id, action)
		}
	}
	config.HasIP = func() bool { return false }
	for _, id := range []string{updateActionCheck, updateActionApply} {
		result, err := config.RunEvent(Event{AppID: HostSettingsID, ActionID: id, ScreenID: "main"})
		if err != nil || !result.NeedsWifi || result.OK {
			t.Errorf("%s without Wi-Fi must ask the mod to connect, not run: %+v err=%v", id, result, err)
		}
	}
}
