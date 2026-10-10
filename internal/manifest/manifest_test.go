package manifest

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// stageApp copies testdata/<name> to a temp folder of the same name and makes bin/* executable,
// because version control does not reliably keep the executable bit on fixtures.
func stageApp(t *testing.T, name string) string {
	t.Helper()
	source := filepath.Join("testdata", name)
	target := filepath.Join(t.TempDir(), name)
	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, _ := filepath.Rel(source, path)
		destination := filepath.Join(target, relative)
		if entry.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		mode := os.FileMode(0o644)
		if strings.HasSuffix(path, ".sh") && name != "bad-run-not-executable" {
			mode = 0o755
		}
		return os.WriteFile(destination, data, mode)
	})
	if err != nil {
		t.Fatalf("staging %s: %v", name, err)
	}
	return target
}

func TestLoadRejectsInvalidManifests(t *testing.T) {
	cases := []struct{ folder, wantProblem string }{
		{"bad-schema", "schema must be 1"},
		{"bad-id-mismatch", "must equal the folder name"},
		{"Bad_ID", "must match ^[a-z0-9]"},
		{"bad-name-colon", "must not contain ':'"},
		{"bad-name-long", "name must be 1-24"},
		{"bad-version", "MAJOR.MINOR.PATCH"},
		{"bad-no-actions", "actions must have 1-12"},
		{"bad-too-many-actions", "actions must have 1-12"},
		{"bad-dup-action-id", "duplicate action id"},
		{"bad-run-dotdot", "must be relative"},
		{"bad-run-absolute", "must be relative"},
		{"bad-run-colon", "must be relative"},
		{"bad-run-missing", "does not exist"},
		{"bad-run-not-executable", "not an executable"},
		{"bad-output", "must be \"show\" or \"none\""},
		{"bad-timeout-with-none", "only allowed with output"},
		{"bad-timeout-range", "timeout_ms 10000 must be 500-9999"},
		{"bad-rescan", "must be one of none, quick, full"},
		{"bad-label-colon", "must not contain ':'"},
		{"bad-json", "not valid JSON"},
		{"bad-hidden-type", "not valid JSON"},
		{"bad-storage-unknown", `unknown folder "secrets"`},
		{"bad-storage-unknown", `unknown folder "/etc"`},
		{"bad-requests-unknown", `unknown request "reboot"`},
		{"bad-requests-duplicate", "listed twice"},
		{"bad-min-host", "min_host"},
	}
	for _, c := range cases {
		t.Run(c.folder, func(t *testing.T) {
			_, _, err := Load(stageApp(t, c.folder))
			if err == nil {
				t.Fatal("expected an error, got none")
			}
			if !strings.Contains(err.Error(), c.wantProblem) {
				t.Fatalf("error %q does not mention %q", err, c.wantProblem)
			}
			var validation *ValidationError
			if !errors.As(err, &validation) {
				t.Fatalf("expected *ValidationError, got %T", err)
			}
		})
	}
}

