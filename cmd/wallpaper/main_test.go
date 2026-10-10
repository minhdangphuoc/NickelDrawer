package main

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"koboapps/internal/storage"
)

// deviceRoot builds a fake /mnt/onboard with a Libra 2 version file and one picture.
func deviceRoot(t *testing.T) (root, appDir, versionFile string) {
	t.Helper()
	root = t.TempDir()
	t.Cleanup(storage.SetRoot(root))
	versionFile = filepath.Join(root, ".kobo", "version")
	write(t, versionFile, []byte("N000,4.1.15,4.38.23828,4.1.15,4.1.15,00000000-0000-0000-0000-000000000388\n"))
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, image.NewRGBA(image.Rect(0, 0, 90, 60))); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(root, "Pictures", "a.png"), buffer.Bytes())
	appDir = filepath.Join(root, "app")
	copyLocales(t, appDir)
	return root, appDir, versionFile
}

func write(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
}

func run(t *testing.T, args ...string) (int, string) {
	t.Helper()
	var out bytes.Buffer
	status := execute(args, &out)
	return status, out.String()
}

func TestImportProducesFullSizeScreensaverPicture(t *testing.T) {
	root, appDir, versionFile := deviceRoot(t)
	common := []string{"--app-dir", appDir, "--locale", "en", "--version-file", versionFile}
	status, output := run(t, append([]string{"import"}, common...)...)
	if status != 0 {
		t.Fatalf("status %d: %s", status, output)
	}
	file, err := os.Open(filepath.Join(root, ".kobo", "screensaver", "nd-wallpaper-0001.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	config, err := png.DecodeConfig(file)
	if err != nil || config.Width != 1264 || config.Height != 1680 {
		t.Fatalf("screensaver picture = %+v, %v", config, err)
	}
	if status, output = run(t, append([]string{"status"}, common...)...); status != 0 || !strings.Contains(output, "Pictures: 1") {
		t.Errorf("status = %d %q", status, output)
	}
	if status, output = run(t, append([]string{"mode", "toggle"}, common...)...); status != 1 && status != 2 {
		// flags must come before positionals, so this order is a usage error by design
		t.Errorf("flags after positionals must not be accepted, got %d %q", status, output)
	}
	if status, output = run(t, "mode", "--app-dir", appDir, "--locale", "en", "--version-file", versionFile, "toggle"); status != 0 || !strings.Contains(output, "Fit") {
		t.Errorf("toggle = %d %q", status, output)
	}
	if status, _ = run(t, "off", "--app-dir", appDir, "--locale", "en"); status != 0 {
		t.Errorf("off status %d", status)
	}
	if _, err := os.Stat(filepath.Join(root, ".kobo", "screensaver", "nd-wallpaper-0001.png")); err == nil {
		t.Error("off must remove our picture")
	}
}

func TestUnknownDeviceIsRefusedWithMessage(t *testing.T) {
	_, appDir, versionFile := deviceRoot(t)
	write(t, versionFile, []byte("N000,4.1.15,4.38.23828,4.1.15,4.1.15,00000000-0000-0000-0000-000000000999\n"))
	status, output := run(t, "import", "--app-dir", appDir, "--locale", "en", "--version-file", versionFile)
	if status == 0 || !strings.Contains(output, "msg.err_device_unknown") && !strings.Contains(output, "not known") {
		t.Errorf("status %d output %q", status, output)
	}
}

func copyLocales(t *testing.T, appDir string) {
	t.Helper()
	files, err := filepath.Glob("../../apps/wallpaper/locales/*.json")
	if err != nil || len(files) == 0 {
		t.Fatalf("no locale files: %v", err)
	}
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		write(t, filepath.Join(appDir, "locales", filepath.Base(file)), content)
	}
}

type stateReply struct {
	Count      int           `json:"count"`
	CurrentID  string        `json:"current_id"`
	Mode       string        `json:"mode"`
	Rotation   string        `json:"rotation"`
	LastImport string        `json:"last_import"`
	Busy       bool          `json:"busy"`
	Status     string        `json:"status"`
	Summary    string        `json:"summary"`
	Pictures   []pictureItem `json:"pictures"`
	Pending    int           `json:"pending"`
	ImportStep string        `json:"import_step"`

	SleepScreenReady bool   `json:"sleep_screen_ready"`
	SleepScreenHint  string `json:"sleep_screen_hint"`
	SleepScreenLine  string `json:"sleep_screen_line"`

	SleepScreenNotice string `json:"sleep_screen_notice"`
}

type pictureItem struct {
	ID    string `json:"id"`
	Thumb string `json:"thumb"`
	Image string `json:"image"`
	Label string `json:"label"`
	Info  string `json:"info"`
	Badge string `json:"badge"`
}

const enSleepHint = "Turn on in Settings → Energy saving and privacy: “Show current read” and “Show book cover full screen”."

func readState(t *testing.T, appDir, lastFile, pidFile string) stateReply {
	t.Helper()
	status, output := run(t, "state", "--app-dir", appDir, "--locale", "en", "--last-file", lastFile, "--pid-file", pidFile, "--conf", filepath.Join(appDir, "Kobo eReader.conf"))
	if status != 0 {
		t.Fatalf("state exit %d: %s", status, output)
	}
	var reply stateReply
	if err := json.Unmarshal([]byte(output), &reply); err != nil || strings.Count(strings.TrimSpace(output), "\n") != 0 {
		t.Fatalf("state must be one JSON line, got %q (%v)", output, err)
	}
	return reply
}

func TestStateOfFreshInstallIsEmptyButComplete(t *testing.T) {
	_, appDir, _ := deviceRoot(t)
	reply := readState(t, appDir, filepath.Join(appDir, "none.txt"), filepath.Join(appDir, "none.pid"))
	if reply.Mode != "fill" || reply.Rotation != "fixed" || reply.Status != "" || reply.SleepScreenHint != enSleepHint ||
		reply.Count != 0 || reply.Summary != "" || reply.Pictures == nil || len(reply.Pictures) != 0 || reply.SleepScreenLine != enSleepHint ||
		reply.SleepScreenNotice != "Sleep screen off · See Options" {
		t.Fatalf("state = %+v", reply)
	}
}

func TestStateAfterImportNamesPictureAndReadsLastResult(t *testing.T) {
	_, appDir, versionFile := deviceRoot(t)
	if status, output := run(t, "import", "--app-dir", appDir, "--locale", "en", "--version-file", versionFile); status != 0 {
		t.Fatalf("import: %s", output)
	}
	lastFile := filepath.Join(appDir, "last.txt")
	write(t, lastFile, []byte("New pictures: 1. Already known: 0.\nPictures: 1\n"))
	pidFile := filepath.Join(appDir, "run.pid")
	write(t, pidFile, []byte(strconv.Itoa(os.Getpid())))
	reply := readState(t, appDir, lastFile, pidFile)
	if reply.Count != 1 || reply.CurrentID != "0001" || !reply.Busy || reply.Status != "Converting…" {
		t.Fatalf("state with a live pid = %+v", reply)
	}
	write(t, pidFile, []byte("2147483646"))
	if reply := readState(t, appDir, lastFile, pidFile); reply.Busy || reply.Status != "" {
		t.Fatalf("idle state (no header status) = %+v", reply)
	}
	if reply.LastImport != "New pictures: 1. Already known: 0." {
		t.Fatalf("last import = %q", reply.LastImport)
	}
}

func TestStateHidesHintWhenNickelSettingsAreOn(t *testing.T) {
	_, appDir, _ := deviceRoot(t)
	write(t, filepath.Join(appDir, "Kobo eReader.conf"), []byte("[PowerOptions]\nShowBookCover=true\nFullscreenCoverStretch=true\n"))
	reply := readState(t, appDir, filepath.Join(appDir, "none.txt"), filepath.Join(appDir, "none.pid"))
	if !reply.SleepScreenReady || reply.SleepScreenHint != "" || reply.SleepScreenNotice != "" {
		t.Fatalf("state = %+v, want ready with an empty hint", reply)
	}
}

func TestImportAndStatusTellWhichKoboSettingsToTick(t *testing.T) {
	_, appDir, versionFile := deviceRoot(t)
	conf := filepath.Join(appDir, "Kobo eReader.conf")
	write(t, conf, []byte("[PowerOptions]\nShowBookCover=true\nFullscreenCoverStretch=false\n"))
	common := []string{"--app-dir", appDir, "--locale", "en", "--version-file", versionFile, "--conf", conf}
	for _, command := range []string{"import", "status"} {
		if status, output := run(t, append([]string{command}, common...)...); status != 0 || !strings.Contains(output, enSleepHint) {
			t.Fatalf("%s exit %d, want the hint in %q", command, status, output)
		}
	}
	write(t, conf, []byte("[PowerOptions]\nShowBookCover=true\nFullscreenCoverStretch=true\n"))
	if _, output := run(t, append([]string{"status"}, common...)...); strings.Contains(output, "Turn on in Settings") {
		t.Fatalf("status shows the hint although both settings are on: %q", output)
	}
}

func TestFirstImportDueExitCodes(t *testing.T) {
	_, appDir, _ := deviceRoot(t)
	common := []string{"--app-dir", appDir, "--locale", "en"}
	if status, output := run(t, append([]string{"first-import-due"}, common...)...); status != 0 || output != "" {
		t.Fatalf("first call: status %d, output %q; want 0 and silent", status, output)
	}
	if status, output := run(t, append([]string{"first-import-due"}, common...)...); status != 1 || output != "" {
		t.Fatalf("second call: status %d, output %q; want 1 and silent", status, output)
	}
}

func TestFirstImportDueReportsFailureWithExitTwo(t *testing.T) {
	_, appDir, _ := deviceRoot(t)
	write(t, filepath.Join(appDir, "data", "images", "index.json"), []byte("not json"))
	status, output := run(t, "first-import-due", "--app-dir", appDir, "--locale", "en")
	if status != 2 || !strings.Contains(output, "index.json") {
		t.Fatalf("status %d, output %q; want 2 and the index path", status, output)
	}
}
