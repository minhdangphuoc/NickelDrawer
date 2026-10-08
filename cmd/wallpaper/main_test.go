package main

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
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
	write(t, filepath.Join(root, "Pictures", "Wallpapers", "a.png"), buffer.Bytes())
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