func TestLoadReportsMissingManifest(t *testing.T) {
	folder := filepath.Join(t.TempDir(), "missing-manifest")
	if err := os.Mkdir(folder, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Load(folder); err == nil || !strings.Contains(err.Error(), "app.json") {
		t.Fatalf("expected an error naming app.json, got %v", err)
	}
}

func TestLoadAcceptsFullManifest(t *testing.T) {
	app, warnings, err := Load(stageApp(t, "valid-full"))
	if err != nil || len(warnings) != 0 {
		t.Fatalf("err=%v warnings=%v", err, warnings)
	}
	if len(app.Actions) != 2 || app.Actions[0].Timeout() != 9500 || !app.Actions[0].Wifi {
		t.Fatalf("unexpected parse: %+v", app)
	}
	if app.Actions[1].Timeout() != 0 || app.Actions[1].RescanMode() != RescanQuick {
		t.Fatalf("unexpected second action: %+v", app.Actions[1])
	}
}

func TestRequestsAndMinHostAreParsed(t *testing.T) {
	app, _, err := Load(stageApp(t, "valid-requests"))
	if err != nil {
		t.Fatal(err)
	}
	if !app.MayRequest(RequestRescan) || !app.MayRequest(RequestWifi) || app.MayRequest("reboot") {
		t.Fatalf("requests = %v", app.Requests)
	}
	for hostVersion, want := range map[string]bool{"1.1.9": true, "1.2.0": false, "2.0.0": false, "1.2.0-beta.1": true, "dev": false} {
		if got := app.NeedsNewerHost(hostVersion); got != want {
			t.Errorf("NeedsNewerHost(%q) = %v, want %v", hostVersion, got, want)
		}
	}
}

func TestLoadAppliesDefaults(t *testing.T) {
	app, _, err := Load(stageApp(t, "valid-defaults"))
	if err != nil {
		t.Fatal(err)
	}
	action := app.Actions[0]
	if !app.InMenu() || action.Timeout() != DefaultTimeout || action.RescanMode() != RescanNone || action.Wifi {
		t.Fatalf("defaults not applied: %+v", app)
	}
}

func TestHiddenFlagHidesTheAppFromEveryList(t *testing.T) {
	cases := []struct {
		folder       string
		wantHidden   bool
		wantInMenu   bool
		wantNoWarned bool
	}{
		{"valid-hidden", true, false, true},
		{"valid-hidden-false", false, true, true},
		{"valid-defaults", false, true, true},
	}
	for _, c := range cases {
		t.Run(c.folder, func(t *testing.T) {
			app, warnings, err := Load(stageApp(t, c.folder))
			if err != nil {
				t.Fatal(err)
			}
			if app.IsHidden() != c.wantHidden || app.InMenu() != c.wantInMenu || (len(warnings) == 0) != c.wantNoWarned {
				t.Fatalf("hidden=%v inMenu=%v warnings=%v", app.IsHidden(), app.InMenu(), warnings)
			}
		})
	}
}

func TestLoadWarnsAboutUnknownFields(t *testing.T) {
	_, warnings, err := Load(stageApp(t, "warn-unknown-fields"))
	if err != nil {
		t.Fatal(err)
	}
	want := []Warning{`unknown field "icon" ignored`, `actions[0]: unknown field "color" ignored`}
	if len(warnings) != len(want) || warnings[0] != want[0] || warnings[1] != want[1] {
		t.Fatalf("warnings = %v, want %v", warnings, want)
	}
}

func TestLoadAllSkipsHiddenAndFiles(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"valid-defaults", "bad-schema"} {
		if err := os.Rename(stageApp(t, name), filepath.Join(root, name)); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{".hidden", "_template"} {
		if err := os.Mkdir(filepath.Join(root, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	scan, err := LoadAll(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(scan.Apps) != 1 || len(scan.Failures) != 1 || scan.Failures[0].AppID != "bad-schema" {
		t.Fatalf("scan = %+v", scan)
	}
	if !strings.Contains(scan.Failures[0].Reason(), "schema must be 1") {
		t.Fatalf("reason = %q", scan.Failures[0].Reason())
	}
}

func TestLoadReadsStorageAndNetworkDeclarations(t *testing.T) {
	app, warnings, err := Load(stageApp(t, "valid-storage"))
	if err != nil || len(warnings) != 0 {
		t.Fatalf("err=%v warnings=%v", err, warnings)
	}
	if !app.Network || app.Storage == nil {
		t.Fatalf("network/storage not read: %+v", app)
	}
	if strings.Join(app.Storage.Reads, ",") != "pictures" || strings.Join(app.Storage.Writes, ",") != "inbox,books" {
		t.Errorf("storage = %+v", app.Storage)
	}
}

func TestActionMenuDefaultsToTrueAndAcceptsFalse(t *testing.T) {
	app, warnings, err := Load(stageApp(t, "valid-action-menu"))
	if err != nil || len(warnings) > 0 {
		t.Fatalf("err=%v warnings=%v", err, warnings)
	}
	if !app.Actions[0].InMenu() || app.Actions[1].InMenu() {
		t.Fatalf("InMenu = %v, %v; want true, false", app.Actions[0].InMenu(), app.Actions[1].InMenu())
	}
}
