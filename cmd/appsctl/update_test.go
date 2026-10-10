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
	if err != nil || catalog["refresh.summary"] == "" {
		t.Fatalf("embedded English unusable: %v", err)
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
	if got := bundle.Message("refresh.summary", map[string]string{"apps": "2"}); got != "Refreshed: 2 apps" {
		t.Fatalf("message = %q", got)
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
