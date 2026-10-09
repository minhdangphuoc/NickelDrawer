package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"koboapps/internal/uikit"
)

func runNewCommand(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := runNew(args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func TestNewCreatesValidApp(t *testing.T) {
	parent := t.TempDir()
	code, stdout, stderr := runNewCommand(t, "hello-world", "--dir", parent)
	if code != exitOK {
		t.Fatalf("exit %d: %s %s", code, stdout, stderr)
	}
	appDir := filepath.Join(parent, "hello-world")
	if reportDirs([]string{appDir}, &bytes.Buffer{}) != exitOK {
		t.Fatal("generated app does not validate")
	}
	if !strings.Contains(stdout, "Hello World") && !strings.Contains(readFile(t, filepath.Join(appDir, "app.json")), "Hello World") {
		t.Error("name was not derived from the id")
	}
	for _, unwanted := range []string{"ui.json", "bin/state.sh"} {
		if _, err := os.Stat(filepath.Join(appDir, unwanted)); err == nil {
			t.Errorf("%s created without --with-ui", unwanted)
		}
	}
	info, err := os.Stat(filepath.Join(appDir, "bin", "run.sh"))
	if err != nil || info.Mode().Perm()&0o100 == 0 {
		t.Errorf("bin/run.sh must be executable: %v", err)
	}
}

func TestNewWithUIProducesValidUIJSON(t *testing.T) {
	parent := t.TempDir()
	code, _, stderr := runNewCommand(t, "my-app", "--with-ui", "--name", "My App", "--dir", parent)
	if code != exitOK {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	appDir := filepath.Join(parent, "my-app")
	if !uikit.HasUI(appDir) {
		t.Fatal("ui.json missing")
	}
	_, report, err := uikit.LoadDir(appDir, []string{"run", "state"})
	if err != nil || len(report.Errors) > 0 {
		t.Fatalf("ui.json invalid: %v %v", err, report.Errors)
	}
}

func TestNewRefusesExistingFolder(t *testing.T) {
	parent := t.TempDir()
	existing := filepath.Join(parent, "taken")
	if err := os.Mkdir(existing, 0o755); err != nil {
		t.Fatal(err)
	}
	code, _, stderr := runNewCommand(t, "taken", "--dir", parent)
	if code != exitFailure || !strings.Contains(stderr, "already exists") {
		t.Errorf("want refusal, got %d %q", code, stderr)
	}
}

func TestNewRejectsBadIDAndName(t *testing.T) {
	parent := t.TempDir()
	for _, args := range [][]string{
		{"Bad_ID", "--dir", parent},
		{"x", "--dir", parent},
		{"ok-id", "--name", "Has: colon", "--dir", parent},
	} {
		if code, _, _ := runNewCommand(t, args...); code != exitFailure {
			t.Errorf("%v: want failure, got %d", args, code)
		}
	}
	if entries, _ := os.ReadDir(parent); len(entries) != 0 {
		t.Errorf("rejected input left files behind: %v", entries)
	}
}

func readFile(t *testing.T, filePath string) string {
	t.Helper()
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
