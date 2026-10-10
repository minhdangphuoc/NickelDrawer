package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/text/unicode/norm"
)

const sampleSentence = "Tôi yêu tiếng Việt vì nó có nhiều dấu thanh và những âm điệu đẹp đẽ. "

// writeEpub writes a minimal EPUB with the given dc:language through the real zip layout.
func writeEpub(t *testing.T, path, language string) {
	t.Helper()
	var buffer bytes.Buffer
	writer := newZip(&buffer)
	writer.add(t, "mimetype", "application/epub+zip")
	writer.add(t, "META-INF/container.xml", `<container><rootfiles><rootfile full-path="c.opf"/></rootfiles></container>`)
	writer.add(t, "c.opf", `<package><metadata xmlns:dc="x"><dc:title>Sách</dc:title><dc:language>`+language+`</dc:language></metadata><manifest><item id="a" href="a.xhtml" media-type="application/xhtml+xml"/></manifest><spine><itemref idref="a"/></spine></package>`)
	writer.add(t, "a.xhtml", `<html><body><p>`+strings.Repeat(sampleSentence, 8)+`</p></body></html>`)
	writer.close(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buffer.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func setup(t *testing.T) (root, appDir string) {
	t.Helper()
	root, appDir = t.TempDir(), t.TempDir()
	locales, err := filepath.Glob("../../apps/library/locales/*.json")
	if err != nil || len(locales) == 0 {
		t.Fatal("no locale files found")
	}
	if err := os.MkdirAll(filepath.Join(appDir, "locales"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, file := range locales {
		content, _ := os.ReadFile(file)
		if err := os.WriteFile(filepath.Join(appDir, "locales", filepath.Base(file)), content, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeEpub(t, filepath.Join(root, "Books", "a.epub"), "en")
	return root, appDir
}

func run(t *testing.T, root, appDir string, args ...string) (string, int) {
	t.Helper()
	var out bytes.Buffer
	full := append([]string{args[0], "--root", root, "--app-dir", appDir, "--locale", "en"}, args[1:]...)
	code := execute(full, &out)
	return out.String(), code
}

func TestScanFixUndoAndStateThroughTheCommandLine(t *testing.T) {
	root, appDir := setup(t)

	if out, code := run(t, root, appDir, "scan"); code != 0 || !strings.Contains(out, "Books: 1. Wrong language: 1. Not NFC: 0. Duplicate groups: 0.") {
		t.Fatalf("scan (%d): %s", code, out)
	}
	if out, code := run(t, root, appDir, "fix-language", "--dry-run"); code != 0 || !strings.Contains(out, "Books/a.epub") {
		t.Fatalf("dry run (%d): %s", code, out)
	}
	out, code := run(t, root, appDir, "fix-language")
	if code != 0 || !strings.Contains(out, "Fixed 1 of 1") || !strings.Contains(out, "Rescan the library") {
		t.Fatalf("fix (%d): %s", code, out)
	}

	var state map[string]any
	out, _ = run(t, root, appDir, "state")
	if err := json.Unmarshal([]byte(out), &state); err != nil || state["wrong"] != "0" || state["can_undo"] != true || state["running"] != false {
		t.Fatalf("state: %v %s", err, out)
	}
	if out, code := run(t, root, appDir, "undo"); code != 0 || !strings.Contains(out, "Restored 1") {
		t.Fatalf("undo (%d): %s", code, out)
	}
}

func TestRemoveDuplicateNeedsConfirmAndStaysInsideLibrary(t *testing.T) {
	root, appDir := setup(t)
	writeEpub(t, filepath.Join(root, "Books", "b.epub"), "en")
	run(t, root, appDir, "scan")

	if out, code := run(t, root, appDir, "dedupe-list"); code != 0 || !strings.Contains(out, "Group 1") {
		t.Fatalf("dedupe-list (%d): %s", code, out)
	}
	if _, code := run(t, root, appDir, "remove-duplicate", "Books/b.epub"); code == 0 {
		t.Error("removal without --confirm must fail")
	}
	if _, code := run(t, root, appDir, "remove-duplicate", "--confirm", "../../etc/passwd"); code == 0 {
		t.Error("a path outside the library must fail")
	}
	if out, code := run(t, root, appDir, "pick", "Books/b.epub"); code != 0 {
		t.Fatalf("pick (%d): %s", code, out)
	}
	if out, code := run(t, root, appDir, "remove-duplicate", "--selected", "--confirm"); code != 0 || !strings.Contains(out, "Moved to trash") {
		t.Fatalf("remove (%d): %s", code, out)
	}
	if _, err := os.Stat(filepath.Join(root, "Books", "b.epub")); !os.IsNotExist(err) {
		t.Error("b.epub should be in the trash")
	}
}

func TestUnknownCommandAndBusyLock(t *testing.T) {
	root, appDir := setup(t)
	if _, code := run(t, root, appDir, "frobnicate"); code != exitUsage {
		t.Errorf("unknown command exit = %d", code)
	}
	if err := os.MkdirAll(filepath.Join(appDir, "data"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "data", "lock"), []byte(strconvPid()), 0o644); err != nil {
		t.Fatal(err)
	}
	out, code := run(t, root, appDir, "scan")
	if code == 0 || !strings.Contains(out, "Still working") {
		t.Errorf("a second run must be refused (%d): %s", code, out)
	}
}

func TestNormalizeNFCAndCancelThroughTheCommandLine(t *testing.T) {
	root, appDir := setup(t)
	writeEpub(t, filepath.Join(root, "Books", norm.NFD.String("Truyện.epub")), "vi")
	run(t, root, appDir, "scan")

	if out, code := run(t, root, appDir, "normalize-nfc", "--dry-run"); code != 0 || !strings.Contains(out, "Would fix 1 item(s)") {
		t.Fatalf("dry run (%d): %s", code, out)
	}
	if out, code := run(t, root, appDir, "normalize-nfc"); code != 0 || !strings.Contains(out, "Fixed 1 of 1") {
		t.Fatalf("normalize (%d): %s", code, out)
	}
	if out, code := run(t, root, appDir, "cancel"); code != 0 || !strings.Contains(out, "Nothing is running") {
		t.Fatalf("cancel with nothing running (%d): %s", code, out)
	}
	var state map[string]any
	out, _ := run(t, root, appDir, "state")
	if err := json.Unmarshal([]byte(out), &state); err != nil || state["not_nfc"] != "0" || state["can_undo"] != true {
		t.Fatalf("state: %v %s", err, out)
	}
	requests, _ := state["requests"].([]any)
	if len(requests) != 1 {
		t.Fatalf("the state must hand over the rescan request once: %v", state["requests"])
	}
	out, _ = run(t, root, appDir, "state")
	state = nil
	_ = json.Unmarshal([]byte(out), &state)
	if _, again := state["requests"]; again {
		t.Fatal("the request was handed over twice")
	}
}
