package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"koboapps/internal/drawer"
)

var updateGolden = flag.Bool("update", false, "rewrite the golden files in testdata/drawer")

// drawerRoot is a fake onboard root: apps/ and koboapps/ with the shipped host locales.
type drawerRoot struct {
	apps, host, menu, errors string
}

func newDrawerRoot(t *testing.T) drawerRoot {
	t.Helper()
	base := t.TempDir()
	root := drawerRoot{
		apps: filepath.Join(base, "apps"), host: filepath.Join(base, "koboapps"),
		menu: filepath.Join(base, "nm", "koboapps"), errors: filepath.Join(base, "errors.txt"),
	}
	for _, dir := range []string{root.apps, filepath.Join(root.host, "data"), filepath.Dir(root.menu)} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	copyHostLocales(t, root.host)
	return root
}

func (r drawerRoot) flags(locale string) []string {
	return []string{"--apps-dir", r.apps, "--koboapps-dir", r.host, "--locale", locale,
		"--out", r.menu, "--errors-file", r.errors, "--cache-dir", filepath.Join(filepath.Dir(r.apps), "cache"),
		"--conf", filepath.Join(filepath.Dir(r.apps), "no-such.conf")}
}

// call runs `appsctl drawer <args>` in process and decodes the JSON reply.
func (r drawerRoot) call(t *testing.T, locale string, args ...string) (map[string]any, int, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := run(append(append([]string{"drawer"}, args...), r.flags(locale)...), &stdout, &stderr)
	var reply map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &reply); err != nil {
		t.Fatalf("stdout is not one JSON document: %v\n%s", err, stdout.String())
	}
	return reply, code, stdout.String()
}

