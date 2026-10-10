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
)

// updatesConfig lays out <root>/.adds/koboapps with the real updates screen and a drawer
// Config pointing at it, so the test covers the shipped ui.json and locales.
func updatesConfig(t *testing.T, version string) (Config, string) {
	t.Helper()
	root := t.TempDir()
	hostDir := filepath.Join(root, ".adds", "koboapps")
	source := filepath.Join("..", "..", "host", "screens")
	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() {
			return walkErr
		}
		relative, _ := filepath.Rel(source, path)
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		target := filepath.Join(hostDir, "screens", relative)
		if mkdirErr := os.MkdirAll(filepath.Dir(target), 0o755); mkdirErr != nil {
			return mkdirErr
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := i18n.LoadBundle(filepath.Join("..", "..", "host", "locales"), "en")
	if err != nil {
		t.Fatal(err)
	}
	config := Config{AppsDir: filepath.Join(root, ".adds", "apps"), HostDir: hostDir, CacheDir: t.TempDir(),
		Locale: "en", Version: version, Bundle: bundle, StatusTimeout: 50 * time.Millisecond}
	writeFileIn(t, hostDir, "data/update.json", `{"repo":"o/r"}`)
	if err := os.MkdirAll(config.AppsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	return config, root
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

func TestUpdatesItemShowsOneUpdateOnceACheckFoundOne(t *testing.T) {
	config, _ := updatesConfig(t, "1.0.0")
	item := systemItem(t, config, HostUpdatesID)
	if item.Badge || item.Status != "" || !item.Screen {
		t.Fatalf("before any check: %+v", item)
	}
	writeFileIn(t, config.HostDir, "data/update/available.json",
		`{"available":true,"version":"1.1.0","current":"1.0.0","notes":"Faster scan.","checked":"2026-10-10T10:00:00Z"}`)
	item = systemItem(t, config, HostUpdatesID)
	if !item.Badge || item.Status != "1 update" {
		t.Fatalf("after a check: %+v", item)
	}
	model, _ := config.BuildModel()
	if !model.UpdateAvailable {
		t.Fatal("the model must say an update is waiting")
	}
	config.Version = "1.1.0" // the update was installed since the check
	if item = systemItem(t, config, HostUpdatesID); item.Badge || item.Status != "" {
		t.Fatalf("a stale check must not show: %+v", item)
	}
}

func TestUpdatesScreenShowsVersionsNotesAndEnablesTheButton(t *testing.T) {
	config, _ := updatesConfig(t, "1.0.0")
	writeFileIn(t, config.HostDir, "data/update/available.json",
		`{"available":true,"version":"1.1.0","current":"1.0.0","notes":"Faster scan.","checked":"2026-10-10T10:00:00Z"}`)
	reply, err := config.Screen(HostUpdatesID, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if reply.Title != "Updates" || !reply.Live {
		t.Fatalf("title=%q live=%v", reply.Title, reply.Live)
	}
	rows := findComponent(t, reply.Body, "kv", 0).Rows.([][2]string)
	if rows[0][1] != "1.0.0" || rows[1][1] != "1.1.0" {
		t.Fatalf("rows = %v", rows)
	}
	if notice := findComponent(t, reply.Body, "notice", 0); notice.Text != "Faster scan." {
		t.Fatalf("notice = %+v", notice)
	}
	apply := findComponent(t, reply.Footer, "button", 0)
	if apply.Action != "apply" || apply.Enabled == nil || !*apply.Enabled || apply.Style != "primary" {
		t.Fatalf("apply button = %+v", apply)
	}
	if reply.Toasts["apply_started"] == "" {
		t.Fatal("the toasts the scripts print must be declared")
	}
}

func TestUpdatesScreenDisablesApplyWithoutAnUpdateOrWhileBusy(t *testing.T) {
	config, _ := updatesConfig(t, "1.0.0")
	reply, _ := config.Screen(HostUpdatesID, "", nil)
	apply := findComponent(t, reply.Footer, "button", 0)
	if *apply.Enabled {
		t.Fatal("nothing to apply before a check")
	}
	if rows := findComponent(t, reply.Body, "kv", 0).Rows.([][2]string); rows[1][1] != "Not checked yet" {
		t.Fatalf("rows = %v", rows)
	}
	writeFileIn(t, config.HostDir, "data/update/available.json",
		`{"available":true,"version":"1.1.0","current":"1.0.0","checked":"2026-10-10T10:00:00Z"}`)
	status, _ := json.Marshal(map[string]any{"state": "running", "pid": os.Getpid(), "time": "2026-10-10T10:00:00Z",
		"report": map[string]any{"key": "update.running"}})
	writeFileIn(t, config.HostDir, "data/update/status.json", string(status))
	reply, _ = config.Screen(HostUpdatesID, "", nil)
	apply = findComponent(t, reply.Footer, "button", 0)
	check := findComponent(t, reply.Footer, "button", 1)
	if *apply.Enabled || *check.Enabled {
		t.Fatalf("both buttons wait while a run is active: apply=%v check=%v", *apply.Enabled, *check.Enabled)
	}
	if text := findComponent(t, reply.Body, "text", 0); !strings.Contains(text.Text, "Updating") {
		t.Fatalf("message = %q", text.Text)
	}
}

func TestUpdatesScreenEventsRunTheHostScripts(t *testing.T) {
	config, _ := updatesConfig(t, "1.0.0")
	var ran []string
	config.RunScript = func(_ context.Context, script string) ScriptOutput {
		ran = append(ran, filepath.Base(script))
		return ScriptOutput{Stdout: `{"toast":"apply_started"}`}
	}
	config.HasIP = func() bool { return true }
	result, err := config.RunEvent(Event{AppID: HostUpdatesID, ActionID: "apply", ScreenID: "main"})
	if err != nil {
		t.Fatal(err)
	}
	if len(ran) != 1 || ran[0] != "update-apply.sh" || result.Toast != "Updating. Keep Wi-Fi on." {
		t.Fatalf("ran=%v result=%+v", ran, result)
	}
	config.HasIP = func() bool { return false }
	if result, _ = config.RunEvent(Event{AppID: HostUpdatesID, ActionID: "check", ScreenID: "main"}); !result.NeedsWifi {
		t.Fatalf("a network action without Wi-Fi must ask the mod to connect: %+v", result)
	}
}
