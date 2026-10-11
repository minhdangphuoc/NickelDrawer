package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type device struct {
	root, conf, flags, locales string
}

func newDevice(t *testing.T) device {
	t.Helper()
	root := t.TempDir()
	return device{
		root:    root,
		conf:    filepath.Join(root, "Kobo eReader.conf"),
		flags:   filepath.Join(root, "extras"),
		locales: filepath.Join(root, "extras", "locales"),
	}
}

func (d device) run(t *testing.T, args ...string) (string, int) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	flags := []string{"--conf", d.conf, "--flags-dir", d.flags, "--locales-dir", d.locales}
	code := run(append(args, flags...), &stdout, &stderr)
	return stdout.String(), code
}

func writeLocale(t *testing.T, d device, name, content string) {
	t.Helper()
	if err := os.MkdirAll(d.locales, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d.locales, name+".json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestListReadsTheDeviceLocaleFile(t *testing.T) {
	d := newDevice(t)
	writeLocale(t, d, "vi", `{"tweak.screensaver.label": "Màn hình chờ"}`)
	output, code := d.run(t, "list", "--locale", "vi_VN")
	if code != exitOK {
		t.Fatalf("exit %d: %s", code, output)
	}
	var reply struct {
		V     int `json:"v"`
		Items []struct{ ID, Label, Kind string }
	}
	if err := json.Unmarshal([]byte(output), &reply); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, output)
	}
	if reply.V != 1 || len(reply.Items) != 9 || reply.Items[0].ID != "screensaver" || reply.Items[0].Label != "Màn hình chờ" {
		t.Fatalf("reply = %+v", reply)
	}
	if reply.Items[1].ID != "screenshot-button" || reply.Items[1].Label != "Screenshot button" {
		t.Fatalf("a key missing from the vi file must fall back to English: %+v", reply.Items[1])
	}
}

func TestMissingLocalesFolderFallsBackToEmbeddedEnglish(t *testing.T) {
	d := newDevice(t)
	output, code := d.run(t, "list")
	if code != exitOK || !strings.Contains(output, `"label":"Sleep screen"`) {
		t.Fatalf("exit %d: %s", code, output)
	}
}

func TestLocaleComesFromTheConfWhenNoFlagIsGiven(t *testing.T) {
	d := newDevice(t)
	if err := os.WriteFile(d.conf, []byte("[ApplicationPreferences]\nCurrentLocale=vi_VN\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeLocale(t, d, "vi", `{"tweak.sleep.label": "Ngủ"}`)
	output, _ := d.run(t, "list")
	if !strings.Contains(output, `"label":"Ngủ"`) {
		t.Fatalf("output = %s", output)
	}
}

func TestApplyTogglesAndReplies(t *testing.T) {
	d := newDevice(t)
	output, code := d.run(t, "apply", "auto-usb")
	if code != exitOK || strings.TrimSpace(output) != `{"v":1,"ok":true,"toast":"Auto USB connect on","value":true}` {
		t.Fatalf("exit %d: %s", code, output)
	}
	content, err := os.ReadFile(d.conf)
	if err != nil || !strings.Contains(string(content), "AutoUsbGadget=true") {
		t.Fatalf("conf = %q, err = %v", content, err)
	}
}

func TestApplyScreenshotButtonWritesTheOffFlag(t *testing.T) {
	d := newDevice(t)
	output, code := d.run(t, "apply", "screenshot-button")
	if code != exitOK || !strings.Contains(output, `"value":false`) {
		t.Fatalf("exit %d: %s", code, output)
	}
	if _, err := os.Stat(filepath.Join(d.flags, "screenshot-off")); err != nil {
		t.Fatalf("flag missing: %v", err)
	}
}

func TestFailuresExitTwoWithOkFalse(t *testing.T) {
	d := newDevice(t)
	tests := [][]string{{"apply", "nope"}, {"apply"}, {"list", "extra"}, {}, {"--nope"}}
	for _, args := range tests {
		output, code := d.run(t, args...)
		var reply struct {
			V     int    `json:"v"`
			OK    bool   `json:"ok"`
			Error string `json:"error"`
		}
		if err := json.Unmarshal([]byte(output), &reply); err != nil {
			t.Fatalf("%v: not JSON: %v\n%s", args, err, output)
		}
		if code != exitFailure || reply.V != 1 || reply.OK || reply.Error == "" {
			t.Errorf("%v: exit %d, reply %+v", args, code, reply)
		}
	}
}

func TestUnknownIDMessageIsLocalized(t *testing.T) {
	d := newDevice(t)
	output, code := d.run(t, "apply", "nope")
	if code != exitFailure || strings.TrimSpace(output) != `{"v":1,"ok":false,"error":"No tweak named 'nope'."}` {
		t.Fatalf("exit %d: %s", code, output)
	}
}
