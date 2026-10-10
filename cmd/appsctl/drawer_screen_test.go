package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"koboapps/internal/drawer"
)

// localsendState is what a running LocalSend state script would print.
const localsendState = `{"running":true,"status":"Receiving","pin":"4821","device":"Kobo Libra 2","ip":"192.168.1.23",` +
	`"summary":"Books 2 · Rejected 1","received":[["book-one.epub","Books · 1.2 MB"],["A Very Long Title of a Book That Needs Eliding.epub","Books · 2.1 MB"]]}`

// installLocalsend copies the shipped LocalSend app, with its shipped state action, but swaps
// bin/state.sh for a script printing localsendState: the real one sources the device's
// /mnt/onboard paths. The golden files show bindings filled from state and locale keys resolved.
func installLocalsend(t *testing.T, root drawerRoot) {
	t.Helper()
	source := filepath.Join("..", "..", "apps", "localsend")
	target := filepath.Join(root.apps, "localsend")
	for _, relative := range []string{"app.json", "ui.json", "locales/en.json", "locales/vi.json", "bin/start.sh", "bin/stop.sh"} {
		data, err := os.ReadFile(filepath.Join(source, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		writeExecutable(t, filepath.Join(target, filepath.FromSlash(relative)), data)
	}
	writeExecutable(t, filepath.Join(target, "bin", "state.sh"), []byte("#!/bin/sh\necho '"+localsendState+"'\n"))
}

func writeExecutable(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o755); err != nil {
		t.Fatal(err)
	}
}

// generousStatusBudget keeps the real state script inside its budget under -race.
func generousStatusBudget(t *testing.T) {
	drawerHooks = func(config *drawer.Config) { config.StatusTimeout = 10 * time.Second }
	t.Cleanup(func() { drawerHooks = func(*drawer.Config) {} })
}

func TestDrawerScreenGolden(t *testing.T) {
	generousStatusBudget(t)
	for _, locale := range []string{"en", "vi"} {
		t.Run(locale, func(t *testing.T) {
			root := newDrawerRoot(t)
			installLocalsend(t, root)
			_, code, raw := root.call(t, locale, "screen", "localsend")
			if code != exitOK {
				t.Fatalf("exit %d: %s", code, raw)
			}
			assertGolden(t, "screen_localsend_"+locale+".json", raw)
		})
	}
}

func TestDrawerScreenErrors(t *testing.T) {
	root := newDrawerRoot(t)
	installFixtureApps(t, root)
	installLocalsend(t, root)
	for name, args := range map[string][]string{
		"no ui.json": {"alpha"}, "unknown screen": {"localsend", "--screen", "nope"}, "unknown app": {"ghost"},
		"missing argument": {}, "host item": {"_refresh"},
	} {
		t.Run(name, func(t *testing.T) {
			reply, code, _ := root.call(t, "en", append([]string{"screen"}, args...)...)
			if code == exitOK || reply["error"] == nil {
				t.Errorf("want an error document and non-zero exit, got %d %v", code, reply)
			}
		})
	}
}

func TestDrawerModelFlagsScreens(t *testing.T) {
	root := newDrawerRoot(t)
	installFixtureApps(t, root)
	installLocalsend(t, root)
	_, _, raw := root.call(t, "en", "model")
	if !strings.Contains(raw, `"id":"localsend","name":"LocalSend","status":"","badge":false,"running":false,"screen":true`) {
		t.Errorf("localsend must be marked as having a screen: %s", raw)
	}
	if strings.Contains(raw, `"id":"alpha","name":"Alpha","status":"Receiving 4821","badge":false,"running":true,"screen"`) {
		t.Errorf("alpha has no ui.json: %s", raw)
	}
}

func TestDrawerEvent(t *testing.T) {
	root := newDrawerRoot(t)
	writeScriptApp(t, root.apps, `{"schema":1,"id":"echo","name":"Echo","version":"1.0.0",
 "actions":[{"id":"mode","label":"Mode","run":"bin/mode.sh","output":"show"}]}`,
		map[string]string{"mode.sh": `printf 'value=[%s]' "${NICKELDRAWER_VALUE-unset}" >"$(dirname "$0")/seen.txt"; echo "prose that must not reach the screen"`})
	cases := map[string]struct {
		args []string
		want string
	}{
		"with value":  {[]string{"echo", "mode", "--value", "fit"}, "value=[fit]"},
		"empty value": {[]string{"echo", "mode", "--value", ""}, "value=[]"},
		"no value":    {[]string{"echo", "mode"}, "value=[unset]"},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			reply, code, raw := root.call(t, "en", append([]string{"event"}, test.args...)...)
			seen, _ := os.ReadFile(filepath.Join(root.apps, "echo", "bin", "seen.txt"))
			if code != exitOK || reply["ok"] != true || reply["text"] != "" || string(seen) != test.want {
				t.Errorf("exit %d: %s (script saw %q)", code, raw, seen)
			}
		})
	}
	for name, args := range map[string][]string{"unknown action": {"echo", "nope"}, "missing action": {"echo"}} {
		t.Run(name, func(t *testing.T) {
			if reply, code, _ := root.call(t, "en", append([]string{"event"}, args...)...); code == exitOK || reply["error"] == nil {
				t.Errorf("want an error, got %d %v", code, reply)
			}
		})
	}
}

func TestDrawerScreenValueAndStale(t *testing.T) {
	root := newDrawerRoot(t)
	dir := writeScriptApp(t, root.apps, `{"schema":1,"id":"echo","name":"Echo","version":"1.0.0",
 "actions":[{"id":"state","label":"State","run":"bin/state.sh","output":"show","menu":false}]}`,
		map[string]string{"state.sh": `echo "{\"line\":\"v=${NICKELDRAWER_VALUE-unset} d=$NICKELDRAWER\"}"`})
	ui := `{"kit":1,"start":"main","vars":{"line":"","selected":""},"screens":{"main":{"title":"T","body":[{"text":"$line"}]}}}`
	if err := os.WriteFile(filepath.Join(dir, "ui.json"), []byte(ui), 0o644); err != nil {
		t.Fatal(err)
	}
	reply, code, raw := root.call(t, "en", "screen", "echo", "--value", "0007")
	body, _ := reply["body"].([]any)
	if code != exitOK || reply["value"] != "0007" || len(body) != 1 || body[0].(map[string]any)["text"] != "v=0007 d=1" {
		t.Errorf("with --value: %d %s", code, raw)
	}
	reply, _, raw = root.call(t, "en", "screen", "echo")
	if _, has := reply["value"]; has || reply["stale"] != nil {
		t.Errorf("no --value, good state: %s", raw)
	}
	if err := os.WriteFile(filepath.Join(dir, "bin", "state.sh"), []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if reply, _, raw = root.call(t, "en", "screen", "echo"); reply["stale"] != true {
		t.Errorf("a failing state must be stale: %s", raw)
	}
}
