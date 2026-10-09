package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmbeddedEnglishEqualsHostLocale(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "..", "host", "locales", "en.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(source, englishHostStrings) {
		t.Fatal("cmd/appsctl/embedded/en.json differs from host/locales/en.json; run `make sync-embedded`")
	}
	catalog, err := embeddedEnglish()
	if err != nil || catalog["menu.refresh"] == "" {
		t.Fatalf("embedded English unusable: %v", err)
	}
}

func TestMissingLocalesFolderStillRendersEnglishHostLabels(t *testing.T) {
	options := newOptions(t)
	options.hostDir = filepath.Join(t.TempDir(), "no-such-koboapps")
	options.locale = "vi"
	writeUpdateConfig(t, options.hostDir)
	writeApp(t, options.appsDir, "good-app", validManifest)
	writeApp(t, options.appsDir, "bad-app", invalidManifest)

	result, err := generateMenu(options)
	if err != nil {
		t.Fatal(err)
	}
	menu, _ := os.ReadFile(options.menuFile)
	for _, want := range []string{"Apps · Refresh list", "Apps · Errors (1)", "Apps · Check for updates", "Apps · Update status"} {
		if !strings.Contains(string(menu), want) {
			t.Errorf("menu lacks %q:\n%s", want, menu)
		}
	}
	for _, raw := range []string{"menu.refresh", "menu.errors", "menu.update"} {
		if strings.Contains(string(menu), raw) {
			t.Errorf("raw key %q reached the menu:\n%s", raw, menu)
		}
	}
	if got := refreshSummary(result); !strings.HasPrefix(got, "Refreshed: 1 apps") {
		t.Fatalf("summary = %q", got)
	}
}

func TestBrokenLocaleFileFallsBackToEnglishAndWarns(t *testing.T) {
	options := newOptions(t)
	options.locale = "vi"
	writeLocaleFile(t, filepath.Join(options.hostDir, "locales"), "vi.json", []byte("{broken"))
	writeLocaleFile(t, filepath.Join(options.hostDir, "locales"), "en.json", []byte("{broken"))
	bundle, err := options.hostBundle()
	if err == nil {
		t.Fatal("broken files should be reported")
	}
	if got := bundle.Message("menu.update_status", nil); got != "Apps · Update status" {
		t.Fatalf("message = %q", got)
	}
}

func TestMenuHasLocalizedUpdateItems(t *testing.T) {
	cases := map[string][]string{
		"en": {"Apps · Check for updates", "Apps · Update status"},
		"vi": {"Ứng dụng · Kiểm tra cập nhật", "Ứng dụng · Trạng thái cập nhật"},
	}
	for locale, labels := range cases {
		options := newOptions(t)
		options.locale = locale
		writeUpdateConfig(t, options.hostDir)
		writeApp(t, options.appsDir, "good-app", validManifest)
		if _, err := generateMenu(options); err != nil {
			t.Fatal(err)
		}
		menu, _ := os.ReadFile(options.menuFile)
		for _, label := range labels {
			if !strings.Contains(string(menu), label) {
				t.Errorf("%s menu lacks %q:\n%s", locale, label, menu)
			}
		}
		if !strings.Contains(string(menu), "/koboapps/bin/update-apply.sh") || !strings.Contains(string(menu), "/koboapps/bin/update-status.sh") {
			t.Errorf("%s menu lacks the update scripts:\n%s", locale, menu)
		}
	}
}

func TestVersionFlag(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"--version"}, &out, &errOut); code != exitOK || out.String() != "appsctl "+version+"\n" {
		t.Fatalf("code=%d out=%q", code, out.String())
	}
}

func TestHostMsgPrintsLocalizedText(t *testing.T) {
	options := newOptions(t)
	var out, errOut bytes.Buffer
	args := []string{"host-msg", "--locale", "vi", "--koboapps-dir", options.hostDir, "update.rolled_back", "version=1.2.3"}
	if code := run(args, &out, &errOut); code != exitOK || out.String() != "Đã khôi phục bản 1.2.3.\n" {
		t.Fatalf("code=%d out=%q err=%q", code, out.String(), errOut.String())
	}
}

func TestUpdateWithoutConfigExplainsWhatIsMissing(t *testing.T) {
	root := t.TempDir()
	options := newOptions(t)
	var out, errOut bytes.Buffer
	code := run([]string{"update", "apply", "--onboard-root", root, "--locale", "vi", "--koboapps-dir", options.hostDir, "--tmp-dir", t.TempDir()}, &out, &errOut)
	if code != exitFailure || !strings.Contains(out.String(), "Chưa thiết lập cập nhật") || !strings.Contains(out.String(), "update.json") {
		t.Fatalf("code=%d out=%q", code, out.String())
	}
}

func TestUpdateStatusBeforeAnyRunAndRollbackWithoutBackup(t *testing.T) {
	root := t.TempDir()
	var out, errOut bytes.Buffer
	if code := run([]string{"update", "status", "--onboard-root", root, "--locale", "en"}, &out, &errOut); code != exitOK ||
		out.String() != "Installed: "+version+"\nNo update has run yet.\n" {
		t.Fatalf("code=%d out=%q", code, out.String())
	}
	out.Reset()
	if code := run([]string{"update", "rollback", "--onboard-root", root, "--locale", "en"}, &out, &errOut); code != exitFailure ||
		out.String() != "No earlier version to restore.\n" {
		t.Fatalf("code=%d out=%q", code, out.String())
	}
	out.Reset()
	if code := run([]string{"update", "status", "--onboard-root", root, "--locale", "en"}, &out, &errOut); code != exitOK ||
		!strings.Contains(out.String(), "No earlier version to restore.") {
		t.Fatalf("status after a failed run: code=%d out=%q", code, out.String())
	}
}

func TestUpdateRejectsUnknownSubcommand(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"update", "install"}, &out, &errOut); code != exitFailure {
		t.Fatalf("code=%d", code)
	}
	if code := run([]string{"update"}, &out, &errOut); code != exitFailure {
		t.Fatalf("code=%d", code)
	}
}

// writeUpdateConfig creates update.json, which switches the update menu items on.
func writeUpdateConfig(t *testing.T, hostDir string) {
	t.Helper()
	writeLocaleFile(t, filepath.Join(hostDir, "data"), "update.json", []byte("{}"))
}
