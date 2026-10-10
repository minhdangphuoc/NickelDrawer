package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runTweaks runs `appsctl drawer <args>` against a temp conf file and returns stdout and the exit code.
func runTweaks(t *testing.T, confPath, locale string, args ...string) (string, int) {
	t.Helper()
	root := newDrawerRoot(t)
	var stdout, stderr bytes.Buffer
	flags := append(root.flags(locale), "--conf", confPath)
	code := run(append(append([]string{"drawer"}, args...), flags...), &stdout, &stderr)
	return stdout.String(), code
}

func TestDrawerTweaksListsLocalizedItems(t *testing.T) {
	confPath := filepath.Join(t.TempDir(), "Kobo eReader.conf")
	output, code := runTweaks(t, confPath, "vi", "tweaks")
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
}

func TestDrawerTweakAppliesAndReplies(t *testing.T) {
	confPath := filepath.Join(t.TempDir(), "Kobo eReader.conf")
	output, code := runTweaks(t, confPath, "en", "tweak", "screenshots")
	if code != exitOK || strings.TrimSpace(output) != `{"v":1,"ok":true,"toast":"Screenshots on","value":true}` {
		t.Fatalf("exit %d: %s", code, output)
	}
	content, err := os.ReadFile(confPath)
	if err != nil || !strings.Contains(string(content), "Screenshots=true") {
		t.Fatalf("conf = %q, err = %v", content, err)
	}
}

func TestDrawerTweakFailureExitsTwoWithOkFalse(t *testing.T) {
	confPath := filepath.Join(t.TempDir(), "Kobo eReader.conf")
	output, code := runTweaks(t, confPath, "en", "tweak", "nope")
	if code != exitFailure || strings.TrimSpace(output) != `{"v":1,"ok":false,"error":"No tweak named 'nope'."}` {
		t.Fatalf("exit %d: %s", code, output)
	}
}