// writeScriptApp writes an app whose actions run the given script bodies (file name -> body).
func writeScriptApp(t *testing.T, appsDir, manifestJSON string, scripts map[string]string) string {
	t.Helper()
	var header struct{ ID string }
	if err := json.Unmarshal([]byte(manifestJSON), &header); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(appsDir, header.ID)
	if err := os.MkdirAll(filepath.Join(dir, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range scripts {
		if err := os.WriteFile(filepath.Join(dir, "bin", name), []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "app.json"), []byte(manifestJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

const alphaManifest = `{"schema":1,"id":"alpha","name":"Alpha","version":"1.2.0","description":"First app",
 "storage":{"reads":["inbox"],"writes":["books"]},"network":true,
 "actions":[
  {"id":"start","label":"Start","run":"bin/start.sh","output":"show","wifi":true,"rescan":"quick"},
  {"id":"stop","label":"Stop","run":"bin/stop.sh","output":"show"},
  {"id":"status","label":"Status","run":"bin/status.sh","output":"show","menu":false},
  {"id":"bg","label":"Background","run":"bin/bg.sh","output":"none"}]}`

const betaManifest = `{"schema":1,"id":"beta","name":"Beta","version":"0.1.0",
 "actions":[{"id":"open","label":"Open","run":"bin/open.sh","output":"show"}]}`

func installFixtureApps(t *testing.T, root drawerRoot) {
	t.Helper()
	writeScriptApp(t, root.apps, alphaManifest, map[string]string{
		"start.sh":  "echo started; echo 'line two'",
		"stop.sh":   "echo 'oops' >&2; echo failing; exit 3",
		"status.sh": `echo '{"running":true,"line":"Receiving 4821"}'`,
		"bg.sh":     "exit 0",
	})
	writeLocale(t, root.apps, "alpha", "vi", `{"name":"Anpha","actions.start.label":"Bắt đầu","actions.stop.label":"Dừng"}`)
	writeScriptApp(t, root.apps, betaManifest, map[string]string{"open.sh": "echo hi"})
	writeApp(t, root.apps, "broken", `{"schema":1,"id":"broken","name":"","version":"1.0.0","actions":[]}`)
	if err := os.WriteFile(filepath.Join(root.host, "data", "update.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertGolden(t *testing.T, name, raw string) {
	t.Helper()
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, []byte(raw), "", "  "); err != nil {
		t.Fatal(err)
	}
	pretty.WriteString("\n")
	path := filepath.Join("testdata", "drawer", name)
	if *updateGolden {
		if err := os.WriteFile(path, pretty.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("missing golden file (run go test -update): %v", err)
	}
	if string(want) != pretty.String() {
		t.Errorf("%s differs from the golden file; run go test -update and review the diff.\ngot:\n%s", name, pretty.String())
	}
}

func TestDrawerModelGolden(t *testing.T) {
	for _, locale := range []string{"en", "vi"} {
		t.Run(locale, func(t *testing.T) {
			root := newDrawerRoot(t)
			installFixtureApps(t, root)
			_, code, raw := root.call(t, locale, "model")
			if code != exitOK {
				t.Fatalf("exit %d: %s", code, raw)
			}
			assertGolden(t, "model_"+locale+".json", raw)
		})
	}
}

func TestDrawerModelOmitsUpdatesWithoutConfig(t *testing.T) {
	root := newDrawerRoot(t)
	_, _, raw := root.call(t, "en", "model")
	if strings.Contains(raw, "_updates") || !strings.Contains(raw, "_refresh") {
		t.Errorf("want _refresh only: %s", raw)
	}
}

func TestDrawerRun(t *testing.T) {
	root := newDrawerRoot(t)
	installFixtureApps(t, root)
	t.Run("success", func(t *testing.T) {
		reply, code, _ := root.call(t, "en", "run", "alpha", "start")
		if code != exitOK || reply["ok"] != true || reply["text"] != "started\nline two" || reply["rescan"] != "quick" {
			t.Errorf("unexpected reply (exit %d): %v", code, reply)
		}
	})
	t.Run("failure keeps stdout and skips rescan", func(t *testing.T) {
		reply, code, _ := root.call(t, "en", "run", "alpha", "stop")
		if code != exitOK || reply["ok"] != false || reply["text"] != "failing" || reply["rescan"] != "none" {
			t.Errorf("unexpected reply (exit %d): %v", code, reply)
		}
	})
	t.Run("background", func(t *testing.T) {
		reply, code, _ := root.call(t, "en", "run", "alpha", "bg")
		if code != exitOK || reply["ok"] != true {
			t.Errorf("unexpected reply (exit %d): %v", code, reply)
		}
	})
	for name, args := range map[string][]string{
		"unknown action": {"alpha", "nope"}, "unknown app": {"ghost", "start"},
		"invalid app": {"broken", "x"}, "traversal": {"../alpha", "start"}, "missing argument": {"alpha"},
	} {
		t.Run(name, func(t *testing.T) {
			reply, code, _ := root.call(t, "en", append([]string{"run"}, args...)...)
			if code == exitOK || reply["error"] == nil {
				t.Errorf("want an error document and non-zero exit, got %d %v", code, reply)
			}
		})
	}
}

func TestDrawerRunNeedsWifi(t *testing.T) {
	root := newDrawerRoot(t)
	installFixtureApps(t, root)
	for _, hasIP := range []bool{false, true} {
		drawerHooks = func(config *drawer.Config) { config.HasIP = func() bool { return hasIP } }
		reply, code, _ := root.call(t, "en", "run", "alpha", "start")
		_, needsWifi := reply["needs_wifi"]
		if code != exitOK || needsWifi == hasIP {
			t.Errorf("hasIP=%v: want needs_wifi=%v, got %v", hasIP, !hasIP, reply)
		}
	}
	drawerHooks = func(*drawer.Config) {}
}

func TestDrawerRunRefreshHostItem(t *testing.T) {
	root := newDrawerRoot(t)
	installFixtureApps(t, root)
	reply, code, _ := root.call(t, "en", "run", "_refresh", "refresh")
	if code != exitOK || reply["ok"] != true {
		t.Fatalf("unexpected reply (exit %d): %v", code, reply)
	}
	if menu, err := os.ReadFile(root.menu); err != nil || !strings.Contains(string(menu), "Alpha") {
		t.Errorf("menu file not regenerated: %v %s", err, menu)
	}
}

func TestDrawerInfo(t *testing.T) {
	root := newDrawerRoot(t)
	installFixtureApps(t, root)
	reply, code, _ := root.call(t, "vi", "info", "alpha")
	storage, _ := reply["storage"].(map[string]any)
	if code != exitOK || reply["name"] != "Anpha" || reply["network"] != true ||
		reply["path"] != filepath.Join(root.apps, "alpha") || storage["writes"].([]any)[0] != "books" {
		t.Errorf("unexpected info (exit %d): %v", code, reply)
	}
	if _, code, _ := root.call(t, "en", "info", "_refresh"); code == exitOK {
		t.Error("host items have no info")
	}
}

func TestDrawerRemove(t *testing.T) {
	root := newDrawerRoot(t)
	installFixtureApps(t, root)
	outside := t.TempDir()
	sentinel := filepath.Join(outside, "keep.txt")
	if err := os.WriteFile(sentinel, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root.apps, "escape")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root.apps, "beta"), filepath.Join(root.apps, "alias")); err != nil {
		t.Fatal(err)
	}
	refused := [][]string{
		{"remove", "escape", "--confirm"}, {"remove", "alias", "--confirm"}, {"remove", "..", "--confirm"},
		{"remove", "../koboapps", "--confirm"}, {"remove", "_refresh", "--confirm"}, {"remove", "", "--confirm"},
		{"remove", "beta"}, // no --confirm
		{"remove", "ghost", "--confirm"},
	}
	for _, args := range refused {
		reply, code, _ := root.call(t, "en", args...)
		if code == exitOK || reply["error"] == nil {
			t.Errorf("%v must be refused, got %d %v", args, code, reply)
		}
	}
	for _, path := range []string{sentinel, filepath.Join(root.apps, "beta", "app.json"), root.host} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("%s was damaged by a refused remove: %v", path, err)
		}
	}
	reply, code, _ := root.call(t, "en", "remove", "beta", "--confirm")
	if code != exitOK || reply["ok"] != true || reply["text"] != "Removed Beta." {
		t.Fatalf("unexpected reply (exit %d): %v", code, reply)
	}
	if _, err := os.Stat(filepath.Join(root.apps, "beta")); !os.IsNotExist(err) {
		t.Error("beta folder still exists")
	}
	if menu, _ := os.ReadFile(root.menu); strings.Contains(string(menu), "Beta") {
		t.Errorf("menu still lists Beta: %s", menu)
	}
	if _, err := os.Stat(filepath.Join(root.apps, "alpha")); err != nil {
		t.Errorf("other apps must survive: %v", err)
	}
}

func TestDrawerRemoveBrokenApp(t *testing.T) {
	root := newDrawerRoot(t)
	installFixtureApps(t, root)
	if _, code, raw := root.call(t, "en", "remove", "broken", "--confirm"); code != exitOK {
		t.Fatalf("a broken app must be removable: %s", raw)
	}
}

func TestDrawerUsageErrors(t *testing.T) {
	var stdout bytes.Buffer
	for _, args := range [][]string{{"drawer"}, {"drawer", "bogus"}, {"drawer", "model", "--nope"}} {
		stdout.Reset()
		if code := run(args, &stdout, &bytes.Buffer{}); code == exitOK || !strings.Contains(stdout.String(), `"error"`) {
			t.Errorf("%v: want JSON error and non-zero exit, got %d %s", args, code, stdout.String())
		}
	}
}
