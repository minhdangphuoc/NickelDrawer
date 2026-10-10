package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeApp(t *testing.T, appsDir, id, manifestJSON string) {
	t.Helper()
	binDir := filepath.Join(appsDir, id, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "run.sh"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appsDir, id, "app.json"), []byte(manifestJSON), 0o644); err != nil {
		t.Fatal(err)
	}
}

const validManifest = `{"schema":1,"id":"good-app","name":"Good","version":"1.0.0",
 "actions":[{"id":"go","label":"Run","run":"bin/run.sh","output":"show"}]}`

const invalidManifest = `{"schema":1,"id":"bad-app","name":"Bad:Name","version":"1.0.0",
 "actions":[{"id":"go","label":"Run","run":"bin/run.sh","output":"show"}]}`

// writeLocale adds locales/<locale>.json to an app folder made by writeApp.
func writeLocale(t *testing.T, appsDir, id, locale, content string) {
	t.Helper()
	dir := filepath.Join(appsDir, id, "locales")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, locale+".json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// copyHostLocales installs the shipped host/locales into a fake koboapps folder, so the tests
// also prove those files load.
func copyHostLocales(t *testing.T, hostDir string) {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("..", "..", "host", "locales", "*.json"))
	if err != nil || len(files) == 0 {
		t.Fatalf("host locales not found: %v", err)
	}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		writeLocaleFile(t, filepath.Join(hostDir, "locales"), filepath.Base(file), data)
	}
}

func writeLocaleFile(t *testing.T, dir, name string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeConf(t *testing.T, locale string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "Kobo eReader.conf")
	content := "[ApplicationPreferences]\nCurrentLocale=" + locale + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// testOptions are the host-folder and apps-folder settings the tests share.
type testOptions struct {
	localeFlags
	appsDir string
}

func newOptions(t *testing.T) testOptions {
	root := t.TempDir()
	hostDir := filepath.Join(root, "koboapps")
	copyHostLocales(t, hostDir)
	return testOptions{
		localeFlags: localeFlags{locale: "en", hostDir: hostDir},
		appsDir:     filepath.Join(root, "apps"),
	}
}

const viLocale = `{"name":"Tốt","actions.go.label":"Chạy"}`

func TestMsgResolvesLocaleFallbackAndPlaceholders(t *testing.T) {
	appsDir := t.TempDir()
	writeApp(t, appsDir, "good-app", validManifest)
	writeLocale(t, appsDir, "good-app", "en", `{"msg.hello":"Hello {name}\nIP {ip}","msg.only_en":"English only"}`)
	writeLocale(t, appsDir, "good-app", "vi", `{"msg.hello":"Chào {name}\nIP {ip}"}`)
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"--locale", "vi", "msg.hello", "name=An", "ip=10.0.0.2"}, "Chào An\nIP 10.0.0.2\n"},
		{[]string{"--locale", "vi", "msg.only_en"}, "English only\n"},
		{[]string{"--locale", "fr", "msg.hello", "name=An", "ip=1"}, "Hello An\nIP 1\n"},
		{[]string{"--conf", writeConf(t, "vi_VN"), "msg.hello", "name=B", "ip=2"}, "Chào B\nIP 2\n"},
	}
	for _, c := range cases {
		var out, errOut bytes.Buffer
		args := append([]string{"msg", "--app", "good-app", "--apps-dir", appsDir}, c.args...)
		if code := run(args, &out, &errOut); code != exitOK || out.String() != c.want {
			t.Errorf("%v: code=%d out=%q err=%q, want %q", c.args, code, out.String(), errOut.String(), c.want)
		}
	}
}

func TestMsgUnknownKeyPrintsKeyWarnsAndSucceeds(t *testing.T) {
	appsDir := t.TempDir()
	writeApp(t, appsDir, "good-app", validManifest)
	var out, errOut bytes.Buffer
	code := run([]string{"msg", "--app", "good-app", "--apps-dir", appsDir, "msg.nope"}, &out, &errOut)
	if code != exitOK || out.String() != "msg.nope\n" || !strings.Contains(errOut.String(), "msg.nope") {
		t.Fatalf("code=%d out=%q err=%q", code, out.String(), errOut.String())
	}
	out.Reset()
	if code := run([]string{"msg", "--app", "../x", "--apps-dir", appsDir, "msg.nope"}, &out, &errOut); code != exitOK || out.String() != "msg.nope\n" {
		t.Fatalf("path-like app id: code=%d out=%q", code, out.String())
	}
}

func TestDeviceNamePrintsModelOrFallback(t *testing.T) {
	versionFile := filepath.Join(t.TempDir(), "version")
	line := "N000000000000,4.1.15,4.38.23828,4.1.15,4.1.15,00000000-0000-0000-0000-000000000388\n"
	if err := os.WriteFile(versionFile, []byte(line), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if code := run([]string{"device-name", "--version-file", versionFile}, &out, &errOut); code != exitOK || out.String() != "Kobo Libra 2\n" {
		t.Fatalf("code=%d out=%q", code, out.String())
	}
	out.Reset()
	if code := run([]string{"device-name", "--version-file", versionFile + ".missing"}, &out, &errOut); code != exitOK || out.String() != "Kobo\n" {
		t.Fatalf("fallback: code=%d out=%q", code, out.String())
	}
}

func TestValidateReportsBrokenTranslations(t *testing.T) {
	options := newOptions(t)
	writeApp(t, options.appsDir, "good-app", validManifest)
	writeLocale(t, options.appsDir, "good-app", "vi", `{"actions.go.label":"Có: hai chấm","stray":"x"}`)
	var out, errOut bytes.Buffer
	code := run([]string{"validate", "--apps-dir", options.appsDir}, &out, &errOut)
	if code != exitInvalid || !strings.Contains(out.String(), "must not contain ':'") || !strings.Contains(out.String(), `unknown key "stray"`) {
		t.Fatalf("code=%d out=%s", code, out.String())
	}
}

func TestValidateExitCodes(t *testing.T) {
	options := newOptions(t)
	writeApp(t, options.appsDir, "good-app", validManifest)
	var out, errOut bytes.Buffer
	if code := run([]string{"validate", "--apps-dir", options.appsDir}, &out, &errOut); code != exitOK {
		t.Fatalf("exit = %d, out=%s err=%s", code, out.String(), errOut.String())
	}
	writeApp(t, options.appsDir, "bad-app", invalidManifest)
	out.Reset()
	if code := run([]string{"validate", filepath.Join(options.appsDir, "bad-app")}, &out, &errOut); code != exitInvalid {
		t.Fatalf("exit = %d, want %d", code, exitInvalid)
	}
	if !strings.Contains(out.String(), "INVALID") {
		t.Fatalf("out = %s", out.String())
	}
}

func TestUnknownCommandIsAUsageError(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"nope"}, &out, &errOut); code != exitFailure {
		t.Fatalf("exit = %d", code)
	}
}
