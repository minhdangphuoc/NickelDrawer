package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/text/unicode/norm"

	"koboapps/internal/drawer"
)

// todoDrawer runs the shipped To-do app through the real drawer protocol, with its scripts
// pointed at a temp data folder and its Go helper built for this computer.
type todoDrawer struct {
	t       *testing.T
	dataDir string
	config  drawer.Config
}

// These tests check app behaviour, not the 1 s state budget (internal/drawer tests that). A
// loaded CI runner can take longer, which would only make the reply stale and the test flaky.
const exampleStateBudget = 10 * time.Second

func newTodoDrawer(t *testing.T) *todoDrawer {
	t.Helper()
	root := newDrawerRoot(t)
	appDir := filepath.Join(root.apps, "todo")
	copyTree(t, filepath.Join("..", "..", "apps", "todo"), appDir)
	build := exec.Command("go", "build", "-o", filepath.Join(appDir, "bin", "todo"), "../todo")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("cannot build the todo helper: %v\n%s", err, output)
	}
	dataDir := t.TempDir()
	common, err := filepath.Abs(filepath.Join("..", "..", "host", "lib", "common.sh"))
	if err != nil {
		t.Fatal(err)
	}
	// The state script asks appsctl for the notice text; a stand-in keeps the test on this computer.
	fakeAppsctl := filepath.Join(t.TempDir(), "appsctl")
	if err := os.WriteFile(fakeAppsctl, []byte("#!/bin/sh\necho 'The saved list was damaged.'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	session := drawer.Config{
		StatusTimeout: exampleStateBudget,
		AppsDir:       root.apps, HostDir: root.host, Locale: "en", Bundle: fallbackBundle(),
		ScriptEnv: []string{"TODO_DATA_DIR=" + dataDir, "TODO_COMMON=" + common, "TODO_APPSCTL=" + fakeAppsctl},
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

func TestTodoWorksThroughTheDrawerProtocol(t *testing.T) {
	d := newTodoDrawer(t)

	if result := d.event("newitem", "buy milk"); result.Toast != "Task added" {
		t.Fatalf("add: %+v", result)
	}
	d.event("newitem", `say "hi" \ there`)
	if result := d.event("newitem", "   "); result.Toast != "Type the task first" {
		t.Fatalf("an empty task: %+v", result)
	}
	if result := d.event("newitem", "buy milk"); result.Toast != "Already on the list" {
		t.Fatalf("a repeated task: %+v", result)
	}
	if rows := d.rows(); len(rows) != 2 || rows[0][0] != "buy milk" || rows[1][0] != `say "hi" \ there` {
		t.Fatalf("rows = %v", rows)
	}

	if result := d.event("toggle", "buy milk"); result.Toast != "" {
		t.Fatalf("a tap on a row shows no toast: %+v", result)
	}
	if rows := d.rows(); rows[0][1] != "✓" || rows[1][1] != "" {
		t.Fatalf("after toggling: %v", rows)
	}

	if result := d.event("clear-done", ""); result.Toast != "Done tasks cleared" {
		t.Fatalf("clear: %+v", result)
	}
	if result := d.event("clear-done", ""); result.Toast != "No done tasks" {
		t.Fatalf("clear again: %+v", result)
	}
	if rows := d.rows(); len(rows) != 1 {
		t.Fatalf("rows = %v", rows)
	}
}

func TestTodoKeepsVietnameseNFCAndSurvivesANewSession(t *testing.T) {
	d := newTodoDrawer(t)
	d.event("newitem", norm.NFD.String("Đọc sách Việt Nam"))
	rows := d.rows()
	if len(rows) != 1 || rows[0][0] != "Đọc sách Việt Nam" {
		t.Fatalf("rows = %v", rows)
	}
	if len(rows[0][0]) != len(norm.NFC.String(rows[0][0])) {
		t.Fatal("the stored text must be NFC")
	}
	// A reboot only drops processes: a second drawer over the same data folder sees the same list.
	again := &todoDrawer{t: t, dataDir: d.dataDir, config: d.config}
	if rows := again.rows(); len(rows) != 1 {
		t.Fatalf("after restart: %v", rows)
	}
}

func TestTodoDamagedFileIsBackedUpAndTheScreenShowsANotice(t *testing.T) {
	d := newTodoDrawer(t)
	if err := os.WriteFile(filepath.Join(d.dataDir, "todo.json"), []byte("{broken"), 0o644); err != nil {
		t.Fatal(err)
	}
	reply, err := d.config.Screen("todo", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	var notice string
	for _, component := range reply.Body {
		if component.Type == "notice" {
			notice = component.Text
		}
	}
	if !strings.Contains(notice, "damaged") {
		t.Fatalf("notice = %q", notice)
	}
	if _, err := os.Stat(filepath.Join(d.dataDir, "todo.json.corrupt")); err != nil {
		t.Fatalf("the damaged file must be kept: %v", err)
	}
}

func TestTodoRowMenuDeletesTheLongPressedRow(t *testing.T) {
	d := newTodoDrawer(t)
	d.event("newitem", "one")
	d.event("newitem", "two")
	value := "two"
	result, err := d.config.RunEvent(drawer.Event{AppID: "todo", ActionID: "remove", ScreenID: "main", Value: &value, List: "items", Index: 1})
	if err != nil || result.Toast != "Task deleted" {
		t.Fatalf("remove: %+v %v", result, err)
	}
	if rows := d.rows(); len(rows) != 1 || rows[0][0] != "one" {
		t.Fatalf("rows = %v", rows)
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
		StatusTimeout: exampleStateBudget,
		AppsDir:       root.apps, HostDir: root.host, Locale: "en", Bundle: fallbackBundle(),
		ScriptEnv: []string{"IMAGEVIEW_IMAGES_DIR=" + images, "IMAGEVIEW_STATE_DIR=" + work, "IMAGEVIEW_MKCBZ=" + fakePacker, "IMAGEVIEW_COMMON=" + common},
	}

	result, err := config.RunEvent(drawer.Event{AppID: "imageview", ActionID: "make-albums", ScreenID: "main"})
	if err != nil || result.Toast != "Making albums..." {
		t.Fatalf("start: %+v %v", result, err)
	}
	// A fast job may finish before RunEvent reads the screen for its toast; the request then
	// arrives in the event reply instead of a later screen read. Either is fine, once.
	requests := result.Requests
	deadline := time.Now().Add(10 * time.Second)
	for len(requests) == 0 && time.Now().Before(deadline) {
		reply, err := config.Screen("imageview", "", nil)
		if err != nil {
			t.Fatal(err)
		}
		requests = reply.Requests
		if len(requests) == 0 {
			time.Sleep(100 * time.Millisecond)
		}
	}
	if len(requests) != 1 || requests[0].Name != "rescan" || requests[0].Mode != "quick" {
		t.Fatalf("the finished job must ask for a quick rescan: %+v", requests)
	}
	again, _ := config.Screen("imageview", "", nil)
	if len(again.Requests) != 0 {
		t.Fatal("the rescan request was handed over twice")
	}
	if _, err := os.Stat(filepath.Join(images, "Albums", "Images.cbz")); err != nil {
		t.Fatalf("album not made: %v", err)
	}
}

func TestUIKitIsHiddenAndItsScreensFollowTheStoredControlsAndPictures(t *testing.T) {
	root := newDrawerRoot(t)
	copyTree(t, filepath.Join("..", "..", "apps", "uikit"), filepath.Join(root.apps, "uikit"))
	work := t.TempDir()
	pictures := filepath.Join(work, "Pictures")
	if err := os.MkdirAll(pictures, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pictures, "one.png"), []byte("not really a picture"), 0o644); err != nil {
		t.Fatal(err)
	}
	common, _ := filepath.Abs(filepath.Join("..", "..", "host", "lib", "common.sh"))
	fakeAppsctl := filepath.Join(work, "appsctl")
	if err := os.WriteFile(fakeAppsctl, []byte("#!/bin/sh\necho 'Notice text'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	config := drawer.Config{
		StatusTimeout: exampleStateBudget,
		AppsDir:       root.apps, HostDir: root.host, Locale: "en", Bundle: fallbackBundle(),
		ScriptEnv: []string{
			"UIKIT_DATA_DIR=" + filepath.Join(work, "data"), "UIKIT_KOBO_DIR=" + filepath.Join(work, "kobo"),
			"UIKIT_PICTURES_DIR=" + pictures, "UIKIT_COMMON=" + common, "UIKIT_APPSCTL=" + fakeAppsctl,
		},
	}

	model, err := config.BuildModel()
	if err != nil {
		t.Fatal(err)
	}
	for _, section := range model.Sections {
		for _, item := range section.Items {
			if item.ID == "uikit" {
				t.Fatal("uikit is hidden by default and must not be listed")
			}
		}
	}

	value := "true"
	result, err := config.RunEvent(drawer.Event{AppID: "uikit", ActionID: "shownote", ScreenID: "more", Value: &value})
	if err != nil || result.Toast != "Note switch changed" {
		t.Fatalf("shownote: %+v %v", result, err)
	}
	reply, err := config.Screen("uikit", "more", nil)
	if err != nil || reply.Stale {
		t.Fatalf("more screen: %v stale=%v", err, reply.Stale)
	}
	var hasNote bool
	for _, component := range reply.Body {
		if component.Type == "notice" && component.Text == "Notice text" {
			hasNote = true
		}
	}
	if !hasNote {
		t.Fatal("the stored switch must show the notice")
	}
	gallery, err := config.Screen("uikit", "pictures", nil)
	if err != nil || len(gallery.Body) < 2 || gallery.Body[1].Type != "gallery" {
		t.Fatalf("gallery = %+v, %v", gallery.Body, err)
	}
	if tiles, isTiles := gallery.Body[1].Tiles.([]drawer.GalleryTile); !isTiles || len(tiles) != 1 {
		t.Fatalf("one PNG in Pictures must give one tile, got %v", gallery.Body[1].Tiles)
	}
}
