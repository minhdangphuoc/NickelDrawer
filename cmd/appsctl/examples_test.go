package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"koboapps/internal/drawer"
)

// todoDrawer runs the shipped to-do example through the real drawer protocol, with its scripts
// pointed at a temp data folder.
type todoDrawer struct {
	t       *testing.T
	dataDir string
	config  drawer.Config
}

func newTodoDrawer(t *testing.T) *todoDrawer {
	t.Helper()
	root := newDrawerRoot(t)
	appDir := filepath.Join(root.apps, "todo")
	copyTree(t, filepath.Join("..", "..", "examples", "todo"), appDir)
	dataDir := t.TempDir()
	common, err := filepath.Abs(filepath.Join("..", "..", "host", "lib", "common.sh"))
	if err != nil {
		t.Fatal(err)
	}
	session := drawer.Config{
		AppsDir: root.apps, HostDir: root.host, Locale: "en", Bundle: fallbackBundle(),
		ScriptEnv: []string{"TODO_DATA_DIR=" + dataDir, "TODO_COMMON=" + common, "TODO_INBOXES=" + filepath.Join(dataDir, "inbox.txt")},
	}
	return &todoDrawer{t: t, dataDir: dataDir, config: session}
}

func copyTree(t *testing.T, from, to string) {
	t.Helper()
	err := filepath.Walk(from, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, _ := filepath.Rel(from, path)
		target := filepath.Join(to, relative)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode().Perm())
	})
	if err != nil {
		t.Fatal(err)
	}
}

func (d *todoDrawer) event(action, value string) drawer.RunResult {
	d.t.Helper()
	result, err := d.config.RunEvent(drawer.Event{AppID: "todo", ActionID: action, ScreenID: "main", Value: &value})
	if err != nil {
		d.t.Fatal(err)
	}
	return result
}

func (d *todoDrawer) rows() [][2]string {
	d.t.Helper()
	reply, err := d.config.Screen("todo", "", nil)
	if err != nil {
		d.t.Fatal(err)
	}
	for _, component := range reply.Body {
		if component.Type == "list" {
			return *component.Items
		}
	}
	d.t.Fatal("no list on the screen")
	return nil
}

func TestTodoExampleWorksThroughTheDrawerProtocol(t *testing.T) {
	d := newTodoDrawer(t)

	if result := d.event("newitem", "buy milk"); result.Toast != "Added" {
		t.Fatalf("add: %+v", result)
	}
	d.event("newitem", `say "hi" \ there`)
	if result := d.event("newitem", "   "); result.Toast != "Type the item first" {
		t.Fatalf("an empty item: %+v", result)
	}
	if rows := d.rows(); len(rows) != 2 || rows[0][0] != "buy milk" || rows[1][0] != `say "hi" \ there` {
		t.Fatalf("rows = %v", rows)
	}

	d.event("toggle", "buy milk")
	if rows := d.rows(); rows[0][1] != "✓" || rows[1][1] != "" {
		t.Fatalf("after toggling: %v", rows)
	}
	d.event("hidedone", "true")
	if rows := d.rows(); len(rows) != 1 || rows[0][0] != `say "hi" \ there` {
		t.Fatalf("done items must hide: %v", rows)
	}
	d.event("hidedone", "false")

	if result := d.event("clear-done", ""); result.Toast != "Done items removed" {
		t.Fatalf("clear: %+v", result)
	}
	if result := d.event("clear-done", ""); result.Toast != "No done items" {
		t.Fatalf("clear again: %+v", result)
	}
	if rows := d.rows(); len(rows) != 1 {
		t.Fatalf("rows = %v", rows)
	}
}

func TestTodoRowMenuRemovesTheLongPressedRow(t *testing.T) {
	d := newTodoDrawer(t)
	d.event("newitem", "one")
	d.event("newitem", "two")
	value := "two"
	result, err := d.config.RunEvent(drawer.Event{AppID: "todo", ActionID: "remove", ScreenID: "main", Value: &value, List: "items", Index: 1})
	if err != nil || result.Toast != "Item removed" {
		t.Fatalf("remove: %+v %v", result, err)
	}
	if rows := d.rows(); len(rows) != 1 || rows[0][0] != "one" {
		t.Fatalf("rows = %v", rows)
	}
}

func TestTodoAddFromFileMovesTheInboxIntoTheList(t *testing.T) {
	d := newTodoDrawer(t)
	if result := d.event("add-from-file", ""); result.Toast != "The inbox file is empty" {
		t.Fatalf("empty inbox: %+v", result)
	}
	if err := os.WriteFile(filepath.Join(d.dataDir, "inbox.txt"), []byte("a\r\n\r\n b \n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if result := d.event("add-from-file", ""); result.Toast != "Added" {
		t.Fatalf("add: %+v", result)
	}
	rows := d.rows()
	encoded, _ := json.Marshal(rows)
	if len(rows) != 2 || !strings.Contains(string(encoded), `"a"`) || !strings.Contains(string(encoded), `"b"`) {
		t.Fatalf("rows = %s", encoded)
	}
	if info, err := os.Stat(filepath.Join(d.dataDir, "inbox.txt")); err != nil || info.Size() != 0 {
		t.Fatal("the inbox must be emptied, not deleted")
	}
}

func TestImageviewExampleBuildsAlbumsAndAsksForOneRescan(t *testing.T) {
	root := newDrawerRoot(t)
	copyTree(t, filepath.Join("..", "..", "examples", "imageview"), filepath.Join(root.apps, "imageview"))
	work := t.TempDir()
	images := filepath.Join(work, "Images")
	if err := os.MkdirAll(images, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(images, "one.jpg"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	fakePacker := filepath.Join(work, "mkcbz")
	if err := os.WriteFile(fakePacker, []byte("#!/bin/sh\nmkdir -p \"$(dirname \"$2\")\" && : > \"$2\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	common, _ := filepath.Abs(filepath.Join("..", "..", "host", "lib", "common.sh"))
	config := drawer.Config{
		AppsDir: root.apps, HostDir: root.host, Locale: "en", Bundle: fallbackBundle(),
		ScriptEnv: []string{"IMAGEVIEW_IMAGES_DIR=" + images, "IMAGEVIEW_STATE_DIR=" + work, "IMAGEVIEW_MKCBZ=" + fakePacker, "IMAGEVIEW_COMMON=" + common},
	}

	result, err := config.RunEvent(drawer.Event{AppID: "imageview", ActionID: "make-albums", ScreenID: "main"})
	if err != nil || result.Toast != "Making albums..." {
		t.Fatalf("start: %+v %v", result, err)
	}
	deadline := time.Now().Add(10 * time.Second)
	var reply drawer.ScreenReply
	for time.Now().Before(deadline) {
		if reply, err = config.Screen("imageview", "", nil); err != nil {
			t.Fatal(err)
		}
		if len(reply.Requests) > 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if len(reply.Requests) != 1 || reply.Requests[0].Name != "rescan" || reply.Requests[0].Mode != "quick" {
		t.Fatalf("the finished job must ask for a quick rescan: %+v", reply.Requests)
	}
	again, _ := config.Screen("imageview", "", nil)
	if len(again.Requests) != 0 {
		t.Fatal("the rescan request was handed over twice")
	}
	if _, err := os.Stat(filepath.Join(images, "Albums", "Images.cbz")); err != nil {
		t.Fatalf("album not made: %v", err)
	}
}
